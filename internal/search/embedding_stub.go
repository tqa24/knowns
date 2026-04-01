//go:build !cgo

package search

import "fmt"

// Embedder produces embedding vectors from text using an ONNX model.
type Embedder struct {
	dimensions  int
	modelConfig EmbeddingModelConfig
}

// EmbedderConfig specifies how to create an Embedder.
type EmbedderConfig struct {
	ModelDir   string
	ModelName  string
	Dimensions int
	MaxTokens  int
	LibPath    string
}

func NewEmbedder(cfg EmbedderConfig) (*Embedder, error) {
	return nil, fmt.Errorf("%w", ErrSemanticRuntimeUnavailable)
}

func (e *Embedder) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("%w", ErrSemanticRuntimeUnavailable)
}

func (e *Embedder) Close() {}

func (e *Embedder) Dimensions() int {
	if e == nil {
		return 0
	}
	return e.dimensions
}

// EmbedQuery embeds text with the model's query prefix prepended.
func (e *Embedder) EmbedQuery(text string) ([]float32, error) {
	return nil, fmt.Errorf("%w", ErrSemanticRuntimeUnavailable)
}

// EmbedDocument embeds text with the model's document prefix prepended.
func (e *Embedder) EmbedDocument(text string) ([]float32, error) {
	return nil, fmt.Errorf("%w", ErrSemanticRuntimeUnavailable)
}

// ModelConfig returns the embedding model configuration.
func (e *Embedder) ModelConfig() EmbeddingModelConfig {
	if e == nil {
		return EmbeddingModelConfig{}
	}
	return e.modelConfig
}

// GetTokenizer returns nil for stub builds (no tokenizer available).
func (e *Embedder) GetTokenizer() Tokenizer {
	return nil
}
