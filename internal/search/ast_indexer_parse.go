//go:build !windows

package search

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// codeMatcher matches paths against gitignore patterns.
// Handles: exact paths, directory/ (trailing slash), ** glob, negation !.
type codeMatcher struct {
	ignore []string // patterns to ignore
	allow  []string // negated patterns (re-enabled)
}

func newCodeMatcher(patterns []string) *codeMatcher {
	m := &codeMatcher{}
	for _, p := range patterns {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		negated := strings.HasPrefix(trimmed, "!")
		if negated {
			m.allow = append(m.allow, strings.TrimPrefix(trimmed, "!"))
		} else {
			m.ignore = append(m.ignore, trimmed)
		}
	}
	return m
}

// ShouldIgnore returns true if path should be ignored.
// Only checks parent directories (not the file itself for now).
func (m *codeMatcher) ShouldIgnore(path string) bool {
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ {
		dir := strings.Join(parts[:i], "/")
		if m.matchOne(dir, true) {
			return true
		}
	}
	return m.matchOne(path, false)
}

func (m *codeMatcher) matchOne(path string, isDir bool) bool {
	for _, a := range m.allow {
		if a == path || (isDir && (a == path+"/" || a == path)) {
			return false
		}
		if isDir && strings.TrimSuffix(a, "/") == path {
			return false
		}
	}
	for _, ign := range m.ignore {
		if m.matches(ign, path, isDir) {
			return true
		}
	}
	return false
}

func (m *codeMatcher) matches(pattern, path string, isDir bool) bool {
	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		return strings.HasPrefix(path, dir+"/") || path == dir
	}
	if strings.Contains(pattern, "*") {
		regex := strings.ReplaceAll(pattern, ".", "\\.")
		regex = strings.ReplaceAll(regex, "*", ".*")
		matched, _ := regexp.MatchString("^"+regex+"$", path)
		return matched
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func parseRawFile(docPath, absPath string) ([]CodeSymbol, []CodeEdge, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, err
	}
	lang := detectLang(docPath)
	if lang == nil {
		return nil, nil, nil
	}

	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, nil, err
	}

	tree := parser.Parse(data, nil)
	if tree == nil {
		return nil, nil, nil
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		return nil, nil, nil
	}

	syms, eds := extractSymbols(docPath, data, root)
	return syms, eds, nil
}

func finalizeCodeParse(symbols []CodeSymbol, edges []CodeEdge) ([]CodeSymbol, []CodeEdge, error) {
	symbols = filterSupportedCodeSymbols(symbols)
	edges = append(edges, buildImplementsEdges(symbols)...)
	edges = ResolveCodeEdges(symbols, edges)
	validIDs := symbolIDSet(symbols)
	edges = filterGraphCodeEdges(edges)
	edges = filterEdgesForSymbols(edges, validIDs)
	symbols = enrichCodeSymbolContent(symbols, edges)
	return symbols, edges, nil
}

func detectLang(path string) *sitter.Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return sitter.NewLanguage(tree_sitter_go.Language())
	case ".ts":
		return sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	case ".tsx":
		return sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
	case ".js":
		return sitter.NewLanguage(tree_sitter_javascript.Language())
	case ".jsx":
		return sitter.NewLanguage(tree_sitter_javascript.Language())
	case ".py":
		return sitter.NewLanguage(tree_sitter_python.Language())
	default:
		return nil
	}
}

func isHiddenFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".") || base == "node_modules" || base == "__pycache__"
}

func isHiddenDir(rel string) bool {
	parts := strings.Split(rel, string(filepath.Separator))
	for _, p := range parts {
		if strings.HasPrefix(p, ".") && p != "." {
			return true
		}
	}
	return false
}

func loadIgnoreList(projectRoot string) (map[string]bool, error) {
	ignore := make(map[string]bool)

	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	if data, err := os.ReadFile(gitignorePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			negated := strings.HasPrefix(line, "!")
			if negated {
				line = line[1:]
			}
			if !negated {
				ignore[line] = true
				if strings.HasSuffix(line, "/") {
					dir := strings.TrimSuffix(line, "/")
					if dir != "" {
						ignore[dir] = true
					}
				}
			} else {
				delete(ignore, line)
				if strings.HasSuffix(line, "/") {
					dir := strings.TrimSuffix(line, "/")
					if dir != "" {
						delete(ignore, dir)
					}
				}
			}
		}
	}

	return ignore, nil
}

func filterGraphCodeEdges(edges []CodeEdge) []CodeEdge {
	filtered := make([]CodeEdge, 0, len(edges))
	for _, edge := range edges {
		switch edge.Type {
		case "contains":
			filtered = append(filtered, edge)
		case "has_method", "implements":
			filtered = append(filtered, edge)
		case "calls":
			if edge.ResolutionStatus == "resolved_internal" || strings.HasPrefix(edge.To, "code::") {
				filtered = append(filtered, edge)
			}
		case "extends":
			if edge.ResolutionStatus == "resolved_internal" || strings.HasPrefix(edge.To, "code::") {
				filtered = append(filtered, edge)
			}
		case "imports", "instantiates":
			if edge.ResolutionStatus == "resolved_internal" || strings.HasPrefix(edge.To, "code::") {
				filtered = append(filtered, edge)
			}
		}
	}
	return filtered
}

func symbolIDSet(symbols []CodeSymbol) map[string]bool {
	ids := make(map[string]bool, len(symbols))
	for _, sym := range symbols {
		ids[CodeChunkID(sym.DocPath, sym.Name)] = true
	}
	return ids
}

func filterEdgesForSymbols(edges []CodeEdge, validIDs map[string]bool) []CodeEdge {
	filtered := make([]CodeEdge, 0, len(edges))
	for _, edge := range edges {
		if validIDs[edge.From] && validIDs[edge.To] {
			filtered = append(filtered, edge)
		}
	}
	return filtered
}
