package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateOpenCodeConfigQuietCreatesConfig(t *testing.T) {
	projectRoot := t.TempDir()

	if err := createOpenCodeConfigQuiet(projectRoot); err != nil {
		t.Fatalf("createOpenCodeConfigQuiet returned error: %v", err)
	}

	config := readJSONFile(t, filepath.Join(projectRoot, "opencode.json"))

	if got := config["$schema"]; got != "https://opencode.ai/config.json" {
		t.Fatalf("expected OpenCode schema, got %#v", got)
	}

	mcp := getMap(t, config, "mcp")
	knowns := getMap(t, mcp, "knowns")
	if got := knowns["type"]; got != "local" {
		t.Fatalf("expected knowns MCP type local, got %#v", got)
	}
	if got := knowns["enabled"]; got != true {
		t.Fatalf("expected knowns MCP enabled true, got %#v", got)
	}

	command, ok := knowns["command"].([]any)
	if !ok {
		t.Fatalf("expected knowns command to be []any, got %T", knowns["command"])
	}
	if len(command) != 5 {
		t.Fatalf("expected 5 command parts, got %d", len(command))
	}
	expected := []string{"npx", "-y", "knowns", "mcp", "--stdio"}
	for i, want := range expected {
		if command[i] != want {
			t.Fatalf("expected command[%d] = %q, got %#v", i, want, command[i])
		}
	}
}

func TestCreateMCPJsonFileQuietUsesNpxKnowns(t *testing.T) {
	projectRoot := t.TempDir()

	if err := createMCPJsonFileQuiet(projectRoot, false); err != nil {
		t.Fatalf("createMCPJsonFileQuiet returned error: %v", err)
	}

	config := readJSONFile(t, filepath.Join(projectRoot, ".mcp.json"))
	mcpServers := getMap(t, config, "mcpServers")
	knowns := getMap(t, mcpServers, "knowns")

	if got := knowns["command"]; got != "npx" {
		t.Fatalf("expected command npx, got %#v", got)
	}

	args, ok := knowns["args"].([]any)
	if !ok {
		t.Fatalf("expected args to be []any, got %T", knowns["args"])
	}
	expected := []string{"-y", "knowns", "mcp", "--stdio"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, want := range expected {
		if args[i] != want {
			t.Fatalf("expected args[%d] = %q, got %#v", i, want, args[i])
		}
	}
}

func TestCreateOpenCodeConfigQuietMergesExistingConfig(t *testing.T) {
	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, "opencode.json")

	existing := map[string]any{
		"model": "anthropic/claude-sonnet-4-5",
		"tools": map[string]any{
			"bash": "ask",
		},
		"mcp": map[string]any{
			"context7": map[string]any{
				"type": "remote",
				"url":  "https://mcp.context7.com/mcp",
			},
		},
	}

	writeJSONFile(t, configPath, existing)

	if err := createOpenCodeConfigQuiet(projectRoot); err != nil {
		t.Fatalf("createOpenCodeConfigQuiet returned error: %v", err)
	}

	config := readJSONFile(t, configPath)
	if got := config["model"]; got != existing["model"] {
		t.Fatalf("expected existing model to be preserved, got %#v", got)
	}

	tools := getMap(t, config, "tools")
	if got := tools["bash"]; got != "ask" {
		t.Fatalf("expected existing tools to be preserved, got %#v", got)
	}

	mcp := getMap(t, config, "mcp")
	if _, ok := mcp["context7"]; !ok {
		t.Fatalf("expected existing MCP entry to be preserved")
	}
	if _, ok := mcp["knowns"]; !ok {
		t.Fatalf("expected knowns MCP entry to be added")
	}
}

func TestCreateInstructionFilesQuietIncludesOpenCode(t *testing.T) {
	projectRoot := t.TempDir()

	if err := createInstructionFilesQuiet(projectRoot, false); err != nil {
		t.Fatalf("createInstructionFilesQuiet returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "OPENCODE.md")); err != nil {
		t.Fatalf("expected OPENCODE.md to be created: %v", err)
	}
}

func TestWriteKnownsGitignoreGitIgnoredTracksDocsAndTemplatesOnly(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")

	if err := os.WriteFile(gitignorePath, []byte("bin/\n"), 0644); err != nil {
		t.Fatalf("seed .gitignore: %v", err)
	}

	if err := writeKnownsGitignore(dir, "git-ignored"); err != nil {
		t.Fatalf("writeKnownsGitignore returned error: %v", err)
	}

	content := readTextFile(t, gitignorePath)
	assertContains(t, content, "bin/\n")
	assertContains(t, content, ".knowns/*")
	assertContains(t, content, "!.knowns/docs/")
	assertContains(t, content, "!.knowns/docs/**")
	assertContains(t, content, "!.knowns/templates/")
	assertContains(t, content, "!.knowns/templates/**")
	assertContains(t, content, knownsGitignoreBegin)
	assertContains(t, content, knownsGitignoreEnd)
	assertNotContains(t, content, ".knowns/\n")
}

func TestWriteKnownsGitignoreGitTrackedRemovesManagedBlock(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	seed := strings.Join([]string{
		"bin/",
		knownsGitignoreBegin,
		".knowns/*",
		"!.knowns/docs/**",
		knownsGitignoreEnd,
		"tmp/",
	}, "\n") + "\n"

	if err := os.WriteFile(gitignorePath, []byte(seed), 0644); err != nil {
		t.Fatalf("seed .gitignore: %v", err)
	}

	if err := writeKnownsGitignore(dir, "git-tracked"); err != nil {
		t.Fatalf("writeKnownsGitignore returned error: %v", err)
	}

	content := readTextFile(t, gitignorePath)
	if want := "bin/\ntmp/\n"; content != want {
		t.Fatalf("unexpected .gitignore content:\nwant:\n%s\n got:\n%s", want, content)
	}
}

func TestWriteKnownsGitignoreNoneLeavesGitignoreUnmanaged(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	seed := strings.Join([]string{
		"bin/",
		knownsGitignoreBegin,
		".knowns/*",
		"!.knowns/docs/**",
		knownsGitignoreEnd,
	}, "\n") + "\n"

	if err := os.WriteFile(gitignorePath, []byte(seed), 0644); err != nil {
		t.Fatalf("seed .gitignore: %v", err)
	}

	if err := writeKnownsGitignore(dir, "none"); err != nil {
		t.Fatalf("writeKnownsGitignore returned error: %v", err)
	}

	content := readTextFile(t, gitignorePath)
	if want := "bin/\n"; content != want {
		t.Fatalf("unexpected .gitignore content:\nwant:\n%s\n got:\n%s", want, content)
	}
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal file %s: %v", path, err)
	}

	return result
}

func writeJSONFile(t *testing.T, path string, value map[string]any) {
	t.Helper()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func getMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()

	result, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be map[string]any, got %T", key, value[key])
	}

	return result
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	return string(data)
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()

	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q, got:\n%s", want, content)
	}
}

func assertNotContains(t *testing.T, content, want string) {
	t.Helper()

	if strings.Contains(content, want) {
		t.Fatalf("expected content not to contain %q, got:\n%s", want, content)
	}
}
