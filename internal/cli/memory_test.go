package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/spf13/cobra"
)

func TestRunMemoryCleanupPlain(t *testing.T) {
	projectRoot := setupMemoryCleanupCLIProject(t)
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	cmd := &cobra.Command{}
	cmd.Flags().Int("older-than", 7, "")
	cmd.Flags().String("layer", models.MemoryLayerProject, "")
	cmd.Flags().Int("limit", 20, "")
	cmd.Flags().Bool("plain", false, "")
	if err := cmd.Flags().Set("plain", "true"); err != nil {
		t.Fatalf("set plain: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	if err := runMemoryCleanup(cmd, nil); err != nil {
		os.Stdout = origStdout
		_ = w.Close()
		t.Fatalf("runMemoryCleanup returned error: %v", err)
	}
	_ = w.Close()
	os.Stdout = origStdout
	_, _ = out.ReadFrom(r)
	got := out.String()
	for _, want := range []string{"MEMORY: stale1", "TITLE: Stale Memory", "AGE_DAYS:", "CONTENT:", "stale content"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "fresh1") {
		t.Fatalf("expected fresh memory to be excluded, got:\n%s", got)
	}
}

func setupMemoryCleanupCLIProject(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	projectRoot := t.TempDir()
	store := storage.NewStore(filepath.Join(projectRoot, ".knowns"))
	if err := store.Init("memory-cleanup-cli-test"); err != nil {
		t.Fatalf("init store: %v", err)
	}
	now := time.Now().UTC()
	if err := store.Memory.Create(&models.MemoryEntry{
		ID:        "stale1",
		Title:     "Stale Memory",
		Layer:     models.MemoryLayerProject,
		Content:   "stale content",
		CreatedAt: now.AddDate(0, 0, -9),
		UpdatedAt: now.AddDate(0, 0, -9),
	}); err != nil {
		t.Fatalf("create stale memory: %v", err)
	}
	if err := store.Memory.Create(&models.MemoryEntry{
		ID:        "fresh1",
		Title:     "Fresh Memory",
		Layer:     models.MemoryLayerProject,
		Content:   "fresh content",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create fresh memory: %v", err)
	}
	return projectRoot
}
