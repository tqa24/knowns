// Package mcp implements the Model Context Protocol server for the Knowns CLI.
// It exposes all Knowns operations as MCP tools that can be called by AI agents.
package mcp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/howznguyen/knowns/internal/mcp/handlers"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/server"
)

// mcpLog writes to stderr so it doesn't interfere with stdio JSON-RPC transport.
var mcpLog = log.New(os.Stderr, "[knowns-mcp] ", log.LstdFlags)

const version = "0.1.0"

// MCPServer wraps the mcp-go server and holds a reference to the active Store.
// The store is nil until set_project is called.
type MCPServer struct {
	srv   *server.MCPServer
	mu    sync.RWMutex
	store *storage.Store
	root  string
}

// NewMCPServer creates and configures a new MCPServer with all registered tools.
// projectHint is an optional project root path. Detection order:
//  1. projectHint (from --project flag or KNOWNS_PROJECT env)
//  2. Walk up from cwd looking for .knowns/
//
// If a project is found, it is automatically set so callers don't need to call
// set_project first. set_project can still be used to switch projects at runtime.
func NewMCPServer(projectHint string) *MCPServer {
	s := &MCPServer{}

	s.srv = server.NewMCPServer(
		"knowns",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	getStore := func() *storage.Store {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.store
	}

	setStore := func(store *storage.Store, root string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.store = store
		s.root = root
	}

	getRoot := func() string {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.root
	}

	// Register all tool groups.
	handlers.RegisterProjectTools(s.srv, getStore, setStore, getRoot)
	handlers.RegisterTaskTools(s.srv, getStore)
	handlers.RegisterDocTools(s.srv, getStore)
	handlers.RegisterTimeTools(s.srv, getStore)
	handlers.RegisterSearchTools(s.srv, getStore)
	handlers.RegisterCodeTools(s.srv, getStore)
	handlers.RegisterBoardTools(s.srv, getStore)
	handlers.RegisterTemplateTools(s.srv, getStore)
	handlers.RegisterValidateTools(s.srv, getStore)
	handlers.RegisterMemoryTools(s.srv, getStore)

	// Working memory is session-scoped (in-memory only).
	workingMemory := handlers.NewWorkingMemoryStore()
	handlers.RegisterWorkingMemoryTools(s.srv, getStore, func() *handlers.WorkingMemoryStore {
		return workingMemory
	})

	// Auto-detect project from hint or cwd.
	s.autoDetectProject(setStore, projectHint)

	return s
}

// autoDetectProject tries to find and set the project store automatically.
// It checks the hint path first, then walks up from cwd.
func (s *MCPServer) autoDetectProject(setStore func(*storage.Store, string), hint string) {
	// 1. Try explicit hint path
	if hint != "" {
		knownsDir := filepath.Join(hint, ".knowns")
		if info, err := os.Stat(knownsDir); err == nil && info.IsDir() {
			store := storage.NewStore(knownsDir)
			if _, err := store.Config.Load(); err == nil {
				setStore(store, hint)
				return
			}
		}
	}

	// 2. Walk up from cwd
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	dir := cwd
	for {
		knownsDir := filepath.Join(dir, ".knowns")
		if info, err := os.Stat(knownsDir); err == nil && info.IsDir() {
			store := storage.NewStore(knownsDir)
			if _, err := store.Config.Load(); err == nil {
				setStore(store, dir)
			}
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return // reached filesystem root
		}
		dir = parent
	}
}

// Start begins serving MCP requests over stdio transport.
// It logs lifecycle events to stderr for diagnostics and uses the mcp-go
// error logger so transport-level issues are visible to users.
func (s *MCPServer) Start() error {
	s.mu.RLock()
	project := s.root
	s.mu.RUnlock()

	mcpLog.Printf("starting (version=%s, project=%q, pid=%d)", version, project, os.Getpid())
	startedAt := time.Now()

	err := server.ServeStdio(
		s.srv,
		server.WithErrorLogger(mcpLog),
	)

	elapsed := time.Since(startedAt).Round(time.Millisecond)
	if err != nil {
		mcpLog.Printf("exited with error after %s: %v", elapsed, err)
		return fmt.Errorf("mcp server: %w", err)
	}
	mcpLog.Printf("stopped cleanly after %s", elapsed)
	return nil
}
