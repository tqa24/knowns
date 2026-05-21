package handlers

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMemoryCleanupDefaultsReturnStaleMemories(t *testing.T) {
	store := setupMemoryCleanupStore(t)
	now := time.Now().UTC()
	createMemoryForCleanupTest(t, store, "old001", models.MemoryLayerProject, now.AddDate(0, 0, -10), now.AddDate(0, 0, -10))
	createMemoryForCleanupTest(t, store, "new001", models.MemoryLayerProject, now.AddDate(0, 0, -1), now.AddDate(0, 0, -1))

	candidates := callMemoryCleanup(t, store, map[string]any{"action": "cleanup"})
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %+v", len(candidates), candidates)
	}
	if candidates[0].ID != "old001" || candidates[0].Content == "" || candidates[0].AgeDays < 9 {
		t.Fatalf("unexpected candidate: %+v", candidates[0])
	}
}

func TestMemoryCleanupCustomThresholdLayerLimitAndSorting(t *testing.T) {
	store := setupMemoryCleanupStore(t)
	now := time.Now().UTC()
	createMemoryForCleanupTest(t, store, "proj01", models.MemoryLayerProject, now.AddDate(0, 0, -40), now.AddDate(0, 0, -40))
	createMemoryForCleanupTest(t, store, "glob01", models.MemoryLayerGlobal, now.AddDate(0, 0, -30), now.AddDate(0, 0, -30))
	createMemoryForCleanupTest(t, store, "glob02", models.MemoryLayerGlobal, now.AddDate(0, 0, -50), now.AddDate(0, 0, -50))
	createMemoryForCleanupTest(t, store, "glob03", models.MemoryLayerGlobal, now.AddDate(0, 0, -10), now.AddDate(0, 0, -10))

	candidates := callMemoryCleanup(t, store, map[string]any{
		"action":        "cleanup",
		"layer":         models.MemoryLayerGlobal,
		"olderThanDays": 14,
		"limit":         1,
	})
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ID != "glob02" {
		t.Fatalf("expected oldest global memory first, got %+v", candidates[0])
	}
}

func TestMemoryCleanupEmptyWhenNoStaleMemories(t *testing.T) {
	store := setupMemoryCleanupStore(t)
	now := time.Now().UTC()
	createMemoryForCleanupTest(t, store, "new001", models.MemoryLayerProject, now.AddDate(0, 0, -1), now.AddDate(0, 0, -1))

	candidates := callMemoryCleanup(t, store, map[string]any{"action": "cleanup"})
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %+v", candidates)
	}
}

func TestMemoryUpdateTouchRemovesEntryFromCleanup(t *testing.T) {
	store := setupMemoryCleanupStore(t)
	now := time.Now().UTC()
	createMemoryForCleanupTest(t, store, "old001", models.MemoryLayerProject, now.AddDate(0, 0, -10), now.AddDate(0, 0, -10))

	before, err := store.Memory.Get("old001")
	if err != nil {
		t.Fatalf("get before: %v", err)
	}
	updated, err := handleMemoryUpdate(func() *storage.Store { return store }, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"action": "update", "id": "old001"}}})
	if err != nil || updated.IsError {
		t.Fatalf("update returned error: %v, result: %+v", err, updated)
	}
	after, err := store.Memory.Get("old001")
	if err != nil {
		t.Fatalf("get after: %v", err)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("expected updatedAt to advance, before=%s after=%s", before.UpdatedAt, after.UpdatedAt)
	}
	candidates := callMemoryCleanup(t, store, map[string]any{"action": "cleanup"})
	if len(candidates) != 0 {
		t.Fatalf("expected touched memory to be excluded, got %+v", candidates)
	}
}

func setupMemoryCleanupStore(t *testing.T) *storage.Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	store := storage.NewStore(filepath.Join(t.TempDir(), ".knowns"))
	if err := store.Init("memory-cleanup-test"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	return store
}

func createMemoryForCleanupTest(t *testing.T, store *storage.Store, id, layer string, createdAt, updatedAt time.Time) {
	t.Helper()
	if err := store.Memory.Create(&models.MemoryEntry{
		ID:        id,
		Title:     "Memory " + id,
		Layer:     layer,
		Category:  "pattern",
		Tags:      []string{"cleanup"},
		Content:   "Content for " + id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}); err != nil {
		t.Fatalf("create memory %s: %v", id, err)
	}
}

func callMemoryCleanup(t *testing.T, store *storage.Store, args map[string]any) []memoryCleanupCandidate {
	t.Helper()
	result, err := handleMemoryCleanup(func() *storage.Store { return store }, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	if err != nil || result.IsError {
		t.Fatalf("cleanup returned error: %v, result: %+v", err, result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected one content item, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	var candidates []memoryCleanupCandidate
	if err := json.Unmarshal([]byte(text.Text), &candidates); err != nil {
		t.Fatalf("unmarshal cleanup output: %v\n%s", err, text.Text)
	}
	return candidates
}
