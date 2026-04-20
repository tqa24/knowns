//go:build !windows

package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexFile_ExtractsNodeFirstGoSymbols(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "sample.go")
	source := `package sample

type User struct{}

type Store interface {
	Save()
}

func Hello() {
	SaveHelper()
}

func SaveHelper() {}

func (u User) Save() {
	SaveHelper()
}
`
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	symbols, edges, err := IndexFile("sample.go", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	if len(symbols) != 6 {
		t.Fatalf("symbol count = %d, want 6", len(symbols))
	}

	kindByID := map[string]string{}
	for _, sym := range symbols {
		kindByID[CodeChunkID(sym.DocPath, sym.Name)] = sym.Kind
	}

	assertKind := func(id, want string) {
		t.Helper()
		if got := kindByID[id]; got != want {
			t.Fatalf("kind for %s = %q, want %q", id, got, want)
		}
	}

	assertKind(CodeChunkID("sample.go", ""), "file")
	assertKind(CodeChunkID("sample.go", "Hello"), "function")
	assertKind(CodeChunkID("sample.go", "SaveHelper"), "function")
	assertKind(CodeChunkID("sample.go", "Save"), "method")
	assertKind(CodeChunkID("sample.go", "User"), "class")
	assertKind(CodeChunkID("sample.go", "Store"), "interface")

	if CodeChunkID("sample.go", "") != "code::sample.go::__file__" {
		t.Fatalf("unexpected file chunk id: %s", CodeChunkID("sample.go", ""))
	}

	containsCount := 0
	callCount := 0
	implementsCount := 0
	for _, edge := range edges {
		switch edge.Type {
		case "contains":
			containsCount++
			if edge.From != CodeChunkID("sample.go", "") {
				t.Fatalf("contains edge from = %q, want file chunk", edge.From)
			}
		case "calls":
			callCount++
			if edge.To != CodeChunkID("sample.go", "SaveHelper") {
				t.Fatalf("call edge target = %q, want SaveHelper", edge.To)
			}
		case "implements":
			implementsCount++
			if edge.From != CodeChunkID("sample.go", "User") || edge.To != CodeChunkID("sample.go", "Store") {
				t.Fatalf("implements edge = %q -> %q, want User -> Store", edge.From, edge.To)
			}
		default:
			t.Fatalf("unexpected edge type %q", edge.Type)
		}
	}

	if containsCount != 5 {
		t.Fatalf("contains count = %d, want 5", containsCount)
	}
	if callCount != 2 {
		t.Fatalf("call count = %d, want 2", callCount)
	}
	if implementsCount != 1 {
		t.Fatalf("implements count = %d, want 1", implementsCount)
	}
}

func TestIndexFile_OmitsUnresolvedCalls(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "unresolved.go")
	source := `package sample

func Hello() {
	MissingHelper()
}
`
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, edges, err := IndexFile("unresolved.go", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	for _, edge := range edges {
		if edge.Type == "calls" {
			t.Fatalf("unexpected unresolved call edge: %+v", edge)
		}
	}
}

