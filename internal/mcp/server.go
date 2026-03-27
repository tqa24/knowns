// Package mcp implements the Model Context Protocol server for the Knowns CLI.
// It exposes all Knowns operations as MCP tools that can be called by AI agents.
package mcp

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/howznguyen/knowns/internal/mcp/handlers"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/server"
)

const version = "0.1.0"

// MCPServer wraps the mcp-go server and holds a reference to the active Store.
// The store is nil until set_project is called.
type MCPServer struct {
	srv    *server.MCPServer
	mu     sync.RWMutex
	store  *storage.Store
	root   string
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
	handlers.RegisterBoardTools(s.srv, getStore)
	handlers.RegisterTemplateTools(s.srv, getStore)
	handlers.RegisterValidateTools(s.srv, getStore)

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
func (s *MCPServer) Start() error {
	return server.ServeStdio(s.srv)
}
