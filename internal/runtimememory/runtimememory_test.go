package runtimememory

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
)

func TestBuildSelectsRelevantProjectAndGlobalMemories(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	entries := []*models.MemoryEntry{
		{Title: "Runtime queue decision", Layer: models.MemoryLayerProject, Category: "decision", Content: "Use the runtime queue pattern for prompt injection jobs.", Tags: []string{"runtime", "queue"}, CreatedAt: now, UpdatedAt: now},
		{Title: "Global OpenCode warning", Layer: models.MemoryLayerGlobal, Category: "warning", Content: "OpenCode prompt hooks must stay bounded to avoid prompt bloat.", Tags: []string{"opencode", "runtime"}, CreatedAt: now, UpdatedAt: now.Add(-time.Hour)},
		{Title: "Unrelated preference", Layer: models.MemoryLayerProject, Category: "preference", Content: "Use playful colors in marketing pages.", Tags: []string{"design"}, CreatedAt: now, UpdatedAt: now.Add(-2 * time.Hour)},
	}
	for _, entry := range entries {
		if err := store.Memory.Create(entry); err != nil {
			t.Fatalf("create memory %q: %v", entry.Title, err)
		}
	}

	pack, err := Build(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "prompt_async",
		UserPrompt:  "implement runtime queue prompt injection for opencode",
		Mode:        ModeAuto,
		MaxItems:    5,
		MaxBytes:    2500,
	})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}
	if pack.Status != StatusCandidate {
		t.Fatalf("status = %q, want %q", pack.Status, StatusCandidate)
	}
	if len(pack.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(pack.Items))
	}
	if pack.Items[0].Title != "Runtime queue decision" {
		t.Fatalf("first item = %q, want runtime queue decision", pack.Items[0].Title)
	}
	if pack.Items[1].Layer != models.MemoryLayerGlobal {
		t.Fatalf("second layer = %q, want global", pack.Items[1].Layer)
	}
	if pack.Items[0].Category == "" || pack.Items[0].Layer == "" || pack.Items[0].UpdatedAt.IsZero() {
		t.Fatalf("missing provenance in first item: %+v", pack.Items[0])
	}
	if pack.Serialized == "" {
		t.Fatal("expected serialized payload")
	}
}

func TestBuildReturnsNoneWhenNoRelevantMemoryExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	entry := &models.MemoryEntry{Title: "UI preference", Layer: models.MemoryLayerProject, Category: "preference", Content: "Prefer serif typography for landing pages.", Tags: []string{"design"}, CreatedAt: now, UpdatedAt: now}
	if err := store.Memory.Create(entry); err != nil {
		t.Fatalf("create memory: %v", err)
	}

	pack, err := Build(store, Input{Runtime: "opencode", ProjectRoot: projectRoot, WorkingDir: projectRoot, ActionType: "prompt_async", UserPrompt: "debug sqlite vector search", Mode: ModeAuto})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}
	if pack.Status != StatusNone {
		t.Fatalf("status = %q, want %q", pack.Status, StatusNone)
	}
	if len(pack.Items) != 0 {
		t.Fatalf("items = %d, want 0", len(pack.Items))
	}
}

func TestBuildHonorsItemAndByteLimits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		entry := &models.MemoryEntry{
			Title:     "Runtime injection pattern",
			Layer:     models.MemoryLayerProject,
			Category:  "pattern",
			Content:   "Runtime injection should stay bounded while carrying prompt context and ranking reasons for repeated prompt execution. This text is intentionally long to force truncation.",
			Tags:      []string{"runtime", "prompt"},
			CreatedAt: now,
			UpdatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		if err := store.Memory.Create(entry); err != nil {
			t.Fatalf("create memory %d: %v", i, err)
		}
	}

	pack, err := Build(store, Input{Runtime: "opencode", ProjectRoot: projectRoot, WorkingDir: projectRoot, ActionType: "prompt_async", UserPrompt: "runtime prompt injection", Mode: ModeAuto, MaxItems: 1, MaxBytes: 260})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}
	if len(pack.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(pack.Items))
	}
	if pack.Bytes > 260 {
		t.Fatalf("bytes = %d, want <= 260", pack.Bytes)
	}
}

func TestLookupAdapterIncludesRequiredRuntimesAndModes(t *testing.T) {
	for _, runtime := range []string{"kiro", "claude-code", "opencode", "antigravity"} {
		adapter, ok := LookupAdapter(runtime)
		if !ok {
			t.Fatalf("missing adapter %q", runtime)
		}
		if len(adapter.SupportedModes) != 4 {
			t.Fatalf("adapter %q modes = %v", runtime, adapter.SupportedModes)
		}
	}
	kiro, _ := LookupAdapter("kiro")
	if !kiro.NativeHooks || kiro.HookKind != HookNative {
		t.Fatalf("kiro adapter = %+v, want native hooks", kiro)
	}
}
