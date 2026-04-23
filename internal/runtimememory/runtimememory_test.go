package runtimememory

import (
	"os"
	"path/filepath"
	"strings"
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
	if !strings.Contains(pack.Serialized, "Knowns Guidance") {
		t.Fatalf("expected guidance header, got %q", pack.Serialized)
	}
	if !strings.Contains(pack.Serialized, "memory({ action: \"list\" })") {
		t.Fatalf("expected memory action list hint, got %q", pack.Serialized)
	}
	if strings.Contains(pack.Serialized, entries[0].Title) || strings.Contains(pack.Serialized, entries[1].Title) {
		t.Fatalf("did not expect memory titles in serialized payload, got %q", pack.Serialized)
	}
	if strings.Contains(pack.Serialized, entries[0].Content) || strings.Contains(pack.Serialized, entries[1].Content) {
		t.Fatalf("did not expect full memory content in serialized payload, got %q", pack.Serialized)
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

func TestBuildSkipsLowSignalPrompts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	entry := &models.MemoryEntry{Title: "Runtime queue decision", Layer: models.MemoryLayerProject, Category: "decision", Content: "Use the runtime queue pattern for prompt injection jobs.", Tags: []string{"runtime", "queue"}, CreatedAt: now, UpdatedAt: now}
	if err := store.Memory.Create(entry); err != nil {
		t.Fatalf("create memory: %v", err)
	}

	for _, prompt := range []string{"hi", "ok", "continue", "thanks", "yes"} {
		pack, err := Build(store, Input{Runtime: "opencode", ProjectRoot: projectRoot, WorkingDir: projectRoot, ActionType: "prompt_async", UserPrompt: prompt, Mode: ModeAuto})
		if err != nil {
			t.Fatalf("build pack for %q: %v", prompt, err)
		}
		if pack.Status != StatusNone {
			t.Fatalf("status for %q = %q, want %q", prompt, pack.Status, StatusNone)
		}
		if len(pack.Items) != 0 {
			t.Fatalf("items for %q = %d, want 0", prompt, len(pack.Items))
		}
	}

	pack, err := Build(store, Input{Runtime: "opencode", ProjectRoot: projectRoot, WorkingDir: projectRoot, ActionType: "prompt_async", UserPrompt: "fix auth", Mode: ModeAuto})
	if err != nil {
		t.Fatalf("build technical short prompt: %v", err)
	}
	if pack.Status != StatusNone {
		t.Fatalf("status for technical short prompt = %q, want %q because no relevant auth memory exists", pack.Status, StatusNone)
	}
}

func TestBuildSkipsWeakSingleCandidate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	entry := &models.MemoryEntry{Title: "Minor graph note", Layer: models.MemoryLayerProject, Category: "decision", Content: "Graph page keeps code and knowledge presets on one page.", Tags: []string{"graph", "page"}, CreatedAt: now, UpdatedAt: now}
	if err := store.Memory.Create(entry); err != nil {
		t.Fatalf("create memory: %v", err)
	}

	pack, err := Build(store, Input{Runtime: "opencode", ProjectRoot: projectRoot, WorkingDir: projectRoot, ActionType: "prompt_async", UserPrompt: "graph page", Mode: ModeAuto})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}
	if pack.Status != StatusNone {
		t.Fatalf("status = %q, want %q for weak candidate", pack.Status, StatusNone)
	}
}

func TestSerializePrefixAddsSilentInstructionForOpenCode(t *testing.T) {
	prefix := serializePrefix("opencode")
	if !strings.Contains(prefix, "Silent supplemental context. Do not quote unless asked.") {
		t.Fatalf("expected silent supplemental instruction, got %q", prefix)
	}
	other := serializePrefix("claude-code")
	if strings.Contains(other, "Silent supplemental context. Do not quote unless asked.") {
		t.Fatalf("did not expect OpenCode-specific instruction for other runtimes, got %q", other)
	}
}

