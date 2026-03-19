package search

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
)

// SearchOptions configures a search query.
type SearchOptions struct {
	Query    string
	Type     string // "all", "task", "doc"
	Mode     string // "keyword", "semantic", "hybrid"
	Status   string
	Priority string
	Assignee string
	Label    string
	Tag      string
	Limit    int
}

const (
	semanticWeight = 0.6
	keywordWeight  = 0.4
)

// Engine provides keyword, semantic, and hybrid search across tasks and docs.
type Engine struct {
	store    *storage.Store
	embedder *Embedder    // nil if semantic not available
	vecStore VectorStore  // nil if semantic not available
}

// NewEngine creates a search engine backed by the given store.
// Pass nil embedder/vecStore for keyword-only mode.
func NewEngine(store *storage.Store, embedder *Embedder, vecStore VectorStore) *Engine {
	return &Engine{
		store:    store,
		embedder: embedder,
		vecStore: vecStore,
	}
}

// SemanticAvailable returns true if the engine can perform semantic search.
func (e *Engine) SemanticAvailable() bool {
	return e.embedder != nil && e.vecStore != nil && e.vecStore.Count() > 0
}

// Search executes a search and returns scored results.
func (e *Engine) Search(opts SearchOptions) ([]models.SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Type == "" {
		opts.Type = "all"
	}
	if opts.Mode == "" {
		opts.Mode = string(ModeHybrid)
	}

	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return []models.SearchResult{}, nil
	}

	mode := SearchMode(opts.Mode)

	// Auto-detect mode: if semantic not available, fall back to keyword.
	if mode != ModeKeyword && !e.SemanticAvailable() {
		mode = ModeKeyword
	}

	var results []models.SearchResult
	var err error

	switch mode {
	case ModeKeyword:
		results, err = e.keywordSearch(query, opts)
	case ModeSemantic:
		results, err = e.semanticSearch(query, opts)
	case ModeHybrid:
		results, err = e.hybridSearch(query, opts)
	default:
		results, err = e.keywordSearch(query, opts)
	}
	if err != nil {
		return nil, err
	}

	// Sort by score descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// ─── keyword search (existing logic) ─────────────────────────────────

func (e *Engine) keywordSearch(query string, opts SearchOptions) ([]models.SearchResult, error) {
	var results []models.SearchResult
	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	if opts.Type == "all" || opts.Type == "task" {
		taskResults, err := e.keywordSearchTasks(queryLower, words, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, taskResults...)
	}

	if opts.Type == "all" || opts.Type == "doc" {
		docResults, err := e.keywordSearchDocs(queryLower, words, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, docResults...)
	}

	return results, nil
}

func (e *Engine) keywordSearchTasks(query string, words []string, opts SearchOptions) ([]models.SearchResult, error) {
	tasks, err := e.store.Tasks.List()
	if err != nil {
		return nil, err
	}

	var results []models.SearchResult
	for _, task := range tasks {
		if opts.Status != "" && task.Status != opts.Status {
			continue
		}
		if opts.Priority != "" && task.Priority != opts.Priority {
			continue
		}
		if opts.Assignee != "" && task.Assignee != opts.Assignee {
			continue
		}
		if opts.Label != "" && !containsStr(task.Labels, opts.Label) {
			continue
		}

		score := scoreTask(task, query, words)
		if score <= 0 {
			continue
		}

		snippet := extractSnippet(task.Description, query, 150)
		if snippet == "" {
			snippet = truncateStr(task.Description, 150)
		}

		results = append(results, models.SearchResult{
			Type:      "task",
			ID:        task.ID,
			Title:     task.Title,
			Score:     score,
			Snippet:   snippet,
			Status:    task.Status,
			Priority:  task.Priority,
			MatchedBy: []string{"keyword"},
		})
	}
	return results, nil
}

func (e *Engine) keywordSearchDocs(query string, words []string, opts SearchOptions) ([]models.SearchResult, error) {
	docs, err := e.store.Docs.List()
	if err != nil {
		return nil, err
	}

	var results []models.SearchResult
	for _, doc := range docs {
		if opts.Tag != "" && !containsStr(doc.Tags, opts.Tag) {
			continue
		}

		score := scoreDoc(doc, query, words)
		if score <= 0 {
			continue
		}

		snippet := extractSnippet(doc.Description, query, 150)
		if snippet == "" {
			snippet = truncateStr(doc.Description, 150)
		}

		results = append(results, models.SearchResult{
			Type:      "doc",
			ID:        doc.Path,
			Title:     doc.Title,
			Score:     score,
			Snippet:   snippet,
			Path:      doc.Path,
			Tags:      doc.Tags,
			MatchedBy: []string{"keyword"},
		})
	}
	return results, nil
}

