package lsp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFindsFilesAndAvailableBinaries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	d := NewDetector(NewRegistry(nil))
	d.LookPath = func(name string) (string, error) {
		if name == "gopls" {
			return "/bin/gopls", nil
		}
		return "", errors.New("missing")
	}
	d.RunCheck = func(context.Context, string, ...string) error { return nil }
	commands, err := d.Detect(context.Background(), root, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Language != "go" || commands[0].Name != "gopls" {
		t.Fatalf("commands = %#v, want gopls go", commands)
	}
}

func TestDetectSilentlySkipsMissingBinaries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	d := NewDetector(NewRegistry(nil))
	d.LookPath = func(string) (string, error) { return "", errors.New("missing") }
	d.RunCheck = func(context.Context, string, ...string) error { t.Fatal("unexpected check"); return nil }
	commands, err := d.Detect(context.Background(), root, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 0 {
		t.Fatalf("commands = %#v, want none", commands)
	}
}

func TestDetectHonorsConfigOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	falseValue := false
	d := NewDetector(NewRegistry(nil))
	d.LookPath = func(string) (string, error) { t.Fatal("unexpected lookpath"); return "", nil }
	commands, err := d.Detect(context.Background(), root, Config{Languages: map[string]LanguageConfig{"go": {Enabled: &falseValue}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 0 {
		t.Fatalf("commands = %#v, want none", commands)
	}
}
