//go:build windows

package embedsidecar

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/howznguyen/knowns/internal/search"
	ort "github.com/yalue/onnxruntime_go"
)

type ModelConfig struct {
	Name          string `json:"name"`
	HuggingFaceID string `json:"huggingFaceId"`
	Dimensions    int    `json:"dimensions"`
	MaxTokens     int    `json:"maxTokens"`
	QueryPrefix   string `json:"queryPrefix,omitempty"`
	DocPrefix     string `json:"docPrefix,omitempty"`
}

type Runtime struct {
	tokenizer   search.Tokenizer
	session     *ort.DynamicAdvancedSession
	inputNames  []string
	outputNames []string
	outputIndex int
	padID       int64
	modelPath   string
	modelDir    string
	model       ModelConfig
}

func (r *Runtime) Init(cfg ModelConfig, cacheDir string) error {
	modelDir, modelPath, err := resolveModelArtifacts(cacheDir, cfg.HuggingFaceID)
	if err != nil {
		return err
	}
	if err := ensureEnvironment(); err != nil {
		return err
	}

	padID, err := readPadTokenID(modelDir)
	if err != nil {
		return err
	}
	tokenizer, err := search.LoadTokenizer(modelDir)
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
	outputIndex := selectOutputIndex(outputInfo)

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

func (r *Runtime) Embed(texts []string, kind string) ([][]float32, error) {
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

	encoded := make([]search.TokenizerOutput, len(texts))
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
		v, err := buildInputValue(name, shape, inputIDs, attentionMask, tokenTypeIDs)
		if err != nil {
			destroyValues(inputs)
			return nil, err
		}
		inputs = append(inputs, v)
	}
	defer destroyValues(inputs)

	outputs := make([]ort.Value, len(r.outputNames))
	if err := r.session.Run(inputs, outputs); err != nil {
		destroyValues(outputs)
		return nil, fmt.Errorf("run model: %w", err)
	}
	defer destroyValues(outputs)

	vectors, err := extractVectors(outputs[r.outputIndex], attentionMask, len(texts), maxSeq, r.model.Dimensions)
	if err != nil {
		return nil, err
	}
	return vectors, nil
}

func (r *Runtime) Dimensions() int {
	return r.model.Dimensions
}

func (r *Runtime) CloseModel() {
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
	r.model = ModelConfig{}
}

func (r *Runtime) Close() {
	r.CloseModel()
	if ort.IsInitialized() {
		_ = ort.DestroyEnvironment()
	}
}

func ensureEnvironment() error {
	if ort.IsInitialized() {
		return nil
	}
	if dll := resolveSharedLibraryPath(); dll != "" {
		ort.SetSharedLibraryPath(dll)
	}
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("initialize onnxruntime: %w", err)
	}
	return nil
}

func resolveSharedLibraryPath() string {
	if p := os.Getenv("KNOWNS_ORT_DLL"); p != "" {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "onnxruntime.dll")
		if isFile(candidate) {
			return candidate
		}
	}
	return ""
}

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
		if !isFile(filepath.Join(dir, "tokenizer.json")) {
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
		if isFile(candidate) {
			return candidate
		}
	}
	return ""
}

func buildInputValue(name string, shape ort.Shape, inputIDs, attentionMask, tokenTypeIDs []int64) (ort.Value, error) {
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

func extractVectors(output ort.Value, attentionMask []int64, batchSize, seqLen, fallbackDims int) ([][]float32, error) {
	tensor, ok := output.(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output tensor type %T", output)
	}
	shape := tensor.GetShape()
	data := tensor.GetData()
	if len(shape) == 2 {
		return reshapeAndNormalize(data, batchSize, int(shape[1])), nil
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
		normalize(vec)
		vectors[row] = vec
	}
	return vectors, nil
}

func reshapeAndNormalize(data []float32, batchSize, dims int) [][]float32 {
	vectors := make([][]float32, batchSize)
	for row := 0; row < batchSize; row++ {
		start := row * dims
		vec := slices.Clone(data[start : start+dims])
		normalize(vec)
		vectors[row] = vec
	}
	return vectors
}

func normalize(vec []float32) {
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

func selectOutputIndex(outputs []ort.InputOutputInfo) int {
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

func destroyValues(values []ort.Value) {
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

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
