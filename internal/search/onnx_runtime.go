package search

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
)

// ORTModelConfig describes an ONNX embedding model for the native runtime.
type ORTModelConfig struct {
	Name          string `json:"name"`
	HuggingFaceID string `json:"huggingFaceId"`
	Dimensions    int    `json:"dimensions"`
	MaxTokens     int    `json:"maxTokens"`
	QueryPrefix   string `json:"queryPrefix,omitempty"`
	DocPrefix     string `json:"docPrefix,omitempty"`
}

// ORTRuntime manages an ONNX Runtime session for embedding inference.
type ORTRuntime struct {
	tokenizer   Tokenizer
	session     *ort.DynamicAdvancedSession
	inputNames  []string
	outputNames []string
	outputIndex int
	padID       int64
	modelPath   string
	modelDir    string
	model       ORTModelConfig
}

// InitORT initializes the ONNX Runtime with the given model configuration.
func (r *ORTRuntime) InitORT(cfg ORTModelConfig, cacheDir string) error {
	modelDir, modelPath, err := resolveModelArtifacts(cacheDir, cfg.HuggingFaceID)
	if err != nil {
		return err
	}
	if err := ensureORTEnvironment(); err != nil {
		return err
	}

	padID, err := readPadTokenID(modelDir)
	if err != nil {
		return err
	}
	tokenizer, err := LoadTokenizer(modelDir)
	if err != nil {
		return err
	}
	inputInfo, outputInfo, err := ort.GetInputOutputInfo(modelPath)
	if err != nil {
		return fmt.Errorf("inspect model IO: %w", err)
	}

	inputNames := make([]string, len(inputInfo))
	for i, info := range inputInfo {
		inputNames[i] = info.Name
	}
	outputNames := make([]string, len(outputInfo))
	for i, info := range outputInfo {
		outputNames[i] = info.Name
	}
	outputIndex := ortSelectOutputIndex(outputInfo)

	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return fmt.Errorf("open model session: %w", err)
	}

	r.CloseModel()
	r.tokenizer = tokenizer
	r.session = session
	r.inputNames = inputNames
	r.outputNames = outputNames
	r.outputIndex = outputIndex
	r.padID = padID
	r.modelPath = modelPath
	r.modelDir = modelDir
	r.model = cfg
	return nil
}

// EmbedORT runs inference on the given texts and returns embedding vectors.
func (r *ORTRuntime) EmbedORT(texts []string, kind string) ([][]float32, error) {
	if r.session == nil || r.tokenizer == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	prefix := r.model.DocPrefix
	if strings.EqualFold(kind, "query") {
		prefix = r.model.QueryPrefix
	}

	encoded := make([]TokenizerOutput, len(texts))
	maxSeq := 0
	for i, text := range texts {
		input := text
		if prefix != "" {
			input = prefix + text
		}
		encoded[i] = r.tokenizer.Encode(input, r.model.MaxTokens)
		if n := len(encoded[i].InputIDs); n > maxSeq {
			maxSeq = n
		}
	}
	if maxSeq == 0 {
		return nil, fmt.Errorf("tokenizer returned empty sequences")
	}

	shape := ort.Shape{int64(len(texts)), int64(maxSeq)}
	inputIDs := make([]int64, len(texts)*maxSeq)
	attentionMask := make([]int64, len(texts)*maxSeq)
	tokenTypeIDs := make([]int64, len(texts)*maxSeq)
	for i := range inputIDs {
		inputIDs[i] = r.padID
	}
	for row, out := range encoded {
		base := row * maxSeq
		copy(inputIDs[base:base+len(out.InputIDs)], out.InputIDs)
		copy(attentionMask[base:base+len(out.AttentionMask)], out.AttentionMask)
		copy(tokenTypeIDs[base:base+len(out.TokenTypeIDs)], out.TokenTypeIDs)
	}

	inputs := make([]ort.Value, 0, len(r.inputNames))
	for _, name := range r.inputNames {
		v, err := ortBuildInputValue(name, shape, inputIDs, attentionMask, tokenTypeIDs)
		if err != nil {
			ortDestroyValues(inputs)
			return nil, err
		}
		inputs = append(inputs, v)
	}
	defer ortDestroyValues(inputs)

	outputs := make([]ort.Value, len(r.outputNames))
	if err := r.session.Run(inputs, outputs); err != nil {
		ortDestroyValues(outputs)
		return nil, fmt.Errorf("run model: %w", err)
	}
	defer ortDestroyValues(outputs)

	vectors, err := ortExtractVectors(outputs[r.outputIndex], attentionMask, len(texts), maxSeq, r.model.Dimensions)
	if err != nil {
		return nil, err
	}
	return vectors, nil
}

