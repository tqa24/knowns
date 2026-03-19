// Package server provides the HTTP server, REST API, SSE broker, and WebSocket
// support for the Knowns CLI Go rewrite.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"github.com/howznguyen/knowns/internal/agents/opencode"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/server/routes"
	"github.com/howznguyen/knowns/internal/storage"
	ui "github.com/howznguyen/knowns/ui"
	"github.com/rs/cors"
)

// wsUpgrader is the WebSocket upgrader used for the terminal endpoint.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Options configures the server behaviour.
type Options struct {
	Dev bool // enable verbose logging (HTTP requests, WebSocket, etc.)
}

// Server is the top-level HTTP server.
type Server struct {
	store           *storage.Store
	router          chi.Router
	sse             *SSEBroker
	port            int
	projectRoot     string
	opts            Options
	opencodeProcess *os.Process // Track auto-started OpenCode server for cleanup
	runtimeOpenCode *opencode.Config
	opencodeProxy   *httputil.ReverseProxy // Shared proxy singleton — reused across requests
	opencodeProxyMu sync.RWMutex
}

type openCodeConfigResolution struct {
	cfg          opencode.Config
	configured   bool
	explicitPort bool
}

func deriveOpenCodePortCandidates(browserPort int, defaultPort int) []int {
	seen := make(map[int]struct{})
	candidates := make([]int, 0, 3)
	appendCandidate := func(port int) {
		if port <= 0 || port > 65535 {
			return
		}
		if _, exists := seen[port]; exists {
			return
		}
		seen[port] = struct{}{}
		candidates = append(candidates, port)
	}

	if browserPort > 0 {
		base := browserPort * 10
		for offset := 0; offset < 3; offset++ {
			appendCandidate(base + offset)
		}
	}

	if len(candidates) == 0 {
		for offset := 0; offset < 3; offset++ {
			appendCandidate(defaultPort + offset)
		}
	}

	return candidates
}

func resolveOpenCodeConfig(browserPort int, stored *models.OpenCodeServerConfig) openCodeConfigResolution {
	cfg := opencode.DefaultConfig()
	if stored != nil {
		if stored.Host != "" {
			cfg.Host = stored.Host
		}
		cfg.Password = stored.Password
		if stored.Port != 0 {
			cfg.Port = stored.Port
			return openCodeConfigResolution{
				cfg:          cfg,
				configured:   true,
				explicitPort: true,
			}
		}
	}

	candidates := deriveOpenCodePortCandidates(browserPort, cfg.Port)
	if len(candidates) > 0 {
		cfg.Port = candidates[0]
	}
	return openCodeConfigResolution{
		cfg:        cfg,
		configured: true,
	}
}

// tryAutoStartOpenCodeServer attempts to start the OpenCode server if the CLI is available.
// Returns the process if started by us, nil if already running or failed.
func tryAutoStartOpenCodeServer(host string, ports []int) (*os.Process, int, error) {
	// Check if opencode CLI is available
	if _, err := exec.LookPath("opencode"); err != nil {
		return nil, 0, fmt.Errorf("opencode CLI not found: %w", err)
	}

	for _, port := range ports {
		// Skip ports that are already occupied. In auto-port mode this avoids
		// binding OpenCode to the Knowns UI port or another local service.
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
		if err == nil {
			conn.Close()
			continue
		}

		// Start opencode serve in background
		// Command must be first positional argument
		args := []string{"serve", "--hostname", host, "--port", strconv.Itoa(port), "--cors", "*"}
		cmd := exec.Command("opencode", args...)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		// Explicitly set empty stdin to prevent interactive mode
		cmd.Stdin = strings.NewReader("")

		if err := cmd.Start(); err != nil {
			return nil, 0, fmt.Errorf("failed to start opencode serve on port %d: %w", port, err)
		}

		log.Printf("[server] Spawned OpenCode server (pid %d) on port %d", cmd.Process.Pid, port)
		return cmd.Process, port, nil
	}

	return nil, 0, fmt.Errorf("no available OpenCode port for host %s in candidates %v", host, ports)
}

// buildOpenCodeProxy creates a shared reverse proxy for the OpenCode server.
// The proxy uses a pooled transport with keep-alive so connections are reused
// across requests instead of creating a new TCP connection each time.
func buildOpenCodeProxy(cfg opencode.Config) *httputil.ReverseProxy {
	target, _ := url.Parse(fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port))

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	proxy.FlushInterval = -1 // flush immediately — required for SSE streaming
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Let the Knowns server own CORS for proxied OpenCode responses.
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Expose-Headers")
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		http.Error(rw, "OpenCode server unavailable: "+proxyErr.Error(), http.StatusBadGateway)
	}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		if cfg.Password != "" {
			req.SetBasicAuth(cfg.Username, cfg.Password)
		}
	}

	return proxy
}

