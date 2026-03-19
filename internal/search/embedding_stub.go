//go:build !cgo

package search

import "fmt"

// Embedder produces embedding vectors from text using an ONNX model.
type Embedder struct {
	dimensions int
}

// EmbedderConfig specifies how to create an Embedder.
type EmbedderConfig struct {
	ModelDir   string
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
