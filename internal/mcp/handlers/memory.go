package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterMemoryTools registers persistent memory tools (project + global layers).
func RegisterMemoryTools(s *server.MCPServer, getStore func() *storage.Store) {

	// ── add_memory ──────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("add_memory",
			mcp.WithDescription("Create a memory entry. Stores knowledge that persists across sessions."),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("Memory content (markdown)"),
			),
			mcp.WithString("title",
				mcp.Description("Memory title"),
			),
			mcp.WithString("layer",
				mcp.Description("Memory layer: 'project' (default) or 'global'"),
			),
			mcp.WithString("category",
				mcp.Description("Category: pattern, decision, convention, preference, etc."),
			),
			mcp.WithArray("tags",
				mcp.Description("Tags for the memory entry"),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			content, err := req.RequireString("content")
			if err != nil {
				return errResult("content is required")
			}

			args := req.GetArguments()
			title, _ := stringArg(args, "title")
			layer, _ := stringArg(args, "layer")
			category, _ := stringArg(args, "category")
			tags, _ := stringSliceArg(args, "tags")

			if layer == "" {
				layer = models.MemoryLayerProject
			}
			if !models.ValidMemoryLayer(layer) || layer == models.MemoryLayerWorking {
				return errResult("layer must be 'project' or 'global' (use add_working_memory for working layer)")
			}

			now := time.Now().UTC()
			entry := &models.MemoryEntry{
				Title:     title,
				Layer:     layer,
				Category:  category,
				Content:   content,
				Tags:      tags,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if entry.Tags == nil {
				entry.Tags = []string{}
			}

			if err := store.Memory.Create(entry); err != nil {
				return errFailed("create memory", err)
			}

			search.BestEffortIndexMemory(store, entry.ID)
			go notifyServer(store, "notify/refresh")

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── get_memory ──────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("get_memory",
			mcp.WithDescription("Get a memory entry by ID."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Memory entry ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			id, err := req.RequireString("id")
			if err != nil {
				return errResult("id is required")
			}

			entry, err := store.Memory.Get(id)
			if err != nil {
				return errNotFound("memory", err)
			}

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── list_memories ───────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("list_memories",
			mcp.WithDescription("List memory entries with optional filters."),
			mcp.WithString("layer",
				mcp.Description("Filter by layer: working, project, global"),
			),
			mcp.WithString("category",
				mcp.Description("Filter by category"),
			),
			mcp.WithString("tag",
				mcp.Description("Filter by tag"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			args := req.GetArguments()
			layer, _ := stringArg(args, "layer")
			category, _ := stringArg(args, "category")
			tag, _ := stringArg(args, "tag")

			entries, err := store.Memory.List(layer)
			if err != nil {
				return errFailed("list memories", err)
			}

			// Apply filters.
			if category != "" {
				filtered := entries[:0]
				for _, e := range entries {
					if e.Category == category {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}
			if tag != "" {
				filtered := entries[:0]
				for _, e := range entries {
					for _, t := range e.Tags {
						if t == tag {
							filtered = append(filtered, e)
							break
						}
					}
				}
				entries = filtered
			}

			// Build summary (don't include full content in list).
			type memorySummary struct {
				ID        string   `json:"id"`
				Title     string   `json:"title"`
				Layer     string   `json:"layer"`
				Category  string   `json:"category,omitempty"`
				Tags      []string `json:"tags,omitempty"`
				CreatedAt string   `json:"createdAt"`
				UpdatedAt string   `json:"updatedAt"`
			}
			summaries := make([]memorySummary, len(entries))
			for i, e := range entries {
				summaries[i] = memorySummary{
					ID:        e.ID,
					Title:     e.Title,
					Layer:     e.Layer,
					Category:  e.Category,
					Tags:      e.Tags,
					CreatedAt: e.CreatedAt.Format(time.RFC3339),
					UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
				}
			}

			out, _ := json.MarshalIndent(summaries, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── update_memory ───────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("update_memory",
			mcp.WithDescription("Update a memory entry."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Memory entry ID"),
			),
			mcp.WithString("title",
				mcp.Description("New title"),
			),
			mcp.WithString("content",
				mcp.Description("New content (replaces existing)"),
			),
			mcp.WithString("category",
				mcp.Description("New category"),
			),
			mcp.WithArray("tags",
				mcp.Description("New tags (replaces existing)"),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			id, err := req.RequireString("id")
			if err != nil {
				return errResult("id is required")
			}

			entry, err := store.Memory.Get(id)
			if err != nil {
				return errNotFound("memory", err)
			}

			args := req.GetArguments()
			if v, ok := stringArg(args, "title"); ok {
				entry.Title = v
			}
			if v, ok := stringArg(args, "content"); ok {
				entry.Content = v
			}
			if v, ok := stringArg(args, "category"); ok {
				entry.Category = v
			}
			if v, ok := stringSliceArg(args, "tags"); ok {
				entry.Tags = v
			}

			entry.UpdatedAt = time.Now().UTC()

			if err := store.Memory.Update(entry); err != nil {
				return errFailed("update memory", err)
			}

			search.BestEffortIndexMemory(store, entry.ID)
			go notifyServer(store, "notify/refresh")

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── delete_memory ───────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("delete_memory",
			mcp.WithDescription("Delete a memory entry. Runs in dry-run mode by default."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Memory entry ID"),
			),
			mcp.WithBoolean("dryRun",
				mcp.Description("Preview only without deleting (default: true)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			id, err := req.RequireString("id")
			if err != nil {
				return errResult("id is required")
			}

			entry, err := store.Memory.Get(id)
			if err != nil {
				return errNotFound("memory", err)
			}

			args := req.GetArguments()
			dryRun := true
			if v, ok := args["dryRun"]; ok {
				if b, ok := v.(bool); ok {
					dryRun = b
				}
			}

			if dryRun {
				result := map[string]any{
					"dryRun":  true,
					"message": fmt.Sprintf("Would delete memory: %s (%s, layer: %s)", entry.ID, entry.Title, entry.Layer),
				}
				out, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if err := store.Memory.Delete(id); err != nil {
				return errFailed("delete memory", err)
			}

			search.BestEffortRemoveMemory(store, id)
			go notifyServer(store, "notify/refresh")

			result := map[string]any{
				"deleted": true,
				"message": fmt.Sprintf("Deleted memory: %s (%s)", entry.ID, entry.Title),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── promote_memory ──────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("promote_memory",
			mcp.WithDescription("Promote a memory entry up one layer (working→project→global)."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Memory entry ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			id, err := req.RequireString("id")
			if err != nil {
				return errResult("id is required")
			}

			entry, err := store.Memory.Promote(id)
			if err != nil {
				return errResult(err.Error())
			}

			search.BestEffortIndexMemory(store, entry.ID)
			go notifyServer(store, "notify/refresh")

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── demote_memory ───────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("demote_memory",
			mcp.WithDescription("Demote a memory entry down one layer (global→project→working)."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("Memory entry ID"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			id, err := req.RequireString("id")
			if err != nil {
				return errResult("id is required")
			}

			entry, err := store.Memory.Demote(id)
			if err != nil {
				return errResult(err.Error())
			}

			search.BestEffortIndexMemory(store, entry.ID)
			go notifyServer(store, "notify/refresh")

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
