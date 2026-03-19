package search

// SearchMode determines how results are found.
type SearchMode string

const (
	ModeKeyword  SearchMode = "keyword"
	ModeSemantic SearchMode = "semantic"
	ModeHybrid   SearchMode = "hybrid"
)

// ChunkType indicates whether a chunk came from a doc or a task.
type ChunkType string

const (
	ChunkTypeDoc  ChunkType = "doc"
	ChunkTypeTask ChunkType = "task"
)

// Chunk is a piece of a task or document prepared for embedding.
type Chunk struct {
	ID         string    `json:"id"`
	Type       ChunkType `json:"type"`
	Content    string    `json:"content"`
	TokenCount int       `json:"tokenCount"`
	Embedding  []float32 `json:"-"` // populated after embedding

	// Doc fields (populated when Type == ChunkTypeDoc).
	DocPath       string `json:"docPath,omitempty"`
	Section       string `json:"section,omitempty"`
	HeadingLevel  int    `json:"headingLevel,omitempty"`
	ParentSection string `json:"parentSection,omitempty"`
	Position      int    `json:"position,omitempty"`

	// Task fields (populated when Type == ChunkTypeTask).
	TaskID   string   `json:"taskId,omitempty"`
	Field    string   `json:"field,omitempty"` // "description", "ac", "plan", "notes"
	Status   string   `json:"status,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

// ChunkResult is the output of chunking a single task or doc.
type ChunkResult struct {
	Chunks      []Chunk
	TotalTokens int
}

// ScoredChunk is a chunk with a similarity score from vector search.
type ScoredChunk struct {
	Chunk
	Score float64
}

// VectorSearchOpts controls how vector search behaves.
type VectorSearchOpts struct {
	TopK      int
	Threshold float64 // minimum cosine similarity (0-1)
}

// MatchMethod describes how a result was found.
type MatchMethod string

const (
	MatchKeyword  MatchMethod = "keyword"
	MatchSemantic MatchMethod = "semantic"
)

// EmbeddingModelConfig holds the configuration for a specific embedding model.
type EmbeddingModelConfig struct {
	Name          string
	Dimensions    int
	MaxTokens     int
	HuggingFaceID string
}

// Known embedding model configurations.
var EmbeddingModels = map[string]EmbeddingModelConfig{
	"gte-small": {
		Name:          "gte-small",
		Dimensions:    384,
		MaxTokens:     512,
		HuggingFaceID: "Xenova/gte-small",
	},
	"all-MiniLM-L6-v2": {
		Name:          "all-MiniLM-L6-v2",
		Dimensions:    384,
		MaxTokens:     256,
		HuggingFaceID: "Xenova/all-MiniLM-L6-v2",
	},
	"gte-base": {
		Name:          "gte-base",
		Dimensions:    768,
		MaxTokens:     512,
		HuggingFaceID: "Xenova/gte-base",
	},
	"bge-small-en-v1.5": {
		Name:          "bge-small-en-v1.5",
		Dimensions:    384,
		MaxTokens:     512,
		HuggingFaceID: "Xenova/bge-small-en-v1.5",
	},
	"bge-base-en-v1.5": {
		Name:          "bge-base-en-v1.5",
		Dimensions:    768,
		MaxTokens:     512,
		HuggingFaceID: "Xenova/bge-base-en-v1.5",
	},
	"nomic-embed-text-v1.5": {
		Name:          "nomic-embed-text-v1.5",
		Dimensions:    768,
		MaxTokens:     8192,
		HuggingFaceID: "Xenova/nomic-embed-text-v1.5",
	},
	"multilingual-e5-small": {
		Name:          "multilingual-e5-small",
		Dimensions:    384,
		MaxTokens:     512,
		HuggingFaceID: "Xenova/multilingual-e5-small",
	},
}
