package codegen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncSkillsForPlatformsWritesAgentCompatiblePlatformsToRepoAgentsDir(t *testing.T) {
	projectRoot := t.TempDir()

	if err := SyncSkillsForPlatforms(projectRoot, []string{"opencode", "codex", "antigravity"}); err != nil {
		t.Fatalf("SyncSkillsForPlatforms returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills")); err != nil {
		t.Fatalf("expected .agents/skills to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".agent", "skills")); !os.IsNotExist(err) {
		t.Fatalf("expected .agent/skills not to be created for opencode/codex/antigravity, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".claude", "skills")); !os.IsNotExist(err) {
		t.Fatalf("expected .claude/skills not to be created, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".kiro", "skills")); !os.IsNotExist(err) {
		t.Fatalf("expected .kiro/skills not to be created, got err=%v", err)
	}
}

func TestSyncSkillsForPlatformsKeepsLegacyAgentDirForGenericAgents(t *testing.T) {
	projectRoot := t.TempDir()

	if err := SyncSkillsForPlatforms(projectRoot, []string{"agents"}); err != nil {
		t.Fatalf("SyncSkillsForPlatforms returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".agent", "skills")); err != nil {
		t.Fatalf("expected .agent/skills to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("expected .agents/skills not to be created for generic agents, got err=%v", err)
	}
}

func TestSyncSkillsForPlatformsPreservesLegacyAgentDirForExistingOpencodeProjects(t *testing.T) {
	projectRoot := t.TempDir()
	legacyDir := filepath.Join(projectRoot, ".agent", "skills")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("mkdir legacy agent skills dir: %v", err)
	}

	if err := SyncSkillsForPlatforms(projectRoot, []string{"opencode"}); err != nil {
		t.Fatalf("SyncSkillsForPlatforms returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills")); err != nil {
		t.Fatalf("expected .agents/skills to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".agent", "skills")); err != nil {
		t.Fatalf("expected existing .agent/skills to be preserved and re-synced: %v", err)
	}
}
