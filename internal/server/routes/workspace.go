package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/storage"
)

// WorkspaceRoutes handles /api/workspaces endpoints for multi-project management.
type WorkspaceRoutes struct {
	manager *storage.Manager
	sse     Broadcaster
}

// Register wires workspace routes onto r.
func (wr *WorkspaceRoutes) Register(r chi.Router) {
	r.Get("/workspaces", wr.list)
	r.Post("/workspaces/switch", wr.switchWorkspace)
	r.Post("/workspaces/scan", wr.scan)
	r.Post("/workspaces/auto-scan", wr.autoScan)
	r.Delete("/workspaces/{id}", wr.remove)
}

// list returns all registered projects from the global registry.
//
// GET /api/workspaces
func (wr *WorkspaceRoutes) list(w http.ResponseWriter, r *http.Request) {
	reg := wr.manager.GetRegistry()
	if reg == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}
	projects := reg.Projects
	if projects == nil {
		respondJSON(w, http.StatusOK, []struct{}{})
		return
	}
	respondJSON(w, http.StatusOK, projects)
}

// switchWorkspace swaps the active store to a different project.
//
// POST /api/workspaces/switch
// Body: {"id": "project-id"}
func (wr *WorkspaceRoutes) switchWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.ID == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	reg := wr.manager.GetRegistry()
	if reg == nil {
		respondError(w, http.StatusInternalServerError, "registry not available")
		return
	}

	// Find project by ID.
	var projectPath string
	for _, p := range reg.Projects {
		if p.ID == body.ID {
			projectPath = p.Path
			break
		}
	}
	if projectPath == "" {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	// Switch the active store.
	if _, err := wr.manager.Switch(projectPath); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast SSE refresh event so UI reloads all data.
	wr.sse.Broadcast(SSEEvent{
		Type: "refresh",
		Data: map[string]string{"reason": "workspace-switch"},
	})

	// Return the updated active project info.
	for _, p := range reg.Projects {
		if p.ID == body.ID {
			respondJSON(w, http.StatusOK, p)
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "switched"})
}

// scan discovers new projects in the given directories.
//
// POST /api/workspaces/scan
// Body: {"dirs": ["/path/to/parent1", "/path/to/parent2"]}
func (wr *WorkspaceRoutes) scan(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Dirs []string `json:"dirs"`
	}
	if err := decodeJSON(r, &body); err != nil || len(body.Dirs) == 0 {
		respondError(w, http.StatusBadRequest, "dirs array is required")
		return
	}

	reg := wr.manager.GetRegistry()
	if reg == nil {
		respondError(w, http.StatusInternalServerError, "registry not available")
		return
	}

	added, err := reg.Scan(body.Dirs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if added == nil {
		respondJSON(w, http.StatusOK, []struct{}{})
		return
	}
	respondJSON(w, http.StatusOK, added)
}

// autoScan discovers projects in common directories automatically.
// Scans home directory and well-known project directories at depth 1.
//
// POST /api/workspaces/auto-scan
func (wr *WorkspaceRoutes) autoScan(w http.ResponseWriter, r *http.Request) {
	reg := wr.manager.GetRegistry()
	if reg == nil {
		respondError(w, http.StatusInternalServerError, "registry not available")
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "cannot determine home directory")
		return
	}

	// Common project directories to scan
	candidates := []string{
		home,
		filepath.Join(home, "Projects"),
		filepath.Join(home, "projects"),
		filepath.Join(home, "Developer"),
		filepath.Join(home, "developer"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "code"),
		filepath.Join(home, "repos"),
		filepath.Join(home, "Repos"),
		filepath.Join(home, "workspace"),
		filepath.Join(home, "Workspace"),
		filepath.Join(home, "src"),
		filepath.Join(home, "go", "src"),
		filepath.Join(home, "dev"),
		filepath.Join(home, "Dev"),
	}

	// Filter to only existing directories
	var dirs []string
	for _, d := range candidates {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			dirs = append(dirs, d)
		}
	}

	added, err := reg.Scan(dirs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if added == nil {
		respondJSON(w, http.StatusOK, []struct{}{})
		return
	}
	respondJSON(w, http.StatusOK, added)
}

// remove deletes a project from the registry.
//
// DELETE /api/workspaces/{id}
func (wr *WorkspaceRoutes) remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	reg := wr.manager.GetRegistry()
	if reg == nil {
		respondError(w, http.StatusInternalServerError, "registry not available")
		return
	}

	if err := reg.Remove(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
