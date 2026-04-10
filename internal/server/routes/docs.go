package routes

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

// docResponse transforms a flat Doc model into the nested shape the UI expects:
// { filename, path, folder, metadata: { title, description, tags, ... }, content, isImported, source }
type docMetadataResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	Tags        []string `json:"tags"`
	Order       *int     `json:"order,omitempty"`
}

type docResponse struct {
	Filename   string              `json:"filename"`
	Path       string              `json:"path"`
	Folder     string              `json:"folder"`
	Metadata   docMetadataResponse `json:"metadata"`
	Content    string              `json:"content"`
	IsImported bool                `json:"isImported,omitempty"`
	Source     string              `json:"source,omitempty"`
}

func toDocResponse(d *models.Doc) docResponse {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	filename := filepath.Base(d.Path) + ".md"
	return docResponse{
		Filename: filename,
		Path:     d.Path,
		Folder:   d.Folder,
		Metadata: docMetadataResponse{
			Title:       d.Title,
			Description: d.Description,
			CreatedAt:   d.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
			Tags:        tags,
			Order:       d.Order,
		},
		Content:    d.Content,
		IsImported: d.IsImported,
		Source:     d.ImportSource,
	}
}

// DocRoutes handles /api/docs endpoints.
type DocRoutes struct {
	store *storage.Store
	mgr   *storage.Manager
	sse   Broadcaster
}

func (dr *DocRoutes) getStore() *storage.Store {
	if dr.mgr != nil {
		return dr.mgr.GetStore()
	}
	return dr.store
}

// Register wires the doc routes onto r.
func (dr *DocRoutes) Register(r chi.Router) {
	r.Get("/docs", dr.list)
	r.Post("/docs", dr.create)
	// Wildcard routes must come after specific ones.
	r.Get("/docs/*", dr.getOrHistory)
	r.Put("/docs/*", dr.update)
}

// getOrHistory dispatches to the history handler if the path ends with /history,
// otherwise falls through to the regular get handler.
func (dr *DocRoutes) getOrHistory(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "*")
	if strings.HasSuffix(raw, "/history") {
		dr.history(w, r)
		return
	}
	dr.get(w, r)
}

// list returns all documents.
//
// GET /api/docs
func (dr *DocRoutes) list(w http.ResponseWriter, r *http.Request) {
	docs, err := dr.getStore().Docs.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]docResponse, 0, len(docs))
	for _, d := range docs {
		result = append(result, toDocResponse(d))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"docs": result,
	})
}

// docPathParam extracts the wildcard path parameter and normalises it.
// Handles URL-encoded paths (e.g. ai%2Fplatforms.md → ai/platforms).
func docPathParam(r *http.Request) string {
	raw := chi.URLParam(r, "*")
	// URL-decode in case the client used encodeURIComponent (e.g. %2F → /).
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimSuffix(raw, ".md")
	return raw
}

// get retrieves a single doc by path.
//
// GET /api/docs/*
func (dr *DocRoutes) get(w http.ResponseWriter, r *http.Request) {
	path := docPathParam(r)
	doc, err := dr.getStore().Docs.Get(path)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, toDocResponse(doc))
}

// create persists a new doc.
//
// POST /api/docs
func (dr *DocRoutes) create(w http.ResponseWriter, r *http.Request) {
	var doc models.Doc
	if err := decodeJSON(r, &doc); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	// Auto-generate path from title + folder when not provided.
	if doc.Path == "" && doc.Title != "" {
		slug := slugifyTitle(doc.Title)
		if doc.Folder != "" {
			doc.Path = doc.Folder + "/" + slug
		} else {
			doc.Path = slug
		}
	}
	if doc.Path == "" {
		respondError(w, http.StatusBadRequest, "path or title is required")
		return
	}

	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	if doc.Tags == nil {
		doc.Tags = []string{}
	}

	if err := dr.getStore().Docs.Create(&doc); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	search.BestEffortIndexDoc(dr.getStore(), doc.Path)

	// Record initial version.
	_ = dr.getStore().Versions.SaveDocVersion(doc.Path, models.DocVersion{
		Changes:  dr.getStore().Versions.TrackDocChanges(nil, &doc),
		Snapshot: storage.DocToSnapshot(&doc),
	})

	dr.sse.Broadcast(SSEEvent{Type: "docs:updated", Data: map[string]string{"path": doc.Path}})
	respondJSON(w, http.StatusCreated, toDocResponse(&doc))
}

// update saves changes to an existing doc.
//
// PUT /api/docs/*
func (dr *DocRoutes) update(w http.ResponseWriter, r *http.Request) {
	path := docPathParam(r)

	existing, err := dr.getStore().Docs.Get(path)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	oldDoc := *existing

	var payload struct {
		Title       *string   `json:"title"`
		Description *string   `json:"description"`
		Content     *string   `json:"content"`
		Tags        *[]string `json:"tags"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	doc := oldDoc
	doc.Path = path
	doc.UpdatedAt = time.Now().UTC()

	if payload.Title != nil {
		doc.Title = *payload.Title
	}
	if payload.Description != nil {
		doc.Description = *payload.Description
	}
	if payload.Content != nil {
		doc.Content = *payload.Content
	}
	if payload.Tags != nil {
		doc.Tags = *payload.Tags
	}

	if doc.Tags == nil {
		doc.Tags = []string{}
	}

	if err := dr.getStore().Docs.Update(&doc); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	search.BestEffortIndexDoc(dr.getStore(), doc.Path)

	// Save version if something changed.
	changes := dr.getStore().Versions.TrackDocChanges(&oldDoc, &doc)
	if len(changes) > 0 {
		_ = dr.getStore().Versions.SaveDocVersion(doc.Path, models.DocVersion{
			Changes:  changes,
			Snapshot: storage.DocToSnapshot(&doc),
		})
	}

	dr.sse.Broadcast(SSEEvent{Type: "docs:updated", Data: map[string]string{"path": path}})
	respondJSON(w, http.StatusOK, toDocResponse(&doc))
}

// history returns the version history for a document.
//
// GET /api/docs/*/history
func (dr *DocRoutes) history(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "*")
	// Strip the trailing /history suffix to get the doc path.
	path := strings.TrimSuffix(raw, "/history")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".md")

	h, err := dr.getStore().Versions.GetDocHistory(path)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, h)
}
