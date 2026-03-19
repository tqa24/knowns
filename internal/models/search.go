package models

// SearchResult is a single hit returned by the search engine, covering both
// tasks and documentation files.
type SearchResult struct {
	// Type is either "task" or "doc".
	Type string `json:"type"`

	// ID is the task ID for task results, or the doc path for doc results.
	ID string `json:"id"`

	Title string `json:"title"`

	// Score is the relevance score (higher = more relevant).
	Score float64 `json:"score"`

	// Snippet is an optional excerpt of matching text.
	Snippet string `json:"snippet,omitempty"`

	// MatchedBy indicates which methods found this result.
	// Possible values: "keyword", "semantic".
	MatchedBy []string `json:"matchedBy,omitempty"`

	// Task-specific fields (populated when Type == "task").
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`

	// Doc-specific fields (populated when Type == "doc").
	Path string   `json:"path,omitempty"`
	Tags []string `json:"tags,omitempty"`
}
