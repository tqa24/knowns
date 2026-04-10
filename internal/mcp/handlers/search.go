package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterSearchTools registers search and retrieval MCP tools.
func RegisterSearchTools(s *server.MCPServer, getStore func() *storage.Store) {
	// search
	s.AddTool(
		mcp.NewTool("search",
			mcp.WithDescription("Unified search across tasks, docs, and memories. Supports hybrid semantic search when enabled."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query"),
			),
			mcp.WithString("type",
				mcp.Description("Search type: all, task, doc, or memory"),
				mcp.Enum("all", "task", "doc", "memory"),
			),
			mcp.WithString("mode",
				mcp.Description("Search mode: hybrid (semantic + keyword), semantic only, or keyword only (default: hybrid)"),
				mcp.Enum("hybrid", "semantic", "keyword"),
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
			mcp.WithNumber("limit",
				mcp.Description("Limit results (default: 20)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			mode := ""
			if v, ok := stringArg(args, "mode"); ok {
				mode = v
			}
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

			out, _ := json.MarshalIndent(results, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// reindex_search
	s.AddTool(
		mcp.NewTool("reindex_search",
			mcp.WithDescription("Rebuild the semantic search index from all tasks, docs, and memories. Use when index is out of sync or after enabling semantic search."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			taskCount := 0
			docCount := 0
			memoryCount := 0

			tasks, err := store.Tasks.List()
			if err == nil {
				taskCount = len(tasks)
			}
			docs, err := store.Docs.List()
			if err == nil {
				docCount = len(docs)
			}
			memories, err := store.Memory.List("")
			if err == nil {
				memoryCount = len(memories)
			}

			embedder, vecStore, initErr := search.InitSemantic(store)
			if embedder != nil {
				defer embedder.Close()
			}
			if vecStore != nil {
				defer vecStore.Close()
			}

			if initErr != nil || embedder == nil || vecStore == nil {
				result := map[string]any{
					"success":     true,
					"taskCount":   taskCount,
					"docCount":    docCount,
					"memoryCount": memoryCount,
					"mode":        "keyword",
					"message":     fmt.Sprintf("Semantic search unavailable. Keyword search will scan %d tasks, %d docs, and %d memories directly.", taskCount, docCount, memoryCount),
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			engine := search.NewEngine(store, embedder, vecStore)
			if err := engine.Reindex(nil); err != nil {
				return errFailed("reindex", err)
			}
			chunkCount := vecStore.Count()

			result := map[string]any{
				"success":     true,
				"taskCount":   taskCount,
				"docCount":    docCount,
				"memoryCount": memoryCount,
				"chunkCount":  chunkCount,
				"mode":        "semantic",
				"message":     fmt.Sprintf("Index rebuilt: %d tasks, %d docs, %d memories, %d chunks", taskCount, docCount, memoryCount, chunkCount),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// retrieve
	s.AddTool(
		mcp.NewTool("retrieve",
			mcp.WithDescription("Mixed-source retrieval across docs, tasks, and memories. Returns ranked candidates and an assembled context pack."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Retrieval query"),
			),
			mcp.WithString("mode",
				mcp.Description("Retrieval mode: hybrid (semantic + keyword), semantic only, or keyword only (default: hybrid)"),
				mcp.Enum("hybrid", "semantic", "keyword"),
			),
			mcp.WithArray("sourceTypes",
				mcp.Description("Optional source types: doc, task, memory"),
			),
			mcp.WithBoolean("expandReferences",
				mcp.Description("Whether to include linked docs/tasks/memories as expanded context"),
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
			mcp.WithNumber("limit",
				mcp.Description("Limit ranked candidates (default: 20)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			expandRefs := searchBoolArg(args, "expandReferences")
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
		},
	)
}

func searchBoolArg(args map[string]interface{}, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
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
