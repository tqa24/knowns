package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/server/routes"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterCodeTool registers the consolidated code intelligence MCP tool.
func RegisterCodeTool(s *server.MCPServer, getStore func() *storage.Store) {
	s.AddTool(
		mcp.NewTool("code",
			mcp.WithDescription("Code intelligence operations. Use 'action' to specify: search, symbols, deps, graph."),
			mcp.WithString("action",
				mcp.Required(),
				mcp.Description("Action to perform"),
				mcp.Enum("search", "symbols", "deps", "graph"),
			),
			mcp.WithString("query",
				mcp.Description("Search query (required for search)"),
			),
			mcp.WithString("mode",
				mcp.Description("Search mode: hybrid, semantic, or keyword (default: hybrid)"),
				mcp.Enum("hybrid", "semantic", "keyword"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Limit results (default: 10 for search, 100 for symbols, 200 for deps)"),
			),
			mcp.WithNumber("neighbors",
				mcp.Description("Max neighbors per match (default: 5) (search)"),
			),
			mcp.WithString("edgeTypes",
				mcp.Description("Optional comma-separated edge types to expand (search)"),
			),
			mcp.WithString("path",
				mcp.Description("Optional doc path filter (symbols)"),
			),
			mcp.WithString("kind",
				mcp.Description("Optional symbol kind filter (symbols)"),
			),
			mcp.WithString("type",
				mcp.Description("Optional edge type filter: calls, contains, has_method, imports, instantiates, implements, extends (deps)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action, err := req.RequireString("action")
			if err != nil {
				return errResult("action is required")
			}
			switch action {
			case "search":
				return handleCodeSearch(getStore, req)
			case "symbols":
				return handleCodeSymbols(getStore, req)
			case "deps":
				return handleCodeDeps(getStore, req)
			case "graph":
				return handleCodeGraph(getStore, req)
			default:
				return errResultf("unknown code action: %s", action)
			}
		},
	)
}

func handleCodeGraph(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}
	nodes, edges := routes.BuildCodeGraph(store)
	if nodes == nil {
		nodes = []routes.GraphNode{}
	}
	if edges == nil {
		edges = []routes.GraphEdge{}
	}
	out, _ := json.MarshalIndent(map[string]any{"nodes": nodes, "edges": edges}, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleCodeSymbols(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}
	db := store.SemanticDB()
	if db == nil {
		return mcp.NewToolResultText("[]"), nil
	}
	defer db.Close()

	args := req.GetArguments()
	pathFilter, _ := stringArg(args, "path")
	kindFilter, _ := stringArg(args, "kind")
	limit := 100
	if v, ok := intArg(args, "limit"); ok && v > 0 {
		limit = v
	}

	rows, err := db.Query(`SELECT id, doc_path, field, COALESCE(name, ''), COALESCE(signature, '') FROM chunks WHERE type = 'code' AND (? = '' OR doc_path = ?) AND (? = '' OR field = ?) ORDER BY doc_path, name, id LIMIT ?`, pathFilter, pathFilter, kindFilter, kindFilter, limit)
	if err != nil {
		return errFailed("list code symbols", err)
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, docPath, kind, name, signature string
		if err := rows.Scan(&id, &docPath, &kind, &name, &signature); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":        id,
			"path":      docPath,
			"kind":      kind,
			"name":      name,
			"signature": signature,
		})
	}

	out, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleCodeDeps(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}
	db := store.SemanticDB()
	if db == nil {
		return mcp.NewToolResultText("[]"), nil
	}
	defer db.Close()

	args := req.GetArguments()
	edgeType, _ := stringArg(args, "type")
	limit := 200
	if v, ok := intArg(args, "limit"); ok && v > 0 {
		limit = v
	}

	rows, err := db.Query(`SELECT from_id, to_id, edge_type, from_path, to_path, raw_target, resolution_status, resolution_confidence FROM code_edges WHERE (? = '' OR edge_type = ?) ORDER BY from_id, edge_type, to_id LIMIT ?`, edgeType, edgeType, limit)
	if err != nil {
		return errFailed("list code dependencies", err)
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var fromID, toID, kind, fromPath, toPath, rawTarget, status, confidence string
		if err := rows.Scan(&fromID, &toID, &kind, &fromPath, &toPath, &rawTarget, &status, &confidence); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"from":       fromID,
			"to":         toID,
			"type":       kind,
			"fromPath":   fromPath,
			"toPath":     toPath,
			"rawTarget":  rawTarget,
			"status":     status,
			"confidence": confidence,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["from"], items[i]["type"], items[i]["to"]) < fmt.Sprint(items[j]["from"], items[j]["type"], items[j]["to"])
	})

	out, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleCodeSearch(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	query, err := req.RequireString("query")
	if err != nil {
		return errResult(err.Error())
	}
	args := req.GetArguments()
	mode := "hybrid"
	if v, ok := stringArg(args, "mode"); ok && v != "" {
		mode = v
	}
	limit := 10
	if v, ok := intArg(args, "limit"); ok && v > 0 {
		limit = v
	}
	neighbors := 5
	if v, ok := intArg(args, "neighbors"); ok && v >= 0 {
		neighbors = v
	}
	edgeTypesCSV, _ := stringArg(args, "edgeTypes")

	embedder, vecStore, _ := search.InitSemantic(store)
	if embedder != nil {
		defer embedder.Close()
	}
	if vecStore != nil {
		defer vecStore.Close()
	}

	graph, err := search.SearchCodeWithNeighbors(store, embedder, vecStore, models.RetrievalOptions{
		Query: query,
		Mode:  mode,
		Limit: limit,
	}, splitCSV(edgeTypesCSV), neighbors)
	if err != nil {
		return errFailed("search code", err)
	}

	out, _ := json.MarshalIndent(graph, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
