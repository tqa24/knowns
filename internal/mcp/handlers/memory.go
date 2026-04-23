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

// RegisterMemoryTool registers the consolidated persistent memory MCP tool.
func RegisterMemoryTool(s *server.MCPServer, getStore func() *storage.Store) {
	s.AddTool(
		mcp.NewTool("memory",
			mcp.WithDescription("Persistent memory operations. Use 'action' to specify: add, get, update, delete, list, promote, demote."),
			mcp.WithString("action",
				mcp.Required(),
				mcp.Description("Action to perform"),
				mcp.Enum("add", "get", "update", "delete", "list", "promote", "demote"),
			),
			mcp.WithString("id",
				mcp.Description("Memory entry ID (required for get, update, delete, promote, demote)"),
			),
			mcp.WithString("content",
				mcp.Description("Memory content in markdown (required for add)"),
			),
			mcp.WithString("title",
				mcp.Description("Memory title (add, update)"),
			),
			mcp.WithString("layer",
				mcp.Description("Memory layer: 'project' (default) or 'global' (add, list)"),
			),
			mcp.WithString("category",
				mcp.Description("Category: pattern, decision, convention, preference, etc. (add, update, list)"),
			),
			mcp.WithArray("tags",
				mcp.Description("Tags for the memory entry (add, update)"),
				mcp.WithStringItems(),
			),
			mcp.WithString("tag",
				mcp.Description("Filter by tag (list)"),
			),
			mcp.WithBoolean("dryRun",
				mcp.Description("Preview only without deleting (default: true) (delete)"),
			),
			mcp.WithArray("clear",
				mcp.Description("Explicitly clear string fields like title, content, or category (update)"),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action, err := req.RequireString("action")
			if err != nil {
				return errResult("action is required")
			}
			switch action {
			case "add":
				return handleMemoryAdd(getStore, req)
			case "get":
				return handleMemoryGet(getStore, req)
			case "update":
				return handleMemoryUpdate(getStore, req)
			case "delete":
				return handleMemoryDelete(getStore, req)
			case "list":
				return handleMemoryList(getStore, req)
			case "promote":
				return handleMemoryPromote(getStore, req)
			case "demote":
				return handleMemoryDemote(getStore, req)
			default:
				return errResultf("unknown memory action: %s", action)
			}
		},
	)
}

func handleMemoryAdd(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if !models.ValidPersistentMemoryLayer(layer) {
		return errResult("layer must be 'project' or 'global'")
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
}

func handleMemoryGet(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	id, err := req.RequireString("id")
	if err != nil {
		return errResult("id is required")
	}

	entry, err := store.Memory.Get(id)
	if err != nil || !models.ValidPersistentMemoryLayer(entry.Layer) {
		return errNotFound("memory", fmt.Errorf("memory %q not found", id))
	}

	out, _ := json.MarshalIndent(entry, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleMemoryList(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	args := req.GetArguments()
	layer, _ := stringArg(args, "layer")
	category, _ := stringArg(args, "category")
	tag, _ := stringArg(args, "tag")

	if layer != "" && !models.ValidPersistentMemoryLayer(layer) {
		return errResult("layer must be 'project' or 'global'")
	}

	entries, err := store.Memory.ListPersistent(layer)
	if err != nil {
		return errFailed("list memories", err)
	}

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
}

func handleMemoryUpdate(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	id, err := req.RequireString("id")
	if err != nil {
		return errResult("id is required")
	}

	entry, err := store.Memory.Get(id)
	if err != nil || !models.ValidPersistentMemoryLayer(entry.Layer) {
		return errNotFound("memory", fmt.Errorf("memory %q not found", id))
	}

	args := req.GetArguments()
	clearFields := stringSetArg(args, "clear")
	if clearFields["title"] {
		entry.Title = ""
	} else if v, ok := stringArg(args, "title"); ok && v != "" {
		entry.Title = v
	}
	if clearFields["content"] {
		entry.Content = ""
	} else if v, ok := stringArg(args, "content"); ok && v != "" {
		entry.Content = v
	}
	if clearFields["category"] {
		entry.Category = ""
	} else if v, ok := stringArg(args, "category"); ok && v != "" {
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
}

func handleMemoryDelete(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	id, err := req.RequireString("id")
	if err != nil {
		return errResult("id is required")
	}

	entry, err := store.Memory.Get(id)
	if err != nil || !models.ValidPersistentMemoryLayer(entry.Layer) {
		return errNotFound("memory", fmt.Errorf("memory %q not found", id))
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
}

func handleMemoryPromote(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	id, err := req.RequireString("id")
	if err != nil {
		return errResult("id is required")
	}

	entry, err := store.Memory.PromotePersistent(id)
	if err != nil {
		return errResult(err.Error())
	}

	search.BestEffortIndexMemory(store, entry.ID)
	go notifyServer(store, "notify/refresh")

	out, _ := json.MarshalIndent(entry, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleMemoryDemote(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	id, err := req.RequireString("id")
	if err != nil {
		return errResult("id is required")
	}

	entry, err := store.Memory.DemotePersistent(id)
	if err != nil {
		return errResult(err.Error())
	}

	search.BestEffortIndexMemory(store, entry.ID)
	go notifyServer(store, "notify/refresh")

	out, _ := json.MarshalIndent(entry, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}
