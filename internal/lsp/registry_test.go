package lsp

import "testing"

func TestRegistryForPath(t *testing.T) {
	registry := NewRegistry(nil)
	tests := map[string]string{
		"main.go":   "go",
		"app.ts":    "typescript",
		"view.tsx":  "typescriptreact",
		"script.js": "javascript",
		"view.jsx":  "javascriptreact",
		"test.py":   "python",
		"lib.rs":    "rust",
		"README.md": "",
	}
	for path, want := range tests {
		got, ok := registry.ForPath(path)
		if want == "" {
			if ok {
				t.Fatalf("ForPath(%q) = %q, want none", path, got.ID)
			}
			continue
		}
		if !ok || got.ID != want {
			t.Fatalf("ForPath(%q) = %q %v, want %q", path, got.ID, ok, want)
		}
	}
}
