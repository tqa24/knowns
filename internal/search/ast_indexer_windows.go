//go:build windows && tree_sitter_stub

package search

import (
	"path/filepath"
	"strings"
)

type codeMatcher struct {
	ignore []string
	allow  []string
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
		matched := strings.TrimSpace(path) != ""
		if matched {
			matched, _ = filepath.Match(strings.ReplaceAll(pattern, "**", "*"), path)
		}
		if matched {
			return true
		}
		return false
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func resolveJSImportCandidate(fromPath, importPath string) []string {
	if importPath == "" || !strings.HasPrefix(importPath, ".") {
		return nil
	}
	baseDir := filepath.ToSlash(filepath.Dir(fromPath))
	base := filepath.ToSlash(filepath.Clean(filepath.Join(baseDir, importPath)))
	return []string{
		base,
		base + ".ts",
		base + ".tsx",
		base + ".js",
		base + ".jsx",
		filepath.ToSlash(filepath.Join(base, "index.ts")),
		filepath.ToSlash(filepath.Join(base, "index.tsx")),
		filepath.ToSlash(filepath.Join(base, "index.js")),
		filepath.ToSlash(filepath.Join(base, "index.jsx")),
	}
}

// Windows builds currently skip tree-sitter-backed AST parsing because the
// upstream language bindings used by this repo do not provide Windows Go
// sources, which breaks compilation in CI.
func parseRawFile(docPath, absPath string) ([]CodeSymbol, []CodeEdge, error) {
	return nil, nil, nil
}

func isSupportedCodeSymbolKind(kind string) bool {
	switch kind {
	case "file", "function", "method", "class", "interface":
		return true
	default:
		return false
	}
}

func filterSupportedCodeSymbols(symbols []CodeSymbol) []CodeSymbol {
	filtered := make([]CodeSymbol, 0, len(symbols))
	for _, sym := range symbols {
		if isSupportedCodeSymbolKind(sym.Kind) {
			filtered = append(filtered, sym)
		}
	}
	return filtered
}

func symbolIDSet(symbols []CodeSymbol) map[string]bool {
	valid := make(map[string]bool, len(symbols))
	for _, sym := range symbols {
		valid[CodeChunkID(sym.DocPath, sym.Name)] = true
	}
	return valid
}

func filterEdgesForSymbols(edges []CodeEdge, validIDs map[string]bool) []CodeEdge {
	filtered := make([]CodeEdge, 0, len(edges))
	for _, edge := range edges {
		if !validIDs[edge.From] {
			continue
		}
		if strings.HasPrefix(edge.To, "code::") && !validIDs[edge.To] {
			continue
		}
		filtered = append(filtered, edge)
	}
	return filtered
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
