// Package mcp implements the Model Context Protocol server for the Knowns CLI.
// It exposes all Knowns operations as MCP tools that can be called by AI agents.
package mcp

import (
	"sync"

	"github.com/howznguyen/knowns/internal/mcp/handlers"
	"github.com/howznguyen/knowns/internal/storage"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
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
func NewMCPServer() *MCPServer {
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

	return s
}

// Start begins serving MCP requests over stdio transport.
func (s *MCPServer) Start() error {
	return server.ServeStdio(s.srv)
}

// noStore returns a standard error result when no project has been set.
func noStore() *mcpgo.CallToolResult {
	return mcpgo.NewToolResultError("No project set. Call set_project first.")
}
