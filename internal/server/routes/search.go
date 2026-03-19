package routes

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

// SearchRoutes handles /api/search endpoints.
type SearchRoutes struct {
	store *storage.Store
}

// Register wires the search routes onto r.
func (sr *SearchRoutes) Register(r chi.Router) {
	r.Get("/search", sr.searchHandler)
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
	engine := search.NewEngine(sr.store, nil, nil)
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
