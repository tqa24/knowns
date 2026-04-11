//go:build cgo

package search

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

	if err := opts.SetIntraOpNumThreads(onnxThreadLimit("KNOWNS_ONNX_INTRA_OP_THREADS", 2)); err != nil {
		return nil, fmt.Errorf("set intra-op threads: %w", err)
	}
	if err := opts.SetInterOpNumThreads(onnxThreadLimit("KNOWNS_ONNX_INTER_OP_THREADS", 1)); err != nil {
		return nil, fmt.Errorf("set inter-op threads: %w", err)
	}

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

// EmbedBatch embeds multiple texts in a single ONNX inference call.
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if len(texts) == 1 {
		vec, err := e.Embed(texts[0])
		if err != nil {
			return nil, err
		}
		return [][]float32{vec}, nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Tokenize all texts and find max sequence length.
	tokens := make([]TokenizerOutput, len(texts))
	maxSeqLen := 0
	for i, text := range texts {
		tokens[i] = e.tokenizer.Encode(text, e.maxTokens)
		if n := len(tokens[i].InputIDs); n > maxSeqLen {
			maxSeqLen = n
		}
	}

	// Flatten into padded batch tensors.
	batchSize := int64(len(texts))
	totalLen := int(batchSize) * maxSeqLen
	inputIDs := make([]int64, totalLen)
	attMask := make([]int64, totalLen)
	tokenTypes := make([]int64, totalLen)

	for i, tok := range tokens {
		off := i * maxSeqLen
		copy(inputIDs[off:], tok.InputIDs)
		copy(attMask[off:], tok.AttentionMask)
		copy(tokenTypes[off:], tok.TokenTypeIDs)
	}

	// Create batch tensors with shape (batchSize, maxSeqLen).
	shape := ort.NewShape(batchSize, int64(maxSeqLen))

	inputIDsTensor, err := ort.NewTensor(shape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("batch input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(shape, attMask)
	if err != nil {
		return nil, fmt.Errorf("batch attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	tokenTypeTensor, err := ort.NewTensor(shape, tokenTypes)
	if err != nil {
		return nil, fmt.Errorf("batch token_type_ids tensor: %w", err)
	}
	defer tokenTypeTensor.Destroy()

	outputShape := ort.NewShape(batchSize, int64(maxSeqLen), int64(e.dimensions))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, fmt.Errorf("batch output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	inputs := []ort.Value{inputIDsTensor, attMaskTensor, tokenTypeTensor}
	outputs := []ort.Value{outputTensor}
	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("batch inference: %w", err)
	}

	// Extract per-item embeddings.
	outputData := outputTensor.GetData()
	results := make([][]float32, len(texts))
	itemSize := maxSeqLen * e.dimensions
	for i := range texts {
		itemOutput := outputData[i*itemSize : (i+1)*itemSize]
		itemMask := attMask[i*maxSeqLen : (i+1)*maxSeqLen]
		results[i] = meanPool(itemOutput, itemMask, maxSeqLen, e.dimensions)
		NormalizeL2(results[i])
	}
	return results, nil
}

// EmbedDocumentBatch embeds multiple texts with the document prefix.
func (e *Embedder) EmbedDocumentBatch(texts []string) ([][]float32, error) {
	if e.modelConfig.DocPrefix == "" {
		return e.EmbedBatch(texts)
	}
	prefixed := make([]string, len(texts))
	for i, t := range texts {
		prefixed[i] = e.modelConfig.DocPrefix + t
	}
	return e.EmbedBatch(prefixed)
}

// EmbedQueryBatch embeds multiple texts with the query prefix.
func (e *Embedder) EmbedQueryBatch(texts []string) ([][]float32, error) {
	if e.modelConfig.QueryPrefix == "" {
		return e.EmbedBatch(texts)
	}
	prefixed := make([]string, len(texts))
	for i, t := range texts {
		prefixed[i] = e.modelConfig.QueryPrefix + t
	}
	return e.EmbedBatch(prefixed)
}

// ModelConfig returns the embedding model configuration.
func (e *Embedder) ModelConfig() EmbeddingModelConfig {
	return e.modelConfig
}

// Tokenizer returns the underlying tokenizer, or nil if not available.
func (e *Embedder) GetTokenizer() Tokenizer {
	return e.tokenizer
}

func onnxThreadLimit(envKey string, defaultValue int) int {
	if raw := strings.TrimSpace(os.Getenv(envKey)); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	if defaultValue > 0 {
		maxThreads := runtime.NumCPU()
		if maxThreads > 0 && defaultValue > maxThreads {
			return maxThreads
		}
		return defaultValue
	}
	if n := runtime.NumCPU(); n > 0 {
		return n
	}
	return 1
}
