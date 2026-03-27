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
	"github.com/howznguyen/knowns/internal/registry"
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
	manager         *storage.Manager // Multi-project store manager (may be nil)
	router          chi.Router
	sse             *SSEBroker
	port            int
	projectRoot     string
	opts            Options
	opencodeDaemon  *opencode.Daemon // Shared OpenCode daemon (may be nil if not configured)
	runtimeOpenCode *opencode.Config
	opencodeProxy   *httputil.ReverseProxy // Shared proxy singleton — reused across requests
	opencodeProxyMu sync.RWMutex
	shutdownCh      chan struct{} // Signals graceful shutdown from /api/shutdown endpoint
	cancelSSEFwd    context.CancelFunc     // Cancels the OpenCode SSE forwarder goroutine
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
	var runtimeOpenCode *opencode.Config
	var daemon *opencode.Daemon

	// Initialize OpenCode daemon if configured (only when a project is loaded).
	if store != nil {
		if resolution := resolveOpenCodeConfig(port, store.Config.GetOpenCodeServerConfig()); resolution.configured {
			cfg := resolution.cfg

			// For non-explicit ports, scan candidates to find one already running.
			if !resolution.explicitPort {
				for _, candidate := range deriveOpenCodePortCandidates(port, cfg.Port) {
					candidateCfg := cfg
					candidateCfg.Port = candidate
					if opencode.NewClient(candidateCfg).IsServerAvailable() {
						cfg = candidateCfg
						break
					}
				}
			}

			// Use the shared daemon to ensure exactly one OpenCode process.
			daemon = opencode.NewDaemon(cfg.Host, cfg.Port)
			if err := daemon.EnsureRunning(); err != nil {
				log.Printf("[server] OpenCode daemon not available: %v, using CLI fallback", err)
				daemon = nil
			}

			cfgCopy := cfg
			runtimeOpenCode = &cfgCopy
		}
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
		opencodeDaemon:  daemon,
		runtimeOpenCode: runtimeOpenCode,
		shutdownCh:      make(chan struct{}, 1),
	}

	// Create multi-project manager wrapping the initial store.
	reg := registry.NewRegistry()
	if err := reg.Load(); err != nil {
		log.Printf("warn: could not load project registry: %v", err)
	}
	s.manager = storage.NewManager(store, reg)

	// Build shared proxy singleton once at startup.
	if runtimeOpenCode != nil {
		s.opencodeProxy = buildOpenCodeProxy(*runtimeOpenCode)
	}

	// Startup recovery: mark all previously running workspaces as stopped.
	if store != nil {
		if err := store.Workspaces.MarkAllStopped(); err != nil {
			log.Printf("warn: could not mark workspaces stopped: %v", err)
		}
	}

	// Startup recovery: mark all streaming chat sessions as idle.
	if store != nil {
		if err := store.Chats.MarkAllIdle(); err != nil {
			log.Printf("warn: could not mark chats idle: %v", err)
		}
	}

	s.router = s.buildRouter()

	// Start OpenCode SSE forwarder — multiplexes OpenCode events into the
	// Knowns SSE stream so each browser tab needs only one SSE connection.
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelSSEFwd = cancel
	s.startOpenCodeSSEForwarder(ctx)

	return s
}

// Start listens on the configured port and serves HTTP.
// It binds the port first, then writes the port file only after a successful bind.
func (s *Server) Start() error {
	addr := ":" + strconv.Itoa(s.port)

	// 1. Bind FIRST — fail fast if port is unavailable.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// 2. Port is bound — now safe to write the port file.
	if err := s.writePortFile(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not write .server-port: %v\n", err)
	}
	if err := s.savePortToConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not save port to config: %v\n", err)
	}

	srv := &http.Server{Handler: s.router}

	// 3. Serve on the already-bound listener.
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// 4. Wait for shutdown signal, server error, or remote shutdown request.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		signal.Stop(quit)
	case err := <-serverErr:
		s.cleanupPortFile()
		return fmt.Errorf("server error: %w", err)
	case <-s.shutdownCh:
		log.Printf("[server] Remote shutdown requested")
	}

	// 5. Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[server] HTTP server shutdown error: %v", err)
	}

	// 6. Cleanup port file + OpenCode.
	s.cleanupPortFile()
	if s.cancelSSEFwd != nil {
		s.cancelSSEFwd()
	}
	s.cleanupOpenCodeServer()

	log.Printf("[server] Shutdown complete")
	return nil
}

// cleanupOpenCodeServer stops the OpenCode daemon if we started it.
func (s *Server) cleanupOpenCodeServer() {
	if s.opencodeDaemon != nil && s.opencodeDaemon.StartedByUs() {
		if err := s.opencodeDaemon.Stop(); err != nil {
			log.Printf("[server] Failed to stop OpenCode daemon: %v", err)
		}
	}
}

// startOpenCodeSSEForwarder subscribes to the OpenCode global SSE stream and
// re-broadcasts every event through the Knowns SSEBroker as "opencode:event".
// This eliminates the need for each browser tab to open its own EventSource to
// OpenCode, reducing per-tab SSE connections from 2 to 1 and avoiding HTTP/1.1
// connection exhaustion when multiple tabs are open.
func (s *Server) startOpenCodeSSEForwarder(ctx context.Context) {
	cfg, configured := s.openCodeConfig()
	if !configured {
		return
	}

	go func() {
		client := opencode.NewClient(cfg)
		backoff := time.Second

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if !client.IsServerAvailable() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
					if backoff < 30*time.Second {
						backoff *= 2
					}
					continue
				}
			}

			backoff = time.Second
			log.Printf("[sse-fwd] Connected to OpenCode global event stream")

			err := client.StreamGlobalEvents(ctx, func(event map[string]any) {
				s.sse.Broadcast(routes.SSEEvent{
					Type: "opencode:event",
					Data: event,
				})
			})

			if ctx.Err() != nil {
				return
			}
			log.Printf("[sse-fwd] OpenCode event stream disconnected: %v, reconnecting...", err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}()
}

// writePortFile saves the active port to .knowns/.server-port so CLI commands
// can discover the running server.
func (s *Server) writePortFile() error {
	portFile := filepath.Join(s.store.Root, ".server-port")
	return os.WriteFile(portFile, []byte(strconv.Itoa(s.port)), 0644)
}

// cleanupPortFile removes the .server-port file on shutdown so stale port
// references don't linger after the server exits.
func (s *Server) cleanupPortFile() {
	portFile := filepath.Join(s.store.Root, ".server-port")
	os.Remove(portFile)
}

// handleShutdown handles POST /api/shutdown for graceful remote stop.
func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "shutting down"})
	// Signal the main loop to initiate graceful shutdown.
	select {
	case s.shutdownCh <- struct{}{}:
	default:
	}
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

	// --- Shutdown endpoint (for remote graceful stop) ---
	r.Post("/api/shutdown", s.handleShutdown)

	// --- API routes ---
	r.Route("/api", func(r chi.Router) {
		routes.SetupRoutes(r, s.store, s.sse, s.projectRoot, s.manager)
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

	// Auto-inject directory for session listing so OpenCode only returns
	// sessions belonging to this project (avoids cross-project leakage).
	if r2.URL.Path == "/session" && r.Method == "GET" && s.projectRoot != "" {
		q := r2.URL.Query()
		if q.Get("directory") == "" {
			q.Set("directory", s.projectRoot)
			r2.URL.RawQuery = q.Encode()
		}
	}

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