// NewServer creates a Server wired to the given store.
// projectRoot is the directory that contains the .knowns/ folder.
// port is the TCP port to listen on (e.g. 3737).
func NewServer(store *storage.Store, projectRoot string, port int, opts Options) *Server {
	var opencodeProcess *os.Process
	var runtimeOpenCode *opencode.Config

	// Initialize OpenCode server client if configured
	if resolution := resolveOpenCodeConfig(port, store.Config.GetOpenCodeServerConfig()); resolution.configured {
		cfg := resolution.cfg
		if resolution.explicitPort {
			client := opencode.NewClient(cfg)
			if !client.IsServerAvailable() {
				var err error
				opencodeProcess, _, err = tryAutoStartOpenCodeServer(cfg.Host, []int{cfg.Port})
				if err != nil {
					log.Printf("[server] OpenCode server not available at %s:%d, using CLI fallback", cfg.Host, cfg.Port)
				} else {
					time.Sleep(2 * time.Second)
				}
			}
		} else {
			for _, candidate := range deriveOpenCodePortCandidates(port, cfg.Port) {
				candidateCfg := cfg
				candidateCfg.Port = candidate
				if opencode.NewClient(candidateCfg).IsServerAvailable() {
					cfg = candidateCfg
					break
				}
			}

			client := opencode.NewClient(cfg)
			if !client.IsServerAvailable() {
				var err error
				opencodeProcess, cfg.Port, err = tryAutoStartOpenCodeServer(cfg.Host, deriveOpenCodePortCandidates(port, cfg.Port))
				if err != nil {
					log.Printf("[server] OpenCode server not available on derived ports for %s, using CLI fallback", cfg.Host)
				} else {
					time.Sleep(2 * time.Second)
				}
			}
		}

		cfgCopy := cfg
		runtimeOpenCode = &cfgCopy
	}

	// Silence standard log output unless dev mode is enabled.
	if !opts.Dev {
		log.SetOutput(io.Discard)
	}

	s := &Server{
		store:           store,
		sse:             NewSSEBroker(),
		port:            port,
		projectRoot:     projectRoot,
		opts:            opts,
		opencodeProcess: opencodeProcess,
		runtimeOpenCode: runtimeOpenCode,
	}

	// Build shared proxy singleton once at startup.
	if runtimeOpenCode != nil {
		s.opencodeProxy = buildOpenCodeProxy(*runtimeOpenCode)
	}

	// Startup recovery: mark all previously running workspaces as stopped.
	if err := store.Workspaces.MarkAllStopped(); err != nil {
		log.Printf("warn: could not mark workspaces stopped: %v", err)
	}

	// Startup recovery: mark all streaming chat sessions as idle.
	if err := store.Chats.MarkAllIdle(); err != nil {
		log.Printf("warn: could not mark chats idle: %v", err)
	}

	s.router = s.buildRouter()
	return s
}

// Start listens on the configured port and serves HTTP.
// It writes the port number to .knowns/.server-port before accepting connections.
func (s *Server) Start() error {
	if err := s.writePortFile(); err != nil {
		// Non-fatal: log and continue.
		fmt.Fprintf(os.Stderr, "warn: could not write .server-port: %v\n", err)
	}

	// Save the active port into config.json so the UI can discover it.
	if err := s.savePortToConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not save port to config: %v\n", err)
	}

	addr := ":" + strconv.Itoa(s.port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[server] HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	signal.Stop(quit)

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[server] HTTP server shutdown error: %v", err)
	}

	// Cleanup: kill auto-started OpenCode server
	s.cleanupOpenCodeServer()

	log.Printf("[server] Shutdown complete")
	return nil
}

// cleanupOpenCodeServer kills the auto-started OpenCode server process.
func (s *Server) cleanupOpenCodeServer() {
	if s.opencodeProcess != nil {
		log.Printf("[server] Stopping auto-started OpenCode server (pid %d)", s.opencodeProcess.Pid)
		if err := s.opencodeProcess.Kill(); err != nil {
			log.Printf("[server] Failed to kill OpenCode server: %v", err)
		} else {
			s.opencodeProcess.Wait()
			log.Printf("[server] OpenCode server stopped")
		}
	}
}

// writePortFile saves the active port to .knowns/.server-port so CLI commands
// can discover the running server.
func (s *Server) writePortFile() error {
	portFile := filepath.Join(s.store.Root, ".server-port")
	return os.WriteFile(portFile, []byte(strconv.Itoa(s.port)), 0644)
}

// savePortToConfig persists the server port into config.json so the browser
// and other tools can discover the running server.
func (s *Server) savePortToConfig() error {
	project, err := s.store.Config.Load()
	if err != nil {
		return err
	}
	project.Settings.ServerPort = s.port
	return s.store.Config.Save(project)
}