func TestBuildSessionBaselineIncludesKNOWNSSummary(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "KNOWNS.md"), []byte("# KNOWNS\n\n- Canonical guidance\n- Read this first\n"), 0644); err != nil {
		t.Fatalf("write KNOWNS.md: %v", err)
	}
	entry := &models.MemoryEntry{Title: "Response style", Layer: models.MemoryLayerProject, Category: "preference", Content: "Answer directly and keep formatting flat.", Tags: []string{"style", "preference"}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.Memory.Create(entry); err != nil {
		t.Fatalf("create memory: %v", err)
	}

	pack, err := Build(store, Input{Runtime: "claude-code", ProjectRoot: projectRoot, WorkingDir: projectRoot, ActionType: "session-start", Mode: ModeAuto})
	if err != nil {
		t.Fatalf("build baseline pack: %v", err)
	}
	if pack.Status != StatusCandidate {
		t.Fatalf("status = %q, want %q", pack.Status, StatusCandidate)
	}
	if !strings.Contains(pack.Serialized, "Read `KNOWNS.md` in the repository root") {
		t.Fatalf("expected KNOWNS instruction in baseline pack, got %q", pack.Serialized)
	}
	if !strings.Contains(pack.Serialized, "memory({ action: \"list\" })") {
		t.Fatalf("expected MCP memory hint in baseline pack, got %q", pack.Serialized)
	}
	if strings.Contains(pack.Serialized, "Canonical guidance") {
		t.Fatalf("did not expect inlined KNOWNS contents, got %q", pack.Serialized)
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

func TestBuildUsesHybridCandidatesWhenAvailable(t *testing.T) {
	t.Cleanup(func() {
		lookupHybridCandidates = defaultHybridCandidates
	})
	lookupHybridCandidates = func(store *storage.Store, input Input, limit int) ([]hybridCandidate, bool) {
		now := time.Now().UTC()
		return []hybridCandidate{
			{
				entry: &models.MemoryEntry{
					ID:        "runtime-hit",
					Title:     "Unified runtime adapter install",
					Layer:     models.MemoryLayerProject,
					Category:  "pattern",
					Content:   "Use the runtime install command and opencode plugin path for unified adapter setup.",
					Tags:      []string{"runtime", "opencode", "codex"},
					UpdatedAt: now,
				},
				score:     0.92,
				matchedBy: []string{"semantic", "keyword"},
			},
			{
				entry: &models.MemoryEntry{
					ID:        "recent-noise",
					Title:     "Windows npm packages must use win32 os",
					Layer:     models.MemoryLayerProject,
					Category:  "failure",
					Content:   "Windows npm binary packages must declare win32 os metadata.",
					Tags:      []string{"windows", "npm", "cli"},
					UpdatedAt: now.Add(-time.Minute),
				},
				score:     0.21,
				matchedBy: []string{"semantic"},
			},
		}, true
	}

	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}

	pack, err := Build(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "user-prompt-submit",
		UserPrompt:  "implement unified runtime adapter install for codex and opencode",
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
	if len(pack.Items) == 0 {
		t.Fatal("expected at least one hybrid-selected item")
	}
	if pack.Items[0].Title != "Unified runtime adapter install" {
		t.Fatalf("first item = %q, want unified runtime adapter install", pack.Items[0].Title)
	}
	if pack.Items[0].Retrieval != "hybrid" {
		t.Fatalf("retrieval = %q, want hybrid", pack.Items[0].Retrieval)
	}
	if !containsString(pack.Items[0].Reasons, "semantic-match") || !containsString(pack.Items[0].Reasons, "hybrid-retrieval") {
		t.Fatalf("expected hybrid reasons, got %v", pack.Items[0].Reasons)
	}
}

func TestBuildFallsBackWhenHybridUnavailable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() {
		lookupHybridCandidates = defaultHybridCandidates
	})
	lookupHybridCandidates = func(store *storage.Store, input Input, limit int) ([]hybridCandidate, bool) {
		return nil, false
	}

	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	entry := &models.MemoryEntry{
		Title:     "Runtime queue decision",
		Layer:     models.MemoryLayerProject,
		Category:  "decision",
		Content:   "Use the runtime queue pattern for prompt injection jobs.",
		Tags:      []string{"runtime", "queue"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Memory.Create(entry); err != nil {
		t.Fatalf("create memory: %v", err)
	}

	pack, err := Build(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "prompt_async",
		UserPrompt:  "implement runtime queue prompt injection for opencode",
		Mode:        ModeAuto,
	})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}
	if len(pack.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(pack.Items))
	}
	if pack.Items[0].Retrieval != "heuristic-fallback" {
		t.Fatalf("retrieval = %q, want heuristic-fallback", pack.Items[0].Retrieval)
	}
	if !containsString(pack.Items[0].Reasons, "heuristic-fallback") {
		t.Fatalf("expected fallback reason, got %v", pack.Items[0].Reasons)
	}
}

