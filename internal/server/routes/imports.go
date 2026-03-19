package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/storage"
)

// ImportRoutes handles /api/imports endpoints.
type ImportRoutes struct {
	store *storage.Store
	sse   Broadcaster
}

// Register wires the import routes onto r.
func (ir *ImportRoutes) Register(r chi.Router) {
	r.Get("/imports", ir.list)
	r.Post("/imports", ir.add)
	r.Get("/imports/{name}", ir.get)
	r.Delete("/imports/{name}", ir.remove)
	r.Post("/imports/sync", ir.sync)
	r.Post("/imports/{name}/sync", ir.syncOne)
	r.Post("/imports/sync-all", ir.syncAll)
}

// ImportEntry describes a single registered import.
type ImportEntry struct {
	Name      string   `json:"name"`
	Source    string   `json:"source,omitempty"`
	Type     string   `json:"type,omitempty"`
	Link     bool     `json:"link"`
	AutoSync bool     `json:"autoSync"`
	LastSync string   `json:"lastSync,omitempty"`
	FileCount int     `json:"fileCount"`
	Files    []string `json:"files,omitempty"`
}

// importsDir returns the path to .knowns/imports/.
func (ir *ImportRoutes) importsDir() string {
	return filepath.Join(ir.store.Root, "imports")
}

// collectFiles recursively collects relative file paths under dir.
func collectFiles(dir string) []string {
	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if files == nil {
		files = []string{}
	}
	return files
}

// buildEntry creates an ImportEntry for the given import directory name.
func (ir *ImportRoutes) buildEntry(name string, includeFiles bool) ImportEntry {
	entry := ImportEntry{Name: name, Type: "local"}
	importDir := filepath.Join(ir.importsDir(), name)

	// Read persisted metadata if available.
	if meta, ok := readImportMeta(importDir); ok {
		entry.Source = meta.Source
		if meta.Type != "" {
			entry.Type = meta.Type
		}
		entry.LastSync = meta.LastSync
	}

	files := collectFiles(importDir)
	// Exclude _import.json from the file list.
	var filtered []string
	for _, f := range files {
		if f != importMetaFile {
			filtered = append(filtered, f)
		}
	}
	if filtered == nil {
		filtered = []string{}
	}
	entry.FileCount = len(filtered)
	if includeFiles {
		entry.Files = filtered
	}
	return entry
}

// list returns all registered imports by scanning .knowns/imports/.
//
// GET /api/imports
func (ir *ImportRoutes) list(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(ir.importsDir())
	if os.IsNotExist(err) {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"imports": []ImportEntry{},
			"count":   0,
		})
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var imports []ImportEntry
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			imports = append(imports, ir.buildEntry(e.Name(), false))
		}
	}
	if imports == nil {
		imports = []ImportEntry{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"imports": imports,
		"count":   len(imports),
	})
}

// get returns details of a single import by name.
//
// GET /api/imports/{name}
func (ir *ImportRoutes) get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	importDir := filepath.Join(ir.importsDir(), name)

	if _, err := os.Stat(importDir); os.IsNotExist(err) {
		respondError(w, http.StatusNotFound, "import not found: "+name)
		return
	}

	entry := ir.buildEntry(name, true)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"import": entry,
	})
}

// addImportRequest is the body for POST /api/imports.
type addImportRequest struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Ref    string `json:"ref"`
	Link   bool   `json:"link"`
	DryRun bool   `json:"dryRun"`
}

// nameFromSource extracts a short name from a git URL or path.
// e.g. "https://github.com/user/my-repo" → "my-repo"
func nameFromSource(source string) string {
	// Strip trailing slashes and .git suffix.
	s := strings.TrimRight(source, "/")
	s = strings.TrimSuffix(s, ".git")
	// Take the last path segment.
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		s = s[idx+1:]
	}
	if s == "" {
		return "import"
	}
	return s
}

// importChange describes one file-level change during an import.
type importChange struct {
	Path   string `json:"path"`
	Action string `json:"action"` // "add", "update", "skip"
}

// importMeta is persisted as _import.json inside each import directory.
type importMeta struct {
	Source   string `json:"source"`
	Type     string `json:"type"`
	Ref      string `json:"ref,omitempty"`
	LastSync string `json:"lastSync,omitempty"`
}

const importMetaFile = "_import.json"

// isGitURL returns true if source looks like a git repository URL.
func isGitURL(source string) bool {
	s := strings.ToLower(source)
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "git@") {
		return true
	}
	if strings.HasSuffix(s, ".git") {
		return true
	}
	return false
}

// writeImportMeta persists import metadata into the import directory.
func writeImportMeta(importDir string, meta importMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(importDir, importMetaFile), data, 0644)
}

// readImportMeta reads import metadata from the import directory.
func readImportMeta(importDir string) (importMeta, bool) {
	data, err := os.ReadFile(filepath.Join(importDir, importMetaFile))
	if err != nil {
		return importMeta{}, false
	}
	var meta importMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return importMeta{}, false
	}
	return meta, true
}

