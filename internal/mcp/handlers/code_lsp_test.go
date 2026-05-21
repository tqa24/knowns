package handlers

import (
	"strings"
	"testing"

	"github.com/howznguyen/knowns/internal/lsp"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestLSPPathRequestNoProject(t *testing.T) {
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"path": "main.go"}}}
	_, _, _, err := lspPathRequest(t.Context(), func() *storage.Store { return nil }, nil, req)
	if err == nil || !strings.Contains(err.Error(), "no project loaded") {
		t.Fatalf("error = %v", err)
	}
}

func TestLSPSeverity(t *testing.T) {
	cases := map[int]string{1: "error", 2: "warning", 3: "info", 4: "hint", 0: "error"}
	for in, want := range cases {
		if got := lspSeverity(in); got != want {
			t.Fatalf("severity %d = %q, want %q", in, got, want)
		}
	}
}

func TestLSPPathRequestNoManager(t *testing.T) {
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"path": "main.go"}}}
	store := storage.NewStore(t.TempDir())
	_, _, _, err := lspPathRequest(t.Context(), func() *storage.Store { return store }, nil, req)
	if err == nil || !strings.Contains(err.Error(), "LSP not available for this project") {
		t.Fatalf("error = %v", err)
	}
}

func TestLSPPathRequestRequiresPath(t *testing.T) {
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
	store := storage.NewStore(t.TempDir())
	_, _, _, err := lspPathRequest(t.Context(), func() *storage.Store { return store }, func() *lsp.Manager { return nil }, req)
	if err == nil || !strings.Contains(err.Error(), "LSP not available for this project") {
		t.Fatalf("error = %v", err)
	}
}
