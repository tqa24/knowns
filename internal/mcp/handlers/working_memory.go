package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// WorkingMemoryStore is a thread-safe in-memory store for session-scoped working memory.
type WorkingMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*models.MemoryEntry
}

// NewWorkingMemoryStore creates a new empty working memory store.
func NewWorkingMemoryStore() *WorkingMemoryStore {
	return &WorkingMemoryStore{
		entries: make(map[string]*models.MemoryEntry),
	}
}

func (w *WorkingMemoryStore) Add(entry *models.MemoryEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if entry.ID == "" {
		entry.ID = models.NewTaskID()
	}
	entry.Layer = models.MemoryLayerWorking
	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	w.entries[entry.ID] = entry
}

func (w *WorkingMemoryStore) Get(id string) (*models.MemoryEntry, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	e, ok := w.entries[id]
	return e, ok
}

func (w *WorkingMemoryStore) List() []*models.MemoryEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]*models.MemoryEntry, 0, len(w.entries))
	for _, e := range w.entries {
		result = append(result, e)
	}
	return result
}

func (w *WorkingMemoryStore) Delete(id string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.entries[id]
	if ok {
		delete(w.entries, id)
	}
	return ok
}

func (w *WorkingMemoryStore) Clear() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	count := len(w.entries)
	w.entries = make(map[string]*models.MemoryEntry)
	return count
}

// RegisterWorkingMemoryTools registers session-scoped working memory tools.
// getStore returns the storage store for persistent working memory.
// getWM returns the in-memory store (used for session-scoped caching).
func RegisterWorkingMemoryTools(s *server.MCPServer, getStore func() *storage.Store, getWM func() *WorkingMemoryStore) {

	// ── add_working_memory ──────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("add_working_memory",
			mcp.WithDescription("Add a working memory entry (persistent, survives restarts)."),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("Memory content"),
			),
			mcp.WithString("title",
				mcp.Description("Memory title"),
			),
			mcp.WithString("category",
				mcp.Description("Category"),
			),
			mcp.WithArray("tags",
				mcp.Description("Tags"),
				mcp.WithStringItems(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}
			wm := getWM()

			content, err := req.RequireString("content")
			if err != nil {
				return errResult("content is required")
			}

			args := req.GetArguments()
			title, _ := stringArg(args, "title")
			category, _ := stringArg(args, "category")
			tags, _ := stringSliceArg(args, "tags")

			entry := &models.MemoryEntry{
				Title:    title,
				Layer:    models.MemoryLayerWorking,
				Category: category,
				Content:  content,
				Tags:     tags,
			}
			if entry.Tags == nil {
				entry.Tags = []string{}
			}

			// Save to persistent storage.
			if err := store.Memory.Create(entry); err != nil {
				return errFailed("create working memory", err)
			}

			// Also add to in-memory cache.
			wm.Add(entry)

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── get_working_memory ──────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("get_working_memory",
			mcp.WithDescription("Get a working memory entry by ID."),
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
				return errResult("working memory entry not found: " + id)
			}

			out, _ := json.MarshalIndent(entry, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── list_working_memories ───────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("list_working_memories",
			mcp.WithDescription("List all working memory entries."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}

			entries, err := store.Memory.List(models.MemoryLayerWorking)
			if err != nil {
				return errFailed("list working memories", err)
			}
			if entries == nil {
				entries = []*models.MemoryEntry{}
			}

			out, _ := json.MarshalIndent(entries, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── delete_working_memory ───────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("delete_working_memory",
			mcp.WithDescription("Delete a working memory entry."),
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
			wm := getWM()

			id, err := req.RequireString("id")
			if err != nil {
				return errResult("id is required")
			}

			if err := store.Memory.Delete(id); err != nil {
				return errResult("working memory entry not found: " + id)
			}

			// Also remove from in-memory cache.
			wm.Delete(id)

			result := map[string]any{
				"deleted": true,
				"id":      id,
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// ── clear_working_memory ────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("clear_working_memory",
			mcp.WithDescription("Clear all working memory entries."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return noProjectError()
			}
			wm := getWM()

			count, err := store.Memory.Clean()
			if err != nil {
				return errFailed("clear working memories", err)
			}

			// Also clear in-memory cache.
			wm.Clear()

			result := map[string]any{
				"cleared": count,
				"message": fmt.Sprintf("Cleared %d working memory entries.", count),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
