// Package mcp implements the Model Context Protocol server for the Knowns CLI.
// It exposes all Knowns operations as MCP tools that can be called by AI agents.
package mcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/howznguyen/knowns/internal/mcp/handlers"
	"github.com/howznguyen/knowns/internal/permissions"
	"github.com/howznguyen/knowns/internal/runtimequeue"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/server"
)

// mcpLog writes to stderr and a log file without touching stdout JSON-RPC transport.
var mcpLog = newMCPLogger()

const version = "0.1.0"

const (
	defaultMCPLogMaxSizeBytes = 10 * 1024 * 1024
	defaultMCPLogMaxBackups   = 3
)

func newMCPLogger() *log.Logger {
	writers := []io.Writer{os.Stderr}
	pid := os.Getpid()

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		logDir := filepath.Join(home, ".knowns", "logs")
		if mkdirErr := os.MkdirAll(logDir, 0755); mkdirErr == nil {
			cleanupOldMCPLogs(logDir, 7*24*time.Hour)
			logPath := filepath.Join(logDir, "mcp.log")
			if writer, openErr := newRotatingFileWriter(logPath, mcpLogMaxSizeBytes(), mcpLogMaxBackups()); openErr == nil {
				writers = append(writers, writer)
			}
		}
	}

	return log.New(io.MultiWriter(writers...), fmt.Sprintf("[knowns-mcp pid=%d] ", pid), log.LstdFlags)
}

// cleanupOldMCPLogs deletes legacy per-PID log files older than maxAge.
// Keeps the shared mcp.log and its rotated backups.
func cleanupOldMCPLogs(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "mcp.log" {
			continue
		}
		if !strings.HasPrefix(name, "mcp-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

type rotatingFileWriter struct {
	path       string
	maxSize    int64
	maxBackups int

	mu   sync.Mutex
	file *os.File
	size int64
}

func newRotatingFileWriter(path string, maxSize int64, maxBackups int) (*rotatingFileWriter, error) {
	if maxSize <= 0 {
		maxSize = defaultMCPLogMaxSizeBytes
	}
	if maxBackups < 0 {
		maxBackups = 0
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	return &rotatingFileWriter{
		path:       path,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		file:       file,
		size:       info.Size(),
	}, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return 0, fmt.Errorf("log file is closed")
	}

	if w.maxSize > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFileWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}

	if w.maxBackups > 0 {
		oldest := fmt.Sprintf("%s.%d", w.path, w.maxBackups)
		_ = os.Remove(oldest)
		for i := w.maxBackups - 1; i >= 1; i-- {
			src := fmt.Sprintf("%s.%d", w.path, i)
			dst := fmt.Sprintf("%s.%d", w.path, i+1)
			_ = os.Rename(src, dst)
		}
		_ = os.Rename(w.path, fmt.Sprintf("%s.1", w.path))
	} else {
		_ = os.Remove(w.path)
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = file
	w.size = 0
	return nil
}

func mcpLogMaxSizeBytes() int64 {
	if raw := os.Getenv("KNOWNS_MCP_LOG_MAX_SIZE_MB"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return int64(n) * 1024 * 1024
		}
	}
	return defaultMCPLogMaxSizeBytes
}

func mcpLogMaxBackups() int {
	if raw := os.Getenv("KNOWNS_MCP_LOG_MAX_BACKUPS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			return n
		}
	}
	return defaultMCPLogMaxBackups
}

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

	// Create global audit store at ~/.knowns/audit.jsonl.
	auditStore := storage.NewGlobalAuditStore()

	// Build permission guard config loader.
	permConfigLoader := func() *permissions.PermissionConfig {
		store := getStore()
		if store == nil {
			return nil
		}
		cfg, err := store.Config.Load()
		if err != nil {
			return nil
		}
		return cfg.Settings.Permissions
	}

	s.srv = server.NewMCPServer(
		"knowns",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithToolHandlerMiddleware(permissions.NewGuardMiddleware(permConfigLoader)),
		server.WithHooks(newAuditHooks(auditStore, getRoot)),
	)

	// Register all tool groups.
	handlers.RegisterProjectTool(s.srv, getStore, setStore, getRoot)
	handlers.RegisterTaskTool(s.srv, getStore)
	handlers.RegisterDocTool(s.srv, getStore)
	handlers.RegisterTimeTool(s.srv, getStore)
	handlers.RegisterSearchTool(s.srv, getStore)
	handlers.RegisterCodeTool(s.srv, getStore)
	// Board view is now part of RegisterTaskTool (action: board).
	handlers.RegisterTemplateTool(s.srv, getStore)
	handlers.RegisterValidateTools(s.srv, getStore)
	handlers.RegisterMemoryTool(s.srv, getStore)

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
	store := s.store
	s.mu.RUnlock()

	mcpLog.Printf("starting (version=%s, project=%q, pid=%d)", version, project, os.Getpid())
	startedAt := time.Now()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var lease *runtimequeue.ClientHandle
	var leaseCtx context.Context
	var cancel context.CancelFunc
	if store != nil && !runtimequeue.ShouldBypassDaemon() {
		var err error
		lease, err = runtimequeue.AcquireClient("mcp", store.Root, false)
		if err != nil {
			mcpLog.Printf("runtime lease unavailable: %v", err)
		} else {
			leaseCtx, cancel = context.WithCancel(context.Background())
			runtimequeue.StartHeartbeat(leaseCtx, lease)
		}
	}

	cleanup := func(reason string) {
		mcpLog.Printf("shutdown: %s", reason)
		signal.Stop(sigCh)
		if cancel != nil {
			cancel()
		}
		if lease != nil {
			_ = lease.Release()
		}
	}

	go func() {
		sig, ok := <-sigCh
		if !ok {
			return
		}
		cleanup(fmt.Sprintf("signal %s", sig))
		os.Exit(0)
	}()

	defer cleanup("serve returned")

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
