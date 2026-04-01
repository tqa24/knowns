//go:build cgo

package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// Embedder produces embedding vectors from text using an ONNX model.
type Embedder struct {
	tokenizer   Tokenizer
	session     *ort.DynamicAdvancedSession
	dimensions  int
	maxTokens   int
	modelConfig EmbeddingModelConfig
	mu          sync.Mutex
}

// EmbedderConfig specifies how to create an Embedder.
type EmbedderConfig struct {
	ModelDir   string
	ModelName  string // key into EmbeddingModels for prefix lookup
	Dimensions int
	MaxTokens  int
	LibPath    string
}

func NewEmbedder(cfg EmbedderConfig) (*Embedder, error) {
	libPath := cfg.LibPath
	if libPath == "" {
		libPath = findONNXLib()
	}
	if libPath == "" {
		return nil, fmt.Errorf("ONNX Runtime library not found. Install with: knowns search --install-runtime")
	}

	ort.SetSharedLibraryPath(libPath)

	if err := ort.InitializeEnvironment(); err != nil {
		msg := err.Error()
		if !strings.Contains(msg, "already") {
			return nil, fmt.Errorf("onnx init: %w", err)
		}
	}

	tokenizer, err := LoadTokenizer(cfg.ModelDir)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	modelPath := ""
	for _, name := range []string{"onnx/model_quantized.onnx", "onnx/model.onnx"} {
		p := filepath.Join(cfg.ModelDir, name)
		if _, err := os.Stat(p); err == nil {
			modelPath = p
			break
		}
	}
	if modelPath == "" {
		return nil, fmt.Errorf("no ONNX model file found in %s", cfg.ModelDir)
	}

	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("session options: %w", err)
	}
	defer opts.Destroy()

	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, opts)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Resolve model config for prefix support.
	modelCfg := EmbeddingModels[cfg.ModelName] // zero value if not found (no prefixes)

	return &Embedder{
		tokenizer:   tokenizer,
		session:     session,
		dimensions:  cfg.Dimensions,
		maxTokens:   cfg.MaxTokens,
		modelConfig: modelCfg,
	}, nil
}

func (e *Embedder) Embed(text string) ([]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	tok := e.tokenizer.Encode(text, e.maxTokens)
	seqLen := int64(len(tok.InputIDs))
	shape := ort.NewShape(1, seqLen)

	inputIDsTensor, err := ort.NewTensor(shape, tok.InputIDs)
	if err != nil {
		return nil, fmt.Errorf("input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(shape, tok.AttentionMask)
	if err != nil {
		return nil, fmt.Errorf("attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	tokenTypeTensor, err := ort.NewTensor(shape, tok.TokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("token_type_ids tensor: %w", err)
	}
	defer tokenTypeTensor.Destroy()

	outputShape := ort.NewShape(1, seqLen, int64(e.dimensions))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, fmt.Errorf("output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	inputs := []ort.Value{inputIDsTensor, attMaskTensor, tokenTypeTensor}
	outputs := []ort.Value{outputTensor}
	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("inference: %w", err)
	}

	outputData := outputTensor.GetData()
	embedding := meanPool(outputData, tok.AttentionMask, int(seqLen), e.dimensions)
	NormalizeL2(embedding)
	return embedding, nil
}

func (e *Embedder) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.session != nil {
		e.session.Destroy()
		e.session = nil
	}
}

func (e *Embedder) Dimensions() int {
	return e.dimensions
}

// EmbedQuery embeds text with the model's query prefix prepended.
func (e *Embedder) EmbedQuery(text string) ([]float32, error) {
	return e.Embed(e.modelConfig.QueryPrefix + text)
}

// EmbedDocument embeds text with the model's document prefix prepended.
func (e *Embedder) EmbedDocument(text string) ([]float32, error) {
	return e.Embed(e.modelConfig.DocPrefix + text)
}

// ModelConfig returns the embedding model configuration.
func (e *Embedder) ModelConfig() EmbeddingModelConfig {
	return e.modelConfig
}

// Tokenizer returns the underlying tokenizer, or nil if not available.
func (e *Embedder) GetTokenizer() Tokenizer {
	return e.tokenizer
}
