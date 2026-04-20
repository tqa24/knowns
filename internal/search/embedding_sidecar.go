package search

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Embedder produces embedding vectors via an out-of-process sidecar (Bun binary)
// that runs transformers.js. It speaks JSON-RPC 2.0 over stdio.
type Embedder struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      *bufio.Reader
	stderr      io.ReadCloser
	dimensions  int
	modelConfig EmbeddingModelConfig

	mu     sync.Mutex
	nextID atomic.Int64
	closed atomic.Bool
}

// EmbedderConfig specifies how to create an Embedder.
type EmbedderConfig struct {
	ModelDir   string // optional; passed as transformers.js cacheDir
	ModelName  string // key into EmbeddingModels
	Dimensions int
	MaxTokens  int
	LibPath    string // unused for sidecar; kept for source compat
}

type rpcReq struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcErr struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

type sidecarModelParams struct {
	Name          string `json:"name"`
	HuggingFaceID string `json:"huggingFaceId"`
	Dimensions    int    `json:"dimensions"`
	MaxTokens     int    `json:"maxTokens"`
	QueryPrefix   string `json:"queryPrefix,omitempty"`
	DocPrefix     string `json:"docPrefix,omitempty"`
}

type initParams struct {
	Model    sidecarModelParams `json:"model"`
	CacheDir string             `json:"cacheDir,omitempty"`
}

type initResult struct {
	Dimensions int `json:"dimensions"`
}

type embedParams struct {
	Texts []string `json:"texts"`
	Kind  string   `json:"kind"`
}

type embedResult struct {
	Vectors [][]float32 `json:"vectors"`
}

// NewEmbedder spawns the sidecar process and initializes it for the given model.
func NewEmbedder(cfg EmbedderConfig) (*Embedder, error) {
	modelCfg, ok := EmbeddingModels[cfg.ModelName]
	if !ok {
		return nil, fmt.Errorf("unknown embedding model %q", cfg.ModelName)
	}

	bin, err := findSidecarBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(bin)
	cmd.Env = sidecarEnv(bin)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("sidecar stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("sidecar stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("sidecar stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("sidecar start: %w", err)
	}

	e := &Embedder{
		cmd:         cmd,
		stdin:       stdin,
		stdout:      bufio.NewReader(stdout),
		stderr:      stderr,
		dimensions:  cfg.Dimensions,
		modelConfig: modelCfg,
	}

	go e.drainStderr()

	cacheDir := cfg.ModelDir
	if cacheDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cacheDir = filepath.Join(home, ".knowns", "models")
		}
	}

	dims := cfg.Dimensions
	if dims <= 0 {
		dims = modelCfg.Dimensions
	}
	maxTok := cfg.MaxTokens
	if maxTok <= 0 {
		maxTok = modelCfg.MaxTokens
	}

	var res initResult
	if err := e.callWithTimeout(120*time.Second, "init", initParams{
		Model: sidecarModelParams{
			Name:          modelCfg.Name,
			HuggingFaceID: modelCfg.HuggingFaceID,
			Dimensions:    dims,
			MaxTokens:     maxTok,
			QueryPrefix:   modelCfg.QueryPrefix,
			DocPrefix:     modelCfg.DocPrefix,
		},
		CacheDir: cacheDir,
	}, &res); err != nil {
		e.Close()
		return nil, fmt.Errorf("sidecar init: %w", err)
	}
	if res.Dimensions > 0 {
		e.dimensions = res.Dimensions
	}
	return e, nil
}

func (e *Embedder) drainStderr() {
	if e.stderr == nil {
		return
	}
	sc := bufio.NewScanner(e.stderr)
	for sc.Scan() {
		fmt.Fprintf(os.Stderr, "[knowns-embed] %s\n", sc.Text())
	}
}

func (e *Embedder) call(method string, params interface{}, out interface{}) error {
	return e.callWithTimeout(60*time.Second, method, params, out)
}

