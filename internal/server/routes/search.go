package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

// SearchRoutes handles /api/search endpoints.
type SearchRoutes struct {
	store *storage.Store
	mgr   *storage.Manager
}

func (sr *SearchRoutes) getStore() *storage.Store {
	if sr.mgr != nil {
		return sr.mgr.GetStore()
	}
	return sr.store
}

// Register wires the search routes onto r.
func (sr *SearchRoutes) Register(r chi.Router) {
	r.Get("/search", sr.searchHandler)
	r.Get("/retrieve", sr.retrieveHandler)
}

// searchHandler executes a search across tasks and docs.
//
// GET /api/search?q={query}&type={all|task|doc}&mode={keyword|semantic|hybrid}&limit={n}&status={s}&priority={p}&assignee={a}&label={l}&tag={t}
func (sr *SearchRoutes) searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	opts := search.SearchOptions{
		Query:    q.Get("q"),
		Type:     q.Get("type"),
		Mode:     q.Get("mode"),
		Status:   q.Get("status"),
		Priority: q.Get("priority"),
		Assignee: q.Get("assignee"),
		Label:    q.Get("label"),
		Tag:      q.Get("tag"),
		Limit:    limit,
	}

	// Create engine (keyword-only from HTTP for now; embedder wired at server level later).
	engine := search.NewEngine(sr.getStore(), nil, nil)
	results, err := engine.Search(opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Group results by type into {tasks: [...], docs: [...]} shape expected by UI.
	taskResults := []models.SearchResult{}
	docResults := []models.SearchResult{}
	for _, r := range results {
		switch r.Type {
		case "task":
			taskResults = append(taskResults, r)
		case "doc":
			docResults = append(docResults, r)
		default:
			taskResults = append(taskResults, r)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tasks": taskResults,
		"docs":  docResults,
	})
}

// retrieveHandler executes mixed-source retrieval and returns ranked candidates
// plus an assembled context pack for agents and internal APIs.
//
// GET /api/retrieve?q={query}&mode={keyword|semantic|hybrid}&sourceType={doc|task|memory}&expandReferences={true|false}
func (sr *SearchRoutes) retrieveHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	expandRefs := false
	if raw := q.Get("expandReferences"); raw != "" {
		expandRefs = strings.EqualFold(raw, "true") || raw == "1"
	}

	embedder, vecStore, _ := search.InitSemantic(sr.getStore())
	if embedder != nil {
		defer embedder.Close()
	}
	if vecStore != nil {
		defer vecStore.Close()
	}

	engine := search.NewEngine(sr.getStore(), embedder, vecStore)
	response, err := engine.Retrieve(models.RetrievalOptions{
		Query:            q.Get("q"),
		Mode:             q.Get("mode"),
		Limit:            limit,
		SourceTypes:      q["sourceType"],
		ExpandReferences: expandRefs,
		Status:           q.Get("status"),
		Priority:         q.Get("priority"),
		Assignee:         q.Get("assignee"),
		Label:            q.Get("label"),
		Tag:              q.Get("tag"),
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, response)
}