// buildRouter assembles the chi.Router with all middleware and route groups.
func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()

	// --- Middleware ---
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if s.opts.Dev {
		r.Use(middleware.Logger)
	}

	// CORS: allow all origins for development.
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	})
	r.Use(c.Handler)

	// --- SSE endpoint ---
	r.Get("/api/events", s.sse.Subscribe)

	// --- API routes ---
	r.Route("/api", func(r chi.Router) {
		routes.SetupRoutes(r, s.store, s.sse, s.projectRoot)
	})

	// --- OpenCode API proxy (for frontend to call OpenCode directly) ---
	r.Route("/api/opencode", func(r chi.Router) {
		r.Get("/status", s.getOpenCodeStatus)
		r.Get("/*", s.proxyOpenCode)
		r.Post("/*", s.proxyOpenCode)
		r.Put("/*", s.proxyOpenCode)
		r.Patch("/*", s.proxyOpenCode)
		r.Delete("/*", s.proxyOpenCode)
	})

	// --- WebSocket chat ---
	r.Get("/ws/chat", s.handleChatWS)

	// --- Static UI assets ---
	s.mountUI(r)

	return r
}

func (s *Server) openCodeConfig() (opencode.Config, bool) {
	if s.runtimeOpenCode != nil {
		return *s.runtimeOpenCode, true
	}
	resolution := resolveOpenCodeConfig(s.port, s.store.Config.GetOpenCodeServerConfig())
	return resolution.cfg, resolution.configured
}

// handleChatWS upgrades the connection to WebSocket for chat sessions.
//
// GET /ws/chat?chatId=xxx
func (s *Server) handleChatWS(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chatId")
	if chatID == "" {
		http.Error(w, "chatId query parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("[ws] Chat WS connection request for chat %s", chatID)

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] Upgrade failed for chat %s: %v", chatID, err)
		return
	}
	defer conn.Close()

	// Read loop keeps connection alive until client disconnects.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ws] Chat WS disconnected for chat %s: %v", chatID, err)
			break
		}
	}
}

// getOrRebuildProxy returns the shared proxy singleton, rebuilding it if the
// runtime config has changed (e.g. after a config reload).
func (s *Server) getOrRebuildProxy(cfg opencode.Config) *httputil.ReverseProxy {
	s.opencodeProxyMu.RLock()
	p := s.opencodeProxy
	s.opencodeProxyMu.RUnlock()
	if p != nil {
		return p
	}
	// First request after a config change — build and cache.
	newProxy := buildOpenCodeProxy(cfg)
	s.opencodeProxyMu.Lock()
	s.opencodeProxy = newProxy
	s.opencodeProxyMu.Unlock()
	return newProxy
}

// proxyOpenCode proxies requests to the OpenCode server using a shared
// reverse proxy singleton so TCP connections are pooled and reused.
func (s *Server) proxyOpenCode(w http.ResponseWriter, r *http.Request) {
	cfg, configured := s.openCodeConfig()
	if !configured {
		http.Error(w, "OpenCode server is not configured", http.StatusServiceUnavailable)
		return
	}

	proxy := s.getOrRebuildProxy(cfg)

	// Rewrite the path: strip the /api/opencode prefix before forwarding.
	r2 := r.Clone(r.Context())
	r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/opencode")
	if r2.URL.Path == "" {
		r2.URL.Path = "/"
	}
	r2.URL.RawQuery = r.URL.RawQuery

	proxy.ServeHTTP(w, r2)
}

func (s *Server) getOpenCodeStatus(w http.ResponseWriter, r *http.Request) {
	cfg, configured := s.openCodeConfig()
	cliAvailable := false
	if _, err := exec.LookPath("opencode"); err == nil {
		cliAvailable = true
	}

	status := map[string]any{
		"configured":   configured,
		"available":    false,
		"host":         cfg.Host,
		"port":         cfg.Port,
		"cliAvailable": cliAvailable,
	}

	if !configured {
		status["error"] = "OpenCode server is not configured."
		writeJSON(w, http.StatusOK, status)
		return
	}

	client := opencode.NewClient(cfg)
	if client.IsServerAvailable() {
		status["available"] = true
		writeJSON(w, http.StatusOK, status)
		return
	}

	message := fmt.Sprintf("Cannot reach OpenCode server at %s:%d.", cfg.Host, cfg.Port)
	if !cliAvailable {
		message += " The `opencode` CLI is not installed."
	}
	status["error"] = message
	writeJSON(w, http.StatusOK, status)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// mountUI embeds the compiled React UI assets and serves them at /.
// All non-API, non-asset paths fall through to index.html (SPA support).
func (s *Server) mountUI(r chi.Router) {
	distFS, err := fs.Sub(ui.Assets, "dist")
	if err != nil {
		// If the embed is unavailable (e.g. during development without a build),
		// skip UI serving gracefully.
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path

		// Skip paths that look like API routes (belt-and-suspenders).
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, req)
			return
		}

		if path == "/" || path == "" {
			path = "index.html"
		} else {
			// Strip leading slash for fs.Stat.
			path = strings.TrimPrefix(path, "/")
		}

		if _, statErr := fs.Stat(distFS, path); statErr == nil {
			fileServer.ServeHTTP(w, req)
			return
		}

		// Fallback to index.html for client-side routing (SPA).
		index, readErr := distFS.Open("index.html")
		if readErr != nil {
			http.NotFound(w, req)
			return
		}
		defer index.Close()

		data, readErr := io.ReadAll(index)
		if readErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
