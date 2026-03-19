package codegen

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	instructionskills "github.com/howznguyen/knowns/internal/instructions/skills"
	"github.com/howznguyen/knowns/internal/util"
)

type skillVersionFile struct {
	CLIVersion string `json:"cliVersion"`
	SyncedAt   string `json:"syncedAt"`
}

// ReadSyncedSkillVersion reads the CLI version stored in the skills version file.
// It checks .claude/skills/.version first, then .agent/skills/.version.
// Returns empty string if neither file is found or readable.
func ReadSyncedSkillVersion(projectRoot string) string {
	candidates := []string{
		filepath.Join(projectRoot, ".claude", "skills", ".version"),
		filepath.Join(projectRoot, ".agent", "skills", ".version"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var vf skillVersionFile
		if err := json.Unmarshal(data, &vf); err != nil {
			continue
		}
		if vf.CLIVersion != "" {
			return vf.CLIVersion
		}
	}
	return ""
}

// BuiltInSkillCount returns the number of embedded built-in skills.
func BuiltInSkillCount() (int, error) {
	skillDirs, err := listSkillDirs()
	if err != nil {
		return 0, err
	}
	return len(skillDirs), nil
}

// platformNeedsClaudeSkills returns true if the platform writes to .claude/skills/.
func platformNeedsClaudeSkills(p string) bool {
	return p == "claude-code"
}

// platformNeedsAgentSkills returns true if the platform writes to .agent/skills/.
func platformNeedsAgentSkills(p string) bool {
	return p == "opencode" || p == "agents"
}

// SyncSkillsForPlatforms copies embedded built-in skills only to the directories
// required by the given platforms. If platforms is empty, all directories are synced
// (backwards-compatible behaviour matching SyncSkills).
func SyncSkillsForPlatforms(projectRoot string, platforms []string) error {
	wantClaude := len(platforms) == 0
	wantAgent := len(platforms) == 0
	for _, p := range platforms {
		if platformNeedsClaudeSkills(p) {
			wantClaude = true
		}
		if platformNeedsAgentSkills(p) {
			wantAgent = true
		}
	}

	var targets []string
	if wantClaude {
		targets = append(targets, filepath.Join(projectRoot, ".claude", "skills"))
	}
	if wantAgent {
		targets = append(targets, filepath.Join(projectRoot, ".agent", "skills"))
	}
	if len(targets) == 0 {
		return nil
	}

	skillDirs, err := listSkillDirs()
	if err != nil {
		return fmt.Errorf("list embedded skills: %w", err)
	}

	for _, targetDir := range targets {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("create target dir %s: %w", targetDir, err)
		}
		for _, skillDir := range skillDirs {
			dst := filepath.Join(targetDir, skillDir)
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("reset target skill %s: %w", dst, err)
			}
			if err := copyEmbeddedDir(skillDir, dst); err != nil {
				return fmt.Errorf("copy skill %s to %s: %w", skillDir, targetDir, err)
			}
		}
		if err := writeSkillVersionFile(filepath.Join(targetDir, ".version")); err != nil {
			return fmt.Errorf("write version file for %s: %w", targetDir, err)
		}
	}
	return nil
}

// SyncSkills copies embedded built-in skills into the project-local AI platform folders.
func SyncSkills(projectRoot string) error {
	skillDirs, err := listSkillDirs()
	if err != nil {
		return fmt.Errorf("list embedded skills: %w", err)
	}

	targets := []string{
		filepath.Join(projectRoot, ".claude", "skills"),
		filepath.Join(projectRoot, ".agent", "skills"),
	}

	for _, targetDir := range targets {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("create target dir %s: %w", targetDir, err)
		}

		for _, skillDir := range skillDirs {
			dst := filepath.Join(targetDir, skillDir)
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("reset target skill %s: %w", dst, err)
			}
			if err := copyEmbeddedDir(skillDir, dst); err != nil {
				return fmt.Errorf("copy skill %s to %s: %w", skillDir, targetDir, err)
			}
		}

		if err := writeSkillVersionFile(filepath.Join(targetDir, ".version")); err != nil {
			return fmt.Errorf("write version file for %s: %w", targetDir, err)
		}
	}

	return nil
}

func listSkillDirs() ([]string, error) {
	entries, err := fs.ReadDir(instructionskills.Files, ".")
	if err != nil {
		return nil, err
	}

	var skillDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(instructionskills.Files, filepath.ToSlash(filepath.Join(entry.Name(), "SKILL.md"))); err == nil {
			skillDirs = append(skillDirs, entry.Name())
		}
	}

	return skillDirs, nil
}

func writeSkillVersionFile(path string) error {
	data, err := json.MarshalIndent(skillVersionFile{
		CLIVersion: util.Version,
		SyncedAt:   time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func copyEmbeddedDir(src, dst string) error {
	entries, err := fs.ReadDir(instructionskills.Files, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.ToSlash(filepath.Join(src, entry.Name()))
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyEmbeddedDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := copyEmbeddedFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// copyEmbeddedFile copies an embedded file to dst, creating parent directories as needed.
func copyEmbeddedFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := instructionskills.Files.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