func (e *Embedder) callWithTimeout(timeout time.Duration, method string, params interface{}, out interface{}) error {
	if e.closed.Load() {
		return fmt.Errorf("sidecar closed")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	id := e.nextID.Add(1)
	req := rpcReq{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	buf = append(buf, '\n')

	if _, err := e.stdin.Write(buf); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	type readResult struct {
		line []byte
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		line, err := e.stdout.ReadBytes('\n')
		ch <- readResult{line, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return fmt.Errorf("read response: %w", r.err)
		}
		var resp rpcResp
		if err := json.Unmarshal(r.line, &resp); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
		if resp.Error != nil {
			return fmt.Errorf("sidecar error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("decode result: %w", err)
			}
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("sidecar timeout after %s", timeout)
	}
}

// Embed embeds a single text as a document.
func (e *Embedder) Embed(text string) ([]float32, error) {
	return e.EmbedDocument(text)
}

func (e *Embedder) EmbedDocument(text string) ([]float32, error) {
	vs, err := e.embedBatch([]string{text}, "doc")
	if err != nil {
		return nil, err
	}
	return vs[0], nil
}

func (e *Embedder) EmbedQuery(text string) ([]float32, error) {
	vs, err := e.embedBatch([]string{text}, "query")
	if err != nil {
		return nil, err
	}
	return vs[0], nil
}

func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	return e.embedBatch(texts, "doc")
}

func (e *Embedder) EmbedDocumentBatch(texts []string) ([][]float32, error) {
	return e.embedBatch(texts, "doc")
}

func (e *Embedder) EmbedQueryBatch(texts []string) ([][]float32, error) {
	return e.embedBatch(texts, "query")
}

func (e *Embedder) embedBatch(texts []string, kind string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	var res embedResult
	if err := e.call("embed", embedParams{Texts: texts, Kind: kind}, &res); err != nil {
		return nil, err
	}
	if len(res.Vectors) != len(texts) {
		return nil, fmt.Errorf("sidecar returned %d vectors for %d texts", len(res.Vectors), len(texts))
	}
	return res.Vectors, nil
}

func (e *Embedder) Dimensions() int {
	if e == nil {
		return 0
	}
	return e.dimensions
}

func (e *Embedder) ModelConfig() EmbeddingModelConfig {
	if e == nil {
		return EmbeddingModelConfig{}
	}
	return e.modelConfig
}

// GetTokenizer returns nil; tokenization happens inside the sidecar.
func (e *Embedder) GetTokenizer() Tokenizer {
	return nil
}

func (e *Embedder) Close() {
	if e == nil || !e.closed.CompareAndSwap(false, true) {
		return
	}
	if e.stdin != nil {
		req := rpcReq{JSONRPC: "2.0", ID: e.nextID.Add(1), Method: "shutdown"}
		if buf, err := json.Marshal(req); err == nil {
			_, _ = e.stdin.Write(append(buf, '\n'))
		}
		_ = e.stdin.Close()
	}
	done := make(chan error, 1)
	go func() { done <- e.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = e.cmd.Process.Kill()
		<-done
	}
}

// IsSidecarAvailable reports whether the knowns-embed sidecar binary can be found.
func IsSidecarAvailable() (bool, string) {
	p, err := findSidecarBinary()
	if err != nil {
		return false, ""
	}
	return true, p
}

// findSidecarBinary locates the knowns-embed binary. Search order:
//  1. KNOWNS_EMBED_BIN env var
//  2. Same directory as the current executable
//  3. ~/.knowns/bin/
//  4. $PATH
func findSidecarBinary() (string, error) {
	name := "knowns-embed"
	if runtime.GOOS == "windows" {
		name = "knowns-embed.exe"
	}

	if p := os.Getenv("KNOWNS_EMBED_BIN"); p != "" {
		if isExecutable(p) {
			return p, nil
		}
		return "", fmt.Errorf("KNOWNS_EMBED_BIN=%q not executable", p)
	}

	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		candidate := filepath.Join(filepath.Dir(exe), name)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".knowns", "bin", name)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}

	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("knowns-embed sidecar binary not found (looked next to knowns binary, in ~/.knowns/bin, and on PATH)")
}

func isExecutable(p string) bool {
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

// sidecarEnv builds the environment for the sidecar process. The bun-compiled
// binary cannot have its rpath patched (LINKEDIT layout breaks install_name_tool),
// so we point dyld/ld.so at the bundle directory containing the native ONNX libs.
func sidecarEnv(bin string) []string {
	dir := filepath.Dir(bin)
	if real, err := filepath.EvalSymlinks(bin); err == nil {
		dir = filepath.Dir(real)
	}
	env := os.Environ()
	switch runtime.GOOS {
	case "darwin":
		env = appendEnvPath(env, "DYLD_LIBRARY_PATH", dir)
		env = appendEnvPath(env, "DYLD_FALLBACK_LIBRARY_PATH", dir)
	case "linux":
		env = appendEnvPath(env, "LD_LIBRARY_PATH", dir)
	case "windows":
		env = appendEnvPath(env, "PATH", dir)
	}
	return env
}

func appendEnvPath(env []string, key, dir string) []string {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			existing := strings.TrimPrefix(kv, prefix)
			if existing == "" {
				env[i] = prefix + dir
			} else {
				env[i] = prefix + dir + sep + existing
			}
			return env
		}
	}
	return append(env, prefix+dir)
}