// Dimensions returns the model's embedding dimensionality.
func (r *ORTRuntime) Dimensions() int {
	return r.model.Dimensions
}

// CloseModel releases the current model session without destroying the environment.
func (r *ORTRuntime) CloseModel() {
	if r.session != nil {
		_ = r.session.Destroy()
		r.session = nil
	}
	r.tokenizer = nil
	r.inputNames = nil
	r.outputNames = nil
	r.outputIndex = 0
	r.padID = 0
	r.modelPath = ""
	r.modelDir = ""
	r.model = ORTModelConfig{}
}

// Close releases all ONNX Runtime resources including the global environment.
func (r *ORTRuntime) Close() {
	r.CloseModel()
	if ort.IsInitialized() {
		_ = ort.DestroyEnvironment()
	}
}

// --- environment & library resolution ---

func ensureORTEnvironment() error {
	if ort.IsInitialized() {
		return nil
	}
	lib := ResolveORTLibraryPath()
	if lib != "" {
		ort.SetSharedLibraryPath(lib)
	} else {
		libName := ortSharedLibName()
		fmt.Fprintf(os.Stderr, "warning: bundled %s not found next to executable or in ~/.knowns/bin; falling back to system search which may load an incompatible version\n", libName)
	}
	if err := ort.InitializeEnvironment(); err != nil {
		hint := ""
		if lib == "" {
			hint = fmt.Sprintf(" (no bundled %s was found — a system copy may have been loaded with an incompatible version; reinstall knowns or set KNOWNS_ORT_LIB to the correct path)", ortSharedLibName())
		}
		return fmt.Errorf("initialize onnxruntime: %w%s", err, hint)
	}
	return nil
}

func ortSharedLibName() string {
	switch runtime.GOOS {
	case "windows":
		return "onnxruntime.dll"
	case "darwin":
		return "libonnxruntime.dylib"
	default:
		return "libonnxruntime.so"
	}
}

// ResolveORTLibraryPath locates the ONNX Runtime shared library.
func ResolveORTLibraryPath() string {
	if p := os.Getenv("KNOWNS_ORT_LIB"); p != "" {
		return p
	}
	if p := os.Getenv("KNOWNS_ORT_DLL"); p != "" {
		return p
	}

	name := ortSharedLibName()

	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, name)
		if ortIsFile(candidate) {
			return candidate
		}
		if runtime.GOOS == "linux" {
			if matches, _ := filepath.Glob(filepath.Join(dir, "libonnxruntime.so*")); len(matches) > 0 {
				return matches[0]
			}
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".knowns", "bin", name)
		if ortIsFile(candidate) {
			return candidate
		}
		if runtime.GOOS == "linux" {
			if matches, _ := filepath.Glob(filepath.Join(home, ".knowns", "bin", "libonnxruntime.so*")); len(matches) > 0 {
				return matches[0]
			}
		}
	}

	return ""
}

// --- model artifact resolution ---

func resolveModelArtifacts(baseDir, huggingFaceID string) (string, string, error) {
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 4)
	addCandidate := func(dir string) {
		if dir == "" {
			return
		}
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		candidates = append(candidates, dir)
	}

	addCandidate(baseDir)
	if baseDir != "" {
		addCandidate(filepath.Join(baseDir, filepath.FromSlash(huggingFaceID)))
	}
	if home, err := os.UserHomeDir(); err == nil {
		root := filepath.Join(home, ".knowns", "models")
		addCandidate(filepath.Join(root, filepath.FromSlash(huggingFaceID)))
	}

	for _, dir := range candidates {
		if !ortIsFile(filepath.Join(dir, "tokenizer.json")) {
			continue
		}
		if modelPath := resolveONNXPath(dir); modelPath != "" {
			return dir, modelPath, nil
		}
	}

	return "", "", fmt.Errorf("embedding model %q not found in %q", huggingFaceID, baseDir)
}