// ─── semantic search ─────────────────────────────────────────────────

func (e *Engine) semanticSearch(query string, opts SearchOptions) ([]models.SearchResult, error) {
	queryVec, err := e.embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	scored := e.vecStore.Search(queryVec, VectorSearchOpts{
		TopK:      opts.Limit * 2, // get more to allow filtering
		Threshold: 0.3,
	})

	return e.scoredChunksToResults(scored, opts, "semantic")
}

// ─── hybrid search ───────────────────────────────────────────────────

func (e *Engine) hybridSearch(query string, opts SearchOptions) ([]models.SearchResult, error) {
	// Run both in sequence (could be parallel, but for simplicity).
	kwResults, err := e.keywordSearch(query, opts)
	if err != nil {
		return nil, err
	}

	semResults, err := e.semanticSearch(query, opts)
	if err != nil {
		// Fall back to keyword only.
		return kwResults, nil
	}

	// Merge results.
	return mergeResults(kwResults, semResults, opts.Limit), nil
}

// mergeResults combines keyword and semantic results with weighted scores.
func mergeResults(kwResults, semResults []models.SearchResult, limit int) []models.SearchResult {
	type mergedItem struct {
		result    models.SearchResult
		kwScore   float64
		semScore  float64
		matchedBy []string
	}

	// Index by unique key (type:id).
	merged := make(map[string]*mergedItem)

	// Normalize keyword scores.
	kwMax := 0.0
	for _, r := range kwResults {
		if r.Score > kwMax {
			kwMax = r.Score
		}
	}

	for _, r := range kwResults {
		key := r.Type + ":" + r.ID
		normScore := 0.0
		if kwMax > 0 {
			normScore = r.Score / kwMax
		}
		merged[key] = &mergedItem{
			result:    r,
			kwScore:   normScore,
			matchedBy: []string{"keyword"},
		}
	}

	// Normalize semantic scores (already 0-1 from cosine similarity).
	for _, r := range semResults {
		key := r.Type + ":" + r.ID
		if item, ok := merged[key]; ok {
			item.semScore = r.Score
			item.matchedBy = []string{"semantic", "keyword"}
		} else {
			merged[key] = &mergedItem{
				result:    r,
				semScore:  r.Score,
				matchedBy: []string{"semantic"},
			}
		}
	}

	// Compute final scores.
	var results []models.SearchResult
	for _, item := range merged {
		finalScore := item.semScore*semanticWeight + item.kwScore*keywordWeight
		item.result.Score = finalScore
		item.result.MatchedBy = item.matchedBy
		results = append(results, item.result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// scoredChunksToResults converts vector search results back to SearchResults.
func (e *Engine) scoredChunksToResults(scored []ScoredChunk, opts SearchOptions, method string) ([]models.SearchResult, error) {
	// Group chunks by source (task ID or doc path) and take the best score.
	type sourceResult struct {
		result models.SearchResult
		score  float64
	}
	seen := make(map[string]*sourceResult)

	for _, sc := range scored {
		var key string
		var result models.SearchResult

		switch sc.Type {
		case ChunkTypeTask:
			key = "task:" + sc.TaskID

			// Apply filters.
			if opts.Status != "" && sc.Status != opts.Status {
				continue
			}
			if opts.Priority != "" && sc.Priority != opts.Priority {
				continue
			}
			if opts.Label != "" && !containsStr(sc.Labels, opts.Label) {
				continue
			}
			if opts.Type == "doc" {
				continue
			}

			// Fetch task title for display.
			title := sc.TaskID
			if task, err := e.store.Tasks.Get(sc.TaskID); err == nil {
				title = task.Title
				result.Status = task.Status
				result.Priority = task.Priority
			}

			result = models.SearchResult{
				Type:      "task",
				ID:        sc.TaskID,
				Title:     title,
				Score:     sc.Score,
				Status:    result.Status,
				Priority:  result.Priority,
				MatchedBy: []string{method},
			}

		case ChunkTypeDoc:
			key = "doc:" + sc.DocPath

			if opts.Tag != "" {
				if doc, err := e.store.Docs.Get(sc.DocPath); err != nil || !containsStr(doc.Tags, opts.Tag) {
					continue
				}
			}
			if opts.Type == "task" {
				continue
			}

			title := sc.DocPath
			var tags []string
			if doc, err := e.store.Docs.Get(sc.DocPath); err == nil {
				title = doc.Title
				tags = doc.Tags
			}

			result = models.SearchResult{
				Type:      "doc",
				ID:        sc.DocPath,
				Title:     title,
				Score:     sc.Score,
				Path:      sc.DocPath,
				Tags:      tags,
				Snippet:   sc.Section,
				MatchedBy: []string{method},
			}

		default:
			continue
		}

		if existing, ok := seen[key]; ok {
			if sc.Score > existing.score {
				existing.result = result
				existing.score = sc.Score
			}
		} else {
			seen[key] = &sourceResult{result: result, score: sc.Score}
		}
	}

	results := make([]models.SearchResult, 0, len(seen))
	for _, sr := range seen {
		results = append(results, sr.result)
	}
	return results, nil
}

// ─── scoring helpers (unchanged from original) ──────────────────────

func scoreTask(task *models.Task, query string, words []string) float64 {
	score := 0.0
	titleLower := strings.ToLower(task.Title)
	idLower := strings.ToLower(task.ID)
	descLower := strings.ToLower(task.Description)
	planLower := strings.ToLower(task.ImplementationPlan)
	notesLower := strings.ToLower(task.ImplementationNotes)

	if titleLower == query {
		score += 100
	} else if strings.Contains(titleLower, query) {
		score += 50
	}
	if strings.Contains(idLower, query) {
		score += 30
	}
	if strings.Contains(descLower, query) {
		score += 20
	}
	if strings.Contains(planLower, query) {
		score += 15
	}
	if strings.Contains(notesLower, query) {
		score += 15
	}
	for _, label := range task.Labels {
		if strings.Contains(strings.ToLower(label), query) {
			score += 10
		}
	}
	for _, ac := range task.AcceptanceCriteria {
		if strings.Contains(strings.ToLower(ac.Text), query) {
			score += 5
		}
	}
	if len(words) > 1 {
		matchCount := 0
		for _, w := range words {
			if strings.Contains(titleLower, w) || strings.Contains(descLower, w) {
				matchCount++
			}
		}
		if matchCount > 0 {
			score += float64(matchCount) / float64(len(words)) * 20
		}
	}
	return score
}

func scoreDoc(doc *models.Doc, query string, words []string) float64 {
	score := 0.0
	titleLower := strings.ToLower(doc.Title)
	descLower := strings.ToLower(doc.Description)
	contentLower := strings.ToLower(doc.Content)
	pathLower := strings.ToLower(doc.Path)

	if titleLower == query {
		score += 100
	} else if strings.Contains(titleLower, query) {
		score += 50
	}
	if strings.Contains(pathLower, query) {
		score += 25
	}
	if strings.Contains(descLower, query) {
		score += 20
	}
	if strings.Contains(contentLower, query) {
		score += 15
	}
	for _, tag := range doc.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			score += 10
		}
	}
	if len(words) > 1 {
		matchCount := 0
		for _, w := range words {
			if strings.Contains(titleLower, w) || strings.Contains(contentLower, w) {
				matchCount++
			}
		}
		if matchCount > 0 {
			score += float64(matchCount) / float64(len(words)) * 20
		}
	}
	return score
}

func extractSnippet(text, query string, maxLen int) string {
	if text == "" || query == "" {
		return ""
	}
	lower := strings.ToLower(text)
	idx := strings.Index(lower, query)
	if idx < 0 {
		return ""
	}
	start := int(math.Max(0, float64(idx-40)))
	end := int(math.Min(float64(len(text)), float64(idx+len(query)+maxLen-40)))
	snippet := text[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}
	return snippet
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Reindex rebuilds the search index using the index service.
// Returns an error if semantic search is not available.
func (e *Engine) Reindex(progress ReindexProgress) error {
	if e.embedder == nil || e.vecStore == nil {
		return fmt.Errorf("semantic search is not available (no ONNX Runtime)")
	}
	svc := NewIndexService(e.store, e.embedder, e.vecStore)
	return svc.Reindex(progress)
}
