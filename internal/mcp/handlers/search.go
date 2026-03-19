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

// RegisterSearchTools registers search and reindex MCP tools.
func RegisterSearchTools(s *server.MCPServer, getStore func() *storage.Store) {
	// search
	s.AddTool(
		mcp.NewTool("search",
			mcp.WithDescription("Unified search across tasks and docs. Supports hybrid semantic search when enabled."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Search query"),
			),
			mcp.WithString("type",
				mcp.Description("Search type: all, task, or doc"),
				mcp.Enum("all", "task", "doc"),
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
				mcp.Description("Filter docs by tag"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Limit results (default: 20)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			query, err := req.RequireString("query")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
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

			// Initialize semantic search (embedder + SQLite vector store).
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
				return mcp.NewToolResultError(err.Error()), nil
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
			mcp.WithDescription("Rebuild the search index from all tasks and docs. Use when index is out of sync or after enabling semantic search."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			// Initialize semantic search.
			embedder, vecStore, initErr := search.InitSemantic(store)
			if embedder != nil {
				defer embedder.Close()
			}
			if vecStore != nil {
				defer vecStore.Close()
			}

			taskCount := 0
			docCount := 0

			tasks, err := store.Tasks.List()
			if err == nil {
				taskCount = len(tasks)
			}
			docs, err := store.Docs.List()
			if err == nil {
				docCount = len(docs)
			}

			if initErr != nil {
				// Keyword-only mode — no index to rebuild.
				result := map[string]any{
					"success":   true,
					"taskCount": taskCount,
					"docCount":  docCount,
					"mode":      "keyword",
					"message":   fmt.Sprintf("Keyword search active (%d tasks, %d docs). Install ONNX Runtime for semantic search with reindexable vector index.", taskCount, docCount),
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			engine := search.NewEngine(store, embedder, vecStore)
			if err := engine.Reindex(nil); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Reindex failed: %v", err)), nil
			}
			chunkCount := vecStore.Count()

			result := map[string]any{
				"success":    true,
				"taskCount":  taskCount,
				"docCount":   docCount,
				"chunkCount": chunkCount,
				"mode":       "semantic",
				"message":    fmt.Sprintf("Index rebuilt: %d tasks, %d docs, %d chunks", taskCount, docCount, chunkCount),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
