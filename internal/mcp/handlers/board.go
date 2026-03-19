package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterBoardTools registers the board view MCP tool.
func RegisterBoardTools(s *server.MCPServer, getStore func() *storage.Store) {
	// get_board
	s.AddTool(
		mcp.NewTool("get_board",
			mcp.WithDescription("Get the current board state with tasks grouped by status."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			tasks, err := store.Tasks.List()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list tasks: %s", err.Error())), nil
			}

			// Determine column order from project config.
			columns := models.DefaultStatuses()
			if proj, err := store.Config.Load(); err == nil {
				if len(proj.Settings.VisibleColumns) > 0 {
					columns = proj.Settings.VisibleColumns
				} else if len(proj.Settings.Statuses) > 0 {
					columns = proj.Settings.Statuses
				}
			}

			// Group tasks by status.
			type column struct {
				Status string         `json:"status"`
				Tasks  []*models.Task `json:"tasks"`
				Count  int            `json:"count"`
			}

			columnMap := make(map[string]*column)
			for _, status := range columns {
				columnMap[status] = &column{
					Status: status,
					Tasks:  []*models.Task{},
				}
			}

			// Track any tasks with statuses not in the column list.
			extraStatuses := make(map[string]bool)
			for _, t := range tasks {
				if _, ok := columnMap[t.Status]; !ok {
					extraStatuses[t.Status] = true
					if _, ok2 := columnMap[t.Status]; !ok2 {
						columnMap[t.Status] = &column{
							Status: t.Status,
							Tasks:  []*models.Task{},
						}
					}
				}
				columnMap[t.Status].Tasks = append(columnMap[t.Status].Tasks, t)
				columnMap[t.Status].Count++
			}

			// Build ordered column list.
			var ordered []*column
			for _, status := range columns {
				if c, ok := columnMap[status]; ok {
					ordered = append(ordered, c)
				}
			}
			// Append any extra statuses.
			for status := range extraStatuses {
				if c, ok := columnMap[status]; ok {
					ordered = append(ordered, c)
				}
			}

			totalCount := 0
			for _, c := range ordered {
				totalCount += c.Count
			}

			board := map[string]any{
				"columns": ordered,
				"total":   totalCount,
			}

			out, _ := json.MarshalIndent(board, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
