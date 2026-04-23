// knowns-embed is a standalone JSON-RPC sidecar for ONNX embedding inference.
// It is kept for backward compatibility but is no longer required — the knowns
// binary now embeds the ONNX runtime directly.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/howznguyen/knowns/internal/embedsidecar"
	"github.com/howznguyen/knowns/internal/util"
)

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initParams struct {
	Model    embedsidecar.ModelConfig `json:"model"`
	CacheDir string                   `json:"cacheDir,omitempty"`
}

type embedParams struct {
	Texts []string `json:"texts"`
	Kind  string   `json:"kind"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var runtime embedsidecar.Runtime
	defer runtime.Close()

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 8*1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal(line, &req); err != nil {
			if err := encoder.Encode(errorResponse(nil, -32700, "parse error")); err != nil {
				return err
			}
			continue
		}

		resp, exit, err := handle(&runtime, req)
		if err != nil {
			return err
		}
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
		if exit {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func handle(runtime *embedsidecar.Runtime, req rpcReq) (*rpcResp, bool, error) {
	switch req.Method {
	case "ping":
		return successResponse(req.ID, map[string]any{"ok": true, "version": util.Version}), false, nil
	case "init":
		var params initParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid params"), false, nil
		}
		if params.Model.HuggingFaceID == "" {
			return errorResponse(req.ID, -32602, "missing params.model"), false, nil
		}
		if err := runtime.InitORT(params.Model, params.CacheDir); err != nil {
			return errorResponse(req.ID, -32000, err.Error()), false, nil
		}
		return successResponse(req.ID, map[string]any{"dimensions": runtime.Dimensions()}), false, nil
	case "embed":
		var params embedParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid params"), false, nil
		}
		if params.Texts == nil {
			return errorResponse(req.ID, -32602, "missing params.texts"), false, nil
		}
		kind := params.Kind
		if kind == "" {
			kind = "doc"
		}
		vectors, err := runtime.EmbedORT(params.Texts, kind)
		if err != nil {
			return errorResponse(req.ID, -32000, err.Error()), false, nil
		}
		return successResponse(req.ID, map[string]any{"vectors": vectors}), false, nil
	case "shutdown":
		return successResponse(req.ID, map[string]any{"ok": true}), true, nil
	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("unknown method: %s", req.Method)), false, nil
	}
}

func successResponse(id json.RawMessage, result any) *rpcResp {
	return &rpcResp{JSONRPC: "2.0", ID: normalizeID(id), Result: result}
}

func errorResponse(id json.RawMessage, code int, message string) *rpcResp {
	return &rpcResp{JSONRPC: "2.0", ID: normalizeID(id), Error: &rpcErr{Code: code, Message: message}}
}

func normalizeID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}