// gitCloneImport clones a git repo, copies .knowns/docs/ and .knowns/templates/
// into .knowns/imports/{name}/, and returns the list of changes.
func (ir *ImportRoutes) gitCloneImport(source, name, ref string, dryRun bool) ([]importChange, []string, error) {
	tmpDir, err := os.MkdirTemp("", "knowns-import-*")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Shallow clone.
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, source, tmpDir)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("git clone failed: %s", strings.TrimSpace(stderr.String()))
	}

	// Check for .knowns/ directory.
	knownsDir := filepath.Join(tmpDir, ".knowns")
	if _, err := os.Stat(knownsDir); os.IsNotExist(err) {
		return nil, []string{"no .knowns directory found in " + source}, nil
	}

	// Collect files from docs/ and templates/ subdirectories.
	var changes []importChange
	var warnings []string
	destBase := filepath.Join(ir.importsDir(), name)

	for _, sub := range []string{"docs", "templates"} {
		srcDir := filepath.Join(knownsDir, sub)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			relPath, _ := filepath.Rel(knownsDir, path)
			relPath = filepath.ToSlash(relPath)
			// Nest docs under {name}/ so paths become "docs/{name}/readme.md".
			// This gives doc paths the import prefix automatically.
			if sub == "docs" {
				subRel := strings.TrimPrefix(relPath, "docs/")
				relPath = "docs/" + name + "/" + subRel
			}
			destPath := filepath.Join(destBase, relPath)

			srcData, err := os.ReadFile(path)
			if err != nil {
				warnings = append(warnings, "could not read "+relPath+": "+err.Error())
				return nil
			}

			action := "add"
			if destData, err := os.ReadFile(destPath); err == nil {
				if bytes.Equal(srcData, destData) {
					action = "skip"
				} else {
					action = "update"
				}
			}

			changes = append(changes, importChange{Path: relPath, Action: action})

			if !dryRun && action != "skip" {
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}
				if err := copyFile(path, destPath); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if changes == nil {
		changes = []importChange{}
	}

	// Write metadata if not dry-run.
	if !dryRun {
		if err := os.MkdirAll(destBase, 0755); err != nil {
			return nil, nil, err
		}
		if err := writeImportMeta(destBase, importMeta{
			Source:   source,
			Type:     "git",
			Ref:      ref,
			LastSync: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			warnings = append(warnings, "could not write import metadata: "+err.Error())
		}
	}

	return changes, warnings, nil
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// importResultJSON builds the standard ImportResult response map.
func importResultJSON(name, source, importType string, dryRun bool, changes []importChange, warnings []string) map[string]interface{} {
	added, updated, skipped := 0, 0, 0
	for _, c := range changes {
		switch c.Action {
		case "add":
			added++
		case "update":
			updated++
		case "skip":
			skipped++
		}
	}
	result := map[string]interface{}{
		"success": true,
		"dryRun":  dryRun,
		"import": map[string]interface{}{
			"name":   name,
			"source": source,
			"type":   importType,
		},
		"changes": changes,
		"summary": map[string]interface{}{
			"added":   added,
			"updated": updated,
			"skipped": skipped,
		},
	}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}
	return result
}

// add registers a new import source.
//
// POST /api/imports
func (ir *ImportRoutes) add(w http.ResponseWriter, r *http.Request) {
	var req addImportRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Source == "" {
		respondError(w, http.StatusBadRequest, "source is required")
		return
	}

	name := req.Name
	if name == "" {
		name = nameFromSource(req.Source)
	}

	// Git URL: clone and copy .knowns/docs + templates.
	if isGitURL(req.Source) {
		changes, warnings, err := ir.gitCloneImport(req.Source, name, req.Ref, req.DryRun)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusCreated, importResultJSON(name, req.Source, "git", req.DryRun, changes, warnings))
		return
	}

	// Non-git source: create empty import directory.
	destDir := filepath.Join(ir.importsDir(), name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		respondError(w, http.StatusInternalServerError, "create import dir: "+err.Error())
		return
	}

	importType := "local"
	if req.Type != "" {
		importType = req.Type
	}

	// Persist metadata.
	_ = writeImportMeta(destDir, importMeta{
		Source:   req.Source,
		Type:     importType,
		LastSync: time.Now().UTC().Format(time.RFC3339),
	})

	respondJSON(w, http.StatusCreated, importResultJSON(name, req.Source, importType, req.DryRun, []importChange{}, nil))
}

// remove deletes an import by name.
//
// DELETE /api/imports/{name}
func (ir *ImportRoutes) remove(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	destDir := filepath.Join(ir.importsDir(), name)
	if err := os.RemoveAll(destDir); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"filesDeleted": true,
	})
}

// sync is a stub for syncing remote import packages.
//
// POST /api/imports/sync
func (ir *ImportRoutes) sync(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "synced"})
}

// syncOne synchronises a single import by name.
//
// POST /api/imports/{name}/sync
func (ir *ImportRoutes) syncOne(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	importDir := filepath.Join(ir.importsDir(), name)

	if _, err := os.Stat(importDir); os.IsNotExist(err) {
		respondError(w, http.StatusNotFound, "import not found: "+name)
		return
	}

	meta, ok := readImportMeta(importDir)
	if !ok || !isGitURL(meta.Source) {
		// No metadata or not a git import — nothing to sync.
		ir.sse.Broadcast(SSEEvent{Type: "imports:synced", Data: map[string]string{"name": name}})
		respondJSON(w, http.StatusOK, importResultJSON(name, meta.Source, meta.Type, false, []importChange{}, nil))
		return
	}

	changes, warnings, err := ir.gitCloneImport(meta.Source, name, meta.Ref, false)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "sync failed: "+err.Error())
		return
	}

	ir.sse.Broadcast(SSEEvent{Type: "imports:synced", Data: map[string]string{"name": name}})
	respondJSON(w, http.StatusOK, importResultJSON(name, meta.Source, "git", false, changes, warnings))
}

// syncAll synchronises all registered imports.
// This is a stub – full sync logic is complex and matches the TS pattern.
//
// POST /api/imports/sync-all
func (ir *ImportRoutes) syncAll(w http.ResponseWriter, r *http.Request) {
	ir.sse.Broadcast(SSEEvent{Type: "imports:sync-all", Data: map[string]interface{}{}})
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"dryRun":  false,
		"results": []interface{}{},
		"summary": map[string]interface{}{
			"total":      0,
			"successful": 0,
			"failed":     0,
		},
	})
}
