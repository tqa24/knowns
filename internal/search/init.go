package search

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/howznguyen/knowns/internal/storage"
)

// ErrSemanticNotConfigured is returned when semantic search is not enabled in config.
var ErrSemanticNotConfigured = fmt.Errorf("semantic search not configured or disabled")

// InitSemantic attempts to initialize semantic search components.
// Returns a descriptive error if initialization fails at any step.
// If the index is outdated (model or chunk version changed), it auto-reindexes.
// On success, the caller is responsible for calling vecStore.Close() and
// embedder.Close() when done.
func InitSemantic(store *storage.Store) (*Embedder, VectorStore, error) {
	cfg, err := store.Config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		return nil, nil, ErrSemanticNotConfigured
	}

	ss := cfg.Settings.SemanticSearch
	if ss == nil || !ss.Enabled || ss.Model == "" {
		return nil, nil, ErrSemanticNotConfigured
	}

	modelConfig, ok := EmbeddingModels[ss.Model]
	if !ok {
		return nil, nil, fmt.Errorf("unknown embedding model %q", ss.Model)
	}

	home, _ := os.UserHomeDir()
	modelDir := filepath.Join(home, ".knowns", "models", modelConfig.HuggingFaceID)

	// Check model is installed.
	onnxPath := filepath.Join(modelDir, "onnx", "model_quantized.onnx")
	if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
		onnxPath = filepath.Join(modelDir, "onnx", "model.onnx")
		if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("embedding model %q not downloaded (run: knowns model download %s)", ss.Model, ss.Model)
		}
	}

	dims := ss.Dimensions
	if dims <= 0 {
		dims = modelConfig.Dimensions
	}
	maxTokens := ss.MaxTokens
	if maxTokens <= 0 {
		maxTokens = modelConfig.MaxTokens
	}

	embedder, err := NewEmbedder(EmbedderConfig{
		ModelDir:   modelDir,
		ModelName:  ss.Model,
		Dimensions: dims,
		MaxTokens:  maxTokens,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("init embedder: %w", err)
	}

	searchDir := filepath.Join(store.Root, ".search")
	vecStore := NewSQLiteVectorStore(searchDir, ss.Model, dims)
	if err := vecStore.Load(); err != nil {
		embedder.Close()
		return nil, nil, fmt.Errorf("load vector store: %w", err)
	}

	// Auto-reindex if model or chunk version changed.
	if vecStore.NeedsRebuild(ss.Model) && vecStore.Count() > 0 {
		svc := NewIndexService(store, embedder, vecStore)
		if err := svc.Reindex(nil); err != nil {
			// Non-fatal: log but continue with stale index.
			fmt.Fprintf(os.Stderr, "warning: auto-reindex failed: %v\n", err)
		}
	}

	return embedder, vecStore, nil
}
