package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterSearchTool registers the consolidated search and retrieval MCP tool.
// Note: reindex_search has been removed — reindexing is handled automatically
// or via the CLI command.
func RegisterSearchTool(s *server.MCPServer, getStore func() *storage.Store) {
	s.AddTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search and retrieval operations. Use 'action' to specify: search, retrieve, resolve."),
			mcp.WithString("action",
				mcp.Required(),
				mcp.Description("Action to perform"),
				mcp.Enum("search", "retrieve", "resolve"),
			),
			// search + retrieve params
			mcp.WithString("query",
				mcp.Description("Search/retrieval query (required for search, retrieve)"),
			),
			mcp.WithString("type",
				mcp.Description("Search type: all, task, doc, or memory (search)"),
				mcp.Enum("all", "task", "doc", "memory"),
			),
			mcp.WithString("mode",
				mcp.Description("Search mode: hybrid (semantic + keyword), semantic only, or keyword only (default: hybrid)"),
				mcp.Enum("hybrid", "semantic", "keyword"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Limit results (default: 20)"),
			),
			mcp.WithString("status",
				mcp.Description("Filter tasks by status"),
			),
			mcp.WithString("priority",
				mcp.Description("Filter tasks by priority"),
			),
			mcp.WithString("assignee",
				mcp.Description("Filter tasks by assignee"),
			),
			mcp.WithString("label",
				mcp.Description("Filter tasks by label"),
			),
			mcp.WithString("tag",
				mcp.Description("Filter docs or memories by tag"),
			),
			// retrieve-specific params
			mcp.WithArray("sourceTypes",
				mcp.Description("Optional source types: doc, task, memory (retrieve)"),
				mcp.WithStringEnumItems([]string{"doc", "task", "memory"}),
			),
			mcp.WithBoolean("expandReferences",
				mcp.Description("Whether to include linked docs/tasks/memories as expanded context (retrieve)"),
			),
			// resolve params
			mcp.WithString("ref",
				mcp.Description("Semantic reference expression, e.g. @doc/guides/setup{implements} (required for resolve)"),
			),
			// structural traversal params (resolve)
			mcp.WithString("direction",
				mcp.Description("Traversal direction from root entity: \"outbound\" (default), \"inbound\", or \"both\" (resolve)"),
				mcp.Enum("outbound", "inbound", "both"),
			),
			mcp.WithNumber("depth",
				mcp.Description("Max traversal hops, 1–3 (default: 1) (resolve)"),
			),
			mcp.WithString("relationTypes",
				mcp.Description("Filter by relation kinds, comma-separated (resolve)"),
			),
			mcp.WithString("entityTypes",
				mcp.Description("Filter result entities by kind, comma-separated (resolve)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action, err := req.RequireString("action")
			if err != nil {
				return errResult("action is required")
			}
			switch action {
			case "search":
				return handleSearch(getStore, req)
			case "retrieve":
				return handleRetrieve(getStore, req)
			case "resolve":
				return handleResolve(getStore, req)
			default:
				return errResultf("unknown search action: %s", action)
			}
		},
	)
}

func handleSearch(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	query, err := req.RequireString("query")
	if err != nil {
		return errResult(err.Error())
	}

	args := req.GetArguments()
	searchType := "all"
	if v, ok := stringArg(args, "type"); ok {
		searchType = v
	}
	mode, _ := stringArg(args, "mode")
	statusFilter, _ := stringArg(args, "status")
	priorityFilter, _ := stringArg(args, "priority")
	assigneeFilter, _ := stringArg(args, "assignee")
	labelFilter, _ := stringArg(args, "label")
	tagFilter, _ := stringArg(args, "tag")
	limit := 20
	if v, ok := intArg(args, "limit"); ok && v > 0 {
		limit = v
	}

	opts := search.SearchOptions{
		Query:    query,
		Type:     searchType,
		Mode:     mode,
		Status:   statusFilter,
		Priority: priorityFilter,
		Assignee: assigneeFilter,
		Label:    labelFilter,
		Tag:      tagFilter,
		Limit:    limit,
	}

	embedder, vecStore, _ := search.InitSemantic(store)
	if embedder != nil {
		defer embedder.Close()
	}
	if vecStore != nil {
		defer vecStore.Close()
	}

	engine := search.NewEngine(store, embedder, vecStore)
	results, err := engine.Search(opts)
	if err != nil {
		return errResult(err.Error())
	}
	if results == nil {
		results = []models.SearchResult{}
	}

	// If results are empty, include project context so the caller can verify
	// the correct project is active.
	if len(results) == 0 {
		wrapper := map[string]any{
			"results":      results,
			"_projectRoot": store.Root,
			"_hint":        "Search returned 0 results. Verify the active project is correct via project({ action: \"current\" }). Use project({ action: \"set\" }) to switch if needed.",
		}
		out, _ := json.MarshalIndent(wrapper, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	}

	out, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleRetrieve(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	query, err := req.RequireString("query")
	if err != nil {
		return errResult(err.Error())
	}

	args := req.GetArguments()
	mode, _ := stringArg(args, "mode")
	statusFilter, _ := stringArg(args, "status")
	priorityFilter, _ := stringArg(args, "priority")
	assigneeFilter, _ := stringArg(args, "assignee")
	labelFilter, _ := stringArg(args, "label")
	tagFilter, _ := stringArg(args, "tag")
	limit := 20
	if v, ok := intArg(args, "limit"); ok && v > 0 {
		limit = v
	}
	expandRefs := boolArg(args, "expandReferences")
	sourceTypes := stringArrayArg(args, "sourceTypes")

	embedder, vecStore, _ := search.InitSemantic(store)
	if embedder != nil {
		defer embedder.Close()
	}
	if vecStore != nil {
		defer vecStore.Close()
	}

	engine := search.NewEngine(store, embedder, vecStore)
	response, err := engine.Retrieve(models.RetrievalOptions{
		Query:            query,
		Mode:             mode,
		Limit:            limit,
		SourceTypes:      sourceTypes,
		ExpandReferences: expandRefs,
		Status:           statusFilter,
		Priority:         priorityFilter,
		Assignee:         assigneeFilter,
		Label:            labelFilter,
		Tag:              tagFilter,
	})
	if err != nil {
		return errResult(err.Error())
	}

	out, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleResolve(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	raw, err := req.RequireString("ref")
	if err != nil {
		return errResult(err.Error())
	}

	args := req.GetArguments()

	// Check for structural traversal params.
	params := models.StructuralParams{}
	if v, ok := stringArg(args, "direction"); ok {
		params.Direction = v
	}
	if v, ok := intArg(args, "depth"); ok {
		params.Depth = v
	}
	if v, ok := stringArg(args, "relationTypes"); ok && v != "" {
		params.RelationTypes = splitCommaSeparated(v)
	}
	if v, ok := stringArg(args, "entityTypes"); ok && v != "" {
		params.EntityTypes = splitCommaSeparated(v)
	}

	// If structural params are present, use structural traversal.
	if params.IsStructural() {
		result, err := store.StructuralResolve(raw, params)
		if err != nil {
			return errResult(err.Error())
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	}

	// Otherwise, use the existing simple resolution.
	out, err := resolveReferenceJSON(store, raw)
	if err != nil {
		return errResult(err.Error())
	}
	return mcp.NewToolResultText(out), nil
}

// splitCommaSeparated splits a comma-separated string into trimmed non-empty parts.
func splitCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func resolveReferenceJSON(store *storage.Store, raw string) (string, error) {
	resolution, err := store.ResolveRawReference(raw)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(resolution, "", "  ")
	return string(out), nil
}

func stringArrayArg(args map[string]interface{}, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			values = append(values, s)
		}
	}
	return values
}