func TestIndexAllFiles_ImplementsAcrossSamePackageFiles(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "store.go"), []byte(`package sample

type Store interface {
	Save()
}
`), 0o644); err != nil {
		t.Fatalf("write store source: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "user.go"), []byte(`package sample

type User struct{}

func (u User) Save() {}
`), 0o644); err != nil {
		t.Fatalf("write user source: %v", err)
	}

	_, edges, err := IndexAllFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("IndexAllFiles: %v", err)
	}

	for _, edge := range edges {
		if edge.Type == "implements" && edge.From == CodeChunkID("user.go", "User") && edge.To == CodeChunkID("store.go", "Store") {
			return
		}
	}

	t.Fatalf("expected implements edge across same-package files")
}

func TestIndexAllFiles_ResolvesCallsAcrossSamePackageFiles(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "sample")
	otherDir := filepath.Join(tmpDir, "other")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir sample: %v", err)
	}
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("mkdir other: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pkgDir, "hello.go"), []byte(`package sample

func Hello() {
	SaveHelper()
}
`), 0o644); err != nil {
		t.Fatalf("write hello source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "helper.go"), []byte(`package sample

func SaveHelper() {}
`), 0o644); err != nil {
		t.Fatalf("write helper source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, "helper.go"), []byte(`package other

func SaveHelper() {}
`), 0o644); err != nil {
		t.Fatalf("write other helper source: %v", err)
	}

	_, edges, err := IndexAllFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("IndexAllFiles: %v", err)
	}

	for _, edge := range edges {
		if edge.Type == "calls" && edge.From == CodeChunkID("sample/hello.go", "Hello") && edge.To == CodeChunkID("sample/helper.go", "SaveHelper") {
			return
		}
	}

	t.Fatalf("expected same-package call edge across files")
}

func TestListCodeCandidateFiles_RespectsIgnoresAndTests(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir pkg: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "ignoredir"), 0o755); err != nil {
		t.Fatalf("mkdir ignoredir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "ui", "dist"), 0o755); err != nil {
		t.Fatalf("mkdir ui/dist: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".knowns"), 0o755); err != nil {
		t.Fatalf("mkdir .knowns: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("ignoredir/\nui/dist/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".knowns", "config.json"), []byte(`{"settings":{"codeIntelligenceIgnore":["skip.go"]}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	files := map[string]string{
		"pkg/main.go":         "package pkg\nfunc Main() {}\n",
		"pkg/main_test.go":    "package pkg\nfunc TestMain() {}\n",
		"ignoredir/hidden.go": "package ignoredir\nfunc Hidden() {}\n",
		"ui/dist/bundle.js":   "function bundle() {}\n",
		"skip.go":             "package main\nfunc Skip() {}\n",
		"pkg/component.ts":    "export function component() {}\n",
	}
	for rel, src := range files {
		abs := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	got, err := ListCodeCandidateFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("ListCodeCandidateFiles: %v", err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"pkg/main.go", "pkg/component.ts"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing candidate %s in %v", want, got)
		}
	}
	for _, unwanted := range []string{"pkg/main_test.go", "ignoredir/hidden.go", "ui/dist/bundle.js", "skip.go"} {
		if strings.Contains(joined, unwanted) {
			t.Fatalf("unexpected candidate %s in %v", unwanted, got)
		}
	}

	gotWithTests, err := ListCodeCandidateFiles(tmpDir, true)
	if err != nil {
		t.Fatalf("ListCodeCandidateFiles(includeTests): %v", err)
	}
	if !strings.Contains(strings.Join(gotWithTests, "\n"), "pkg/main_test.go") {
		t.Fatalf("expected test file when includeTests=true, got %v", gotWithTests)
	}
}

func TestIndexAllFilesWithProgress_CallbackPerCandidate(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte("package sample\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte("package sample\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatalf("write b.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "c_test.go"), []byte("package sample\nfunc TestC() {}\n"), 0o644); err != nil {
		t.Fatalf("write c_test.go: %v", err)
	}

	var seen []string
	_, _, err := IndexAllFilesWithProgress(tmpDir, false, func(rel string) {
		seen = append(seen, rel)
	})
	if err != nil {
		t.Fatalf("IndexAllFilesWithProgress: %v", err)
	}
	if strings.Join(seen, ",") != "a.go,b.go" {
		t.Fatalf("progress callback files = %v, want [a.go b.go]", seen)
	}
}

func TestIndexFile_ExtractsFunctionValuedVariableDeclarations(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "sample.ts")
	source := `const saveHelper = () => {}
export const formatValue = function () {}

function hello() {
	saveHelper()
	formatValue()
}

func TestIndexFile_AddsClassMethodOwnershipAndExtendsEdges(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "sample.ts")
	source := "class BaseController {}\n\nclass UserController extends BaseController {\n\tindex() {\n\t\treturn 1\n\t}\n}\n"
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, edges, err := IndexFile("sample.ts", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	hasMethod := false
	hasExtends := false
	for _, edge := range edges {
		if edge.Type == "has_method" && edge.From == CodeChunkID("sample.ts", "UserController") && edge.To == CodeChunkID("sample.ts", "index") {
			hasMethod = true
		}
		if edge.Type == "extends" && edge.From == CodeChunkID("sample.ts", "UserController") && edge.To == CodeChunkID("sample.ts", "BaseController") {
			hasExtends = true
		}
	}
	if !hasMethod {
		t.Fatalf("expected class -> method ownership edge")
	}
	if !hasExtends {
		t.Fatalf("expected extends edge")
	}
}

func TestIndexFile_ExtractsDecoratedTypescriptMethods(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "controller.ts")
	source := "@Controller()\nexport class AppController {\n  @Get('health')\n  getHealth(): string {\n    return 'ok'\n  }\n\n  @Get()\n  getHello(): string {\n    return 'hello'\n  }\n}\n"
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	symbols, edges, err := IndexFile("controller.ts", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	kindByID := map[string]string{}
	for _, sym := range symbols {
		kindByID[CodeChunkID(sym.DocPath, sym.Name)] = sym.Kind
	}
	if got := kindByID[CodeChunkID("controller.ts", "getHealth")]; got != "method" {
		t.Fatalf("kind for getHealth = %q, want method", got)
	}
	if got := kindByID[CodeChunkID("controller.ts", "getHello")]; got != "method" {
		t.Fatalf("kind for getHello = %q, want method", got)
	}

	ownerEdges := map[string]bool{
		CodeChunkID("controller.ts", "getHealth"): false,
		CodeChunkID("controller.ts", "getHello"):  false,
	}
	for _, edge := range edges {
		if edge.Type == "has_method" && edge.From == CodeChunkID("controller.ts", "AppController") {
			if _, ok := ownerEdges[edge.To]; ok {
				ownerEdges[edge.To] = true
			}
		}
	}
	for methodID, found := range ownerEdges {
		if !found {
			t.Fatalf("expected ownership edge for %s", methodID)
		}
	}
}
`
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	symbols, edges, err := IndexFile("sample.ts", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	kindByID := map[string]string{}
	for _, sym := range symbols {
		kindByID[CodeChunkID(sym.DocPath, sym.Name)] = sym.Kind
	}

	if got := kindByID[CodeChunkID("sample.ts", "saveHelper")]; got != "function" {
		t.Fatalf("kind for saveHelper = %q, want function", got)
	}
	if got := kindByID[CodeChunkID("sample.ts", "formatValue")]; got != "function" {
		t.Fatalf("kind for formatValue = %q, want function", got)
	}

	wantCalls := map[string]bool{
		CodeChunkID("sample.ts", "saveHelper"):  false,
		CodeChunkID("sample.ts", "formatValue"): false,
	}
	for _, edge := range edges {
		if edge.Type == "calls" && edge.From == CodeChunkID("sample.ts", "hello") {
			if _, ok := wantCalls[edge.To]; ok {
				wantCalls[edge.To] = true
			}
		}
	}
	for to, found := range wantCalls {
		if !found {
			t.Fatalf("expected hello call edge to %s", to)
		}
	}
}

func TestIndexFile_DoesNotExtractPlainValueVariablesAsFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "values.ts")
	source := `const answer = 42
const label = "x"
`
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	symbols, _, err := IndexFile("values.ts", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	for _, sym := range symbols {
		if sym.Name == "answer" || sym.Name == "label" {
			t.Fatalf("unexpected plain value variable symbol: %+v", sym)
		}
	}
}

func TestIndexFile_EnrichesContentWithResolvedEdges(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "sample.go")
	source := `package sample

type User struct{}

func SaveHelper() {}

func Hello() {
	SaveHelper()
	_ = User{}
}
`
	if err := os.WriteFile(absPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	symbols, edges, err := IndexFile("sample.go", absPath)
	if err != nil {
		t.Fatalf("IndexFile: %v", err)
	}

	var hello CodeSymbol
	for _, sym := range symbols {
		if sym.Name == "Hello" {
			hello = sym
			break
		}
	}
	if hello.Name == "" {
		t.Fatalf("expected Hello symbol")
	}
	if !strings.Contains(hello.Content, "calls: SaveHelper") {
		t.Fatalf("hello content missing call summary: %q", hello.Content)
	}
	if !strings.Contains(hello.Content, "instantiates: User") {
		t.Fatalf("hello content missing instantiates summary: %q", hello.Content)
	}

	foundInstantiate := false
	for _, edge := range edges {
		if edge.Type == "instantiates" && edge.From == CodeChunkID("sample.go", "Hello") && edge.To == CodeChunkID("sample.go", "User") {
			foundInstantiate = true
		}
	}
	if !foundInstantiate {
		t.Fatalf("expected resolved instantiates edge for Hello -> User")
	}
}

func TestIndexAllFiles_ResolvesRelativeImportToFileChunk(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "helper.ts"), []byte(`export function saveHelper() {}`), 0o644); err != nil {
		t.Fatalf("write helper source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main.ts"), []byte(`import { saveHelper } from "./helper"

export function hello() {
	saveHelper()
}
`), 0o644); err != nil {
		t.Fatalf("write main source: %v", err)
	}

	_, edges, err := IndexAllFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("IndexAllFiles: %v", err)
	}

	for _, edge := range edges {
		if edge.Type == "imports" && edge.From == CodeChunkID("main.ts", "") && edge.To == CodeChunkID("helper.ts", "") {
			return
		}
	}

	t.Fatalf("expected resolved import edge to helper file chunk")
}

func TestIndexAllFiles_UsesCodeIntelligenceIgnoreConfig(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".knowns"), 0o755); err != nil {
		t.Fatalf("mkdir .knowns: %v", err)
	}
	config := `{
	  "name": "sample",
	  "id": "sample",
	  "createdAt": "2026-04-08T00:00:00Z",
	  "settings": {
	    "defaultPriority": "medium",
	    "statuses": ["todo"],
	    "codeIntelligenceIgnore": ["ignored/**"]
	  }
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".knowns", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "ignored"), 0o755); err != nil {
		t.Fatalf("mkdir ignored: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ignored", "skip.go"), []byte(`package ignored

func Skip() {}
`), 0o644); err != nil {
		t.Fatalf("write ignored source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.go"), []byte(`package sample

func Keep() {}
`), 0o644); err != nil {
		t.Fatalf("write keep source: %v", err)
	}

	symbols, _, err := IndexAllFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("IndexAllFiles: %v", err)
	}

	for _, sym := range symbols {
		if sym.DocPath == "ignored/skip.go" {
			t.Fatalf("expected ignored file to be skipped: %+v", sym)
		}
	}
}
