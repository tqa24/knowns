package search

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// Embedder produces embedding vectors from text using an ONNX model.
type Embedder struct {
	tokenizer  Tokenizer
	session    *ort.DynamicAdvancedSession
	dimensions int
	maxTokens  int
	mu         sync.Mutex
}

// EmbedderConfig specifies how to create an Embedder.
type EmbedderConfig struct {
	ModelDir   string // path to model directory (e.g. ~/.knowns/models/Xenova/gte-small)
	Dimensions int
	MaxTokens  int
	LibPath    string // optional explicit path to ONNX Runtime shared library
}

// NewEmbedder creates an Embedder, loading the ONNX model and tokenizer.
// Returns an error if ONNX Runtime is not available or the model can't be loaded.
func NewEmbedder(cfg EmbedderConfig) (*Embedder, error) {
	// 1. Resolve and set the ONNX Runtime library path.
	libPath := cfg.LibPath
	if libPath == "" {
		libPath = findONNXLib()
	}
	if libPath == "" {
		return nil, fmt.Errorf("ONNX Runtime library not found. Install with: knowns search --install-runtime")
	}

	ort.SetSharedLibraryPath(libPath)

	if err := ort.InitializeEnvironment(); err != nil {
		// Already initialized is fine — the exact message varies by library version.
		msg := err.Error()
		if !strings.Contains(msg, "already") {
			return nil, fmt.Errorf("onnx init: %w", err)
		}
	}

	// 2. Load tokenizer.
	tokenizer, err := LoadTokenizer(cfg.ModelDir)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	// 3. Find ONNX model file.
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

	// 4. Create session with dynamic shapes.
	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("session options: %w", err)
	}
	defer opts.Destroy()

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &Embedder{
		tokenizer:  tokenizer,
		session:    session,
		dimensions: cfg.Dimensions,
		maxTokens:  cfg.MaxTokens,
	}, nil
}

// Embed generates an embedding vector for the given text.
func (e *Embedder) Embed(text string) ([]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Tokenize.
	tok := e.tokenizer.Encode(text, e.maxTokens)
	seqLen := int64(len(tok.InputIDs))

	// 2. Create input tensors.
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

	// 3. Create output tensor.
	outputShape := ort.NewShape(1, seqLen, int64(e.dimensions))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, fmt.Errorf("output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	// 4. Run inference.
	inputs := []ort.Value{inputIDsTensor, attMaskTensor, tokenTypeTensor}
	outputs := []ort.Value{outputTensor}

	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("inference: %w", err)
	}

	// 5. Mean pooling over non-padding tokens.
	outputData := outputTensor.GetData()
	embedding := meanPool(outputData, tok.AttentionMask, int(seqLen), e.dimensions)

	// 6. L2 normalize.
	NormalizeL2(embedding)

	return embedding, nil
}

// Close releases ONNX resources.
func (e *Embedder) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.session != nil {
		e.session.Destroy()
		e.session = nil
	}
}

// Dimensions returns the embedding vector size.
func (e *Embedder) Dimensions() int {
	return e.dimensions
}

// meanPool computes the mean of token embeddings, ignoring padding tokens.
// outputData shape: [1, seqLen, dims], attMask shape: [seqLen].
func meanPool(outputData []float32, attentionMask []int64, seqLen, dims int) []float32 {
	result := make([]float32, dims)
	count := float32(0)

	for t := 0; t < seqLen; t++ {
		if attentionMask[t] == 0 {
			continue
		}
		count++
		offset := t * dims
		for d := 0; d < dims; d++ {
			result[d] += outputData[offset+d]
		}
	}

	if count > 0 {
		for d := range result {
			result[d] /= count
		}
	}

	return result
}

// findONNXLib searches for the ONNX Runtime shared library in standard locations.
func findONNXLib() string {
	// 1. Environment variable.
	if p := os.Getenv("KNOWNS_ONNX_LIB"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	home, _ := os.UserHomeDir()
	libName := onnxLibName()

	// 2. ~/.knowns/lib/
	if home != "" {
		p := filepath.Join(home, ".knowns", "lib", libName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 3. System paths.
	systemPaths := onnxSystemPaths()
	for _, dir := range systemPaths {
		p := filepath.Join(dir, libName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// onnxLibName returns the platform-specific library filename.
func onnxLibName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	default:
		return "libonnxruntime.so"
	}
}

// onnxSystemPaths returns common system library directories.
func onnxSystemPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/usr/local/lib", "/opt/homebrew/lib"}
	case "linux":
		return []string{"/usr/local/lib", "/usr/lib", "/usr/lib/x86_64-linux-gnu", "/usr/lib/aarch64-linux-gnu"}
	case "windows":
		return []string{`C:\Program Files\onnxruntime\lib`}
	}
	return nil
}

// IsONNXAvailable checks if ONNX Runtime is available on the system.
func IsONNXAvailable() (bool, string) {
	path := findONNXLib()
	return path != "", path
}

// ONNXRuntimeDownloadURL returns the download URL for the current platform.
func ONNXRuntimeDownloadURL() (string, string, error) {
	const version = "1.24.3"
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var platform, ext string
	switch {
	case goos == "darwin" && goarch == "arm64":
		platform = "osx-arm64"
		ext = "tgz"
	case goos == "darwin" && goarch == "amd64":
		// macOS x86_64 not provided in recent releases; use arm64 with Rosetta
		platform = "osx-arm64"
		ext = "tgz"
	case goos == "linux" && goarch == "amd64":
		platform = "linux-x64"
		ext = "tgz"
	case goos == "linux" && goarch == "arm64":
		platform = "linux-aarch64"
		ext = "tgz"
	case goos == "windows" && goarch == "amd64":
		platform = "win-x64"
		ext = "zip"
	case goos == "windows" && goarch == "arm64":
		platform = "win-arm64"
		ext = "zip"
	default:
		return "", "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
	}

	url := fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-%s-%s.%s",
		version, platform, version, ext)
	libName := onnxLibName()

	return url, libName, nil
}