func resolveONNXPath(modelDir string) string {
	for _, candidate := range []string{
		filepath.Join(modelDir, "onnx", "model_quantized.onnx"),
		filepath.Join(modelDir, "onnx", "model.onnx"),
	} {
		if ortIsFile(candidate) {
			return candidate
		}
	}
	return ""
}

// --- tensor helpers ---

func ortBuildInputValue(name string, shape ort.Shape, inputIDs, attentionMask, tokenTypeIDs []int64) (ort.Value, error) {
	switch strings.ToLower(name) {
	case "input_ids":
		return ort.NewTensor(shape, inputIDs)
	case "attention_mask":
		return ort.NewTensor(shape, attentionMask)
	case "token_type_ids":
		return ort.NewTensor(shape, tokenTypeIDs)
	default:
		return nil, fmt.Errorf("unsupported model input %q", name)
	}
}

func ortExtractVectors(output ort.Value, attentionMask []int64, batchSize, seqLen, fallbackDims int) ([][]float32, error) {
	tensor, ok := output.(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output tensor type %T", output)
	}
	shape := tensor.GetShape()
	data := tensor.GetData()
	if len(shape) == 2 {
		return ortReshapeAndNormalize(data, batchSize, int(shape[1])), nil
	}
	if len(shape) != 3 {
		return nil, fmt.Errorf("unexpected output shape %v", shape)
	}
	batch := int(shape[0])
	steps := int(shape[1])
	dims := int(shape[2])
	if batch != batchSize {
		return nil, fmt.Errorf("unexpected batch size %d", batch)
	}
	if dims <= 0 {
		dims = fallbackDims
	}
	vectors := make([][]float32, batch)
	for row := 0; row < batch; row++ {
		vec := make([]float32, dims)
		var count float32
		for col := 0; col < steps; col++ {
			mask := float32(0)
			if col < seqLen && attentionMask[row*seqLen+col] != 0 {
				mask = 1
			}
			if mask == 0 {
				continue
			}
			count += mask
			offset := (row*steps + col) * dims
			for d := 0; d < dims; d++ {
				vec[d] += data[offset+d]
			}
		}
		if count > 0 {
			for d := 0; d < dims; d++ {
				vec[d] /= count
			}
		}
		ortNormalize(vec)
		vectors[row] = vec
	}
	return vectors, nil
}

func ortReshapeAndNormalize(data []float32, batchSize, dims int) [][]float32 {
	vectors := make([][]float32, batchSize)
	for row := 0; row < batchSize; row++ {
		start := row * dims
		vec := slices.Clone(data[start : start+dims])
		ortNormalize(vec)
		vectors[row] = vec
	}
	return vectors
}

func ortNormalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v * v)
	}
	if sum == 0 {
		return
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= inv
	}
}

func ortSelectOutputIndex(outputs []ort.InputOutputInfo) int {
	for i, info := range outputs {
		if strings.EqualFold(info.Name, "last_hidden_state") {
			return i
		}
	}
	for i, info := range outputs {
		if len(info.Dimensions) == 3 {
			return i
		}
	}
	return 0
}

func ortDestroyValues(values []ort.Value) {
	for _, v := range values {
		if v != nil {
			_ = v.Destroy()
		}
	}
}

func readPadTokenID(modelDir string) (int64, error) {
	path := filepath.Join(modelDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read model config: %w", err)
	}
	var cfg struct {
		PadTokenID *int64 `json:"pad_token_id"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, fmt.Errorf("parse model config: %w", err)
	}
	if cfg.PadTokenID == nil {
		return 0, fmt.Errorf("model config missing pad_token_id")
	}
	return *cfg.PadTokenID, nil
}

func ortIsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
