// Package storage provides read/write access to the .knowns/ directory format.
// It is fully backward-compatible with the TypeScript Knowns CLI.
package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/howznguyen/knowns/internal/models"
)

// Store is the top-level coordinator for all .knowns/ sub-stores.
type Store struct {
	// Root is the absolute path to the .knowns/ directory.
	Root       string
	Tasks      *TaskStore
	Docs       *DocStore
	Config     *ConfigStore
	Time       *TimeStore
	Templates  *TemplateStore
	Versions   *VersionStore
	Workspaces *WorkspaceStore
	Chats      *ChatStore
}

// NewStore creates a Store rooted at the given .knowns/ directory path.
// The directory does not need to exist yet; call Init to create it.
func NewStore(root string) *Store {
	s := &Store{Root: root}
	s.Tasks = &TaskStore{root: root}
	s.Docs = &DocStore{root: root}
	s.Config = &ConfigStore{root: root}
	s.Time = &TimeStore{root: root}
	s.Templates = &TemplateStore{root: root}
	s.Versions = &VersionStore{root: root}
	s.Workspaces = &WorkspaceStore{root: root}
	s.Chats = &ChatStore{root: root}
	return s
}

// FindProjectRoot walks up from startDir looking for a .knowns/ directory.
// Returns the absolute path to the .knowns/ directory, or an error if not found.
func FindProjectRoot(startDir string) (string, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, ".knowns")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .knowns/ directory found (started from %s)", startDir)
}

// Init creates the .knowns/ directory structure for a new project.
func (s *Store) Init(name string) error {
	dirs := []string{
		s.Root,
		filepath.Join(s.Root, "tasks"),
		filepath.Join(s.Root, "docs"),
		filepath.Join(s.Root, "archive"),
		filepath.Join(s.Root, "versions"),
		filepath.Join(s.Root, "templates"),
		filepath.Join(s.Root, "imports"),
		filepath.Join(s.Root, "worktrees"),
		filepath.Join(s.Root, ".search"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("init: create dir %s: %w", d, err)
		}
	}

	// Write default config if it does not exist yet.
	configPath := filepath.Join(s.Root, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := s.Config.initDefault(name); err != nil {
			return fmt.Errorf("init: write config: %w", err)
		}
	}

	// Write empty time state if it does not exist.
	timePath := filepath.Join(s.Root, "time.json")
	if _, err := os.Stat(timePath); os.IsNotExist(err) {
		if err := s.Time.SaveState(&models.TimeState{Active: []models.ActiveTimer{}}); err != nil {
			return fmt.Errorf("init: write time.json: %w", err)
		}
	}

	// Write empty time-entries if it does not exist.
	entriesPath := filepath.Join(s.Root, "time-entries.json")
	if _, err := os.Stat(entriesPath); os.IsNotExist(err) {
		if err := writeJSON(entriesPath, map[string]interface{}{}); err != nil {
			return fmt.Errorf("init: write time-entries.json: %w", err)
		}
	}

	// Write empty workspaces list if it does not exist.
	wsPath := filepath.Join(s.Root, "workspaces.json")
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		if err := writeJSON(wsPath, []interface{}{}); err != nil {
			return fmt.Errorf("init: write workspaces.json: %w", err)
		}
	}

	// Write empty chats list if it does not exist.
	chatsPath := filepath.Join(s.Root, "chats.json")
	if _, err := os.Stat(chatsPath); os.IsNotExist(err) {
		if err := writeJSON(chatsPath, []interface{}{}); err != nil {
			return fmt.Errorf("init: write chats.json: %w", err)
		}
	}

	return nil
}