func TestBuildKeepsEmptyPackCleanWhenHybridReturnsNoUsableCandidates(t *testing.T) {
	t.Cleanup(func() {
		lookupHybridCandidates = defaultHybridCandidates
	})
	lookupHybridCandidates = func(store *storage.Store, input Input, limit int) ([]hybridCandidate, bool) {
		now := time.Now().UTC()
		return []hybridCandidate{{
			entry: &models.MemoryEntry{
				ID:        "noise",
				Title:     "Completely unrelated preference",
				Layer:     models.MemoryLayerProject,
				Category:  "preference",
				Content:   "Use playful colors in marketing pages.",
				Tags:      []string{"design"},
				UpdatedAt: now,
			},
			score:     0.05,
			matchedBy: []string{"semantic"},
		}}, true
	}

	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}

	pack, err := Build(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "user-prompt-submit",
		UserPrompt:  "implement unified runtime adapter install for codex and opencode",
		Mode:        ModeAuto,
	})
	if err != nil {
		t.Fatalf("build pack: %v", err)
	}
	if pack.Status != StatusNone {
		t.Fatalf("status = %q, want %q", pack.Status, StatusNone)
	}
	if pack.Serialized != "" {
		t.Fatalf("expected empty serialized payload, got %q", pack.Serialized)
	}
	if strings.Contains(pack.Serialized, "Knowns Guidance") {
		t.Fatalf("expected no serialized memory pack for empty result")
	}
}

func TestCaptureStoresStableGlobalPreference(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}

	entry, created, err := Capture(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "user-prompt-submit",
		UserPrompt:  "toi muon AI tu luu memory, khong doi toi nhac moi them",
		Mode:        ModeAuto,
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if !created {
		t.Fatal("expected capture to create a memory")
	}
	if entry == nil {
		t.Fatal("expected created entry")
	}
	if entry.Layer != models.MemoryLayerGlobal {
		t.Fatalf("layer = %q, want %q", entry.Layer, models.MemoryLayerGlobal)
	}
	if entry.Category != "preference" {
		t.Fatalf("category = %q, want preference", entry.Category)
	}
	if !strings.Contains(entry.Content, "proactively save durable memory") {
		t.Fatalf("unexpected content: %q", entry.Content)
	}

	_, createdAgain, err := Capture(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "user-prompt-submit",
		UserPrompt:  "toi muon AI tu luu memory, khong doi toi nhac moi them",
		Mode:        ModeAuto,
	})
	if err != nil {
		t.Fatalf("capture duplicate: %v", err)
	}
	if createdAgain {
		t.Fatal("expected duplicate capture to be skipped")
	}
}

func TestCaptureStoresProjectDecisionForKnownsSourceOfTruth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}

	entry, created, err := Capture(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "user-prompt-submit",
		UserPrompt:  "AGENTS.md must read behavior from KNOWNS.md in this repo",
		Mode:        ModeAuto,
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if !created {
		t.Fatal("expected project decision memory to be created")
	}
	if entry.Layer != models.MemoryLayerProject {
		t.Fatalf("layer = %q, want %q", entry.Layer, models.MemoryLayerProject)
	}
	if entry.Category != "decision" {
		t.Fatalf("category = %q, want decision", entry.Category)
	}
	if !strings.Contains(entry.Content, "Compatibility shim files") {
		t.Fatalf("unexpected content: %q", entry.Content)
	}
}

func TestCaptureStoresWorkingContextForTemporaryInstruction(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("runtime-memory"); err != nil {
		t.Fatalf("init store: %v", err)
	}

	entry, created, err := Capture(store, Input{
		Runtime:     "opencode",
		ProjectRoot: projectRoot,
		WorkingDir:  projectRoot,
		ActionType:  "user-prompt-submit",
		UserPrompt:  "for now we are debugging the runtime queue workaround",
		Mode:        ModeAuto,
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if !created {
		t.Fatal("expected memory to be created")
	}
	if entry.Layer != models.MemoryLayerProject {
		t.Fatalf("layer = %q, want %q", entry.Layer, models.MemoryLayerProject)
	}
	if entry.Category != "context" {
		t.Fatalf("category = %q, want context", entry.Category)
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
