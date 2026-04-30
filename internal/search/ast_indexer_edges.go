package search

import (
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type edgeTarget struct {
	RawTarget        string
	TargetName       string
	TargetQualifier  string
	TargetModuleHint string
	ReceiverTypeHint string
}

type importedBinding struct {
	ModulePath string
	ExportName string
}

type fileImportIndex struct {
	Named     map[string]importedBinding
	Default   map[string]importedBinding
	Namespace map[string]string
}

func newFileImportIndex() fileImportIndex {
	return fileImportIndex{
		Named:     make(map[string]importedBinding),
		Default:   make(map[string]importedBinding),
		Namespace: make(map[string]string),
	}
}

func (idx fileImportIndex) lookup(local string) (importedBinding, bool) {
	if b, ok := idx.Named[local]; ok {
		return b, true
	}
	if b, ok := idx.Default[local]; ok {
		return b, true
	}
	return importedBinding{}, false
}

func normalizeJSImportPath(raw string) string {
	return strings.Trim(raw, `"'`+"`")
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

func extractFileImports(docPath string, data []byte) fileImportIndex {
	idx := newFileImportIndex()
	text := string(data)
	ext := strings.ToLower(filepath.Ext(docPath))

	switch ext {
	case ".ts", ".tsx", ".js", ".jsx":
		combinedRE := regexp.MustCompile(`(?m)^\s*import\s+([A-Za-z_$][\w$]*)\s*,\s*\{([^}]*)\}\s*from\s*["']([^"']+)["']`)
		for _, m := range combinedRE.FindAllStringSubmatch(text, -1) {
			idx.Default[m[1]] = importedBinding{ModulePath: normalizeJSImportPath(m[3]), ExportName: "default"}
			for _, part := range strings.Split(m[2], ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				exportName := part
				localName := part
				if left, right, ok := strings.Cut(part, " as "); ok {
					exportName = strings.TrimSpace(left)
					localName = strings.TrimSpace(right)
				}
				idx.Named[localName] = importedBinding{ModulePath: normalizeJSImportPath(m[3]), ExportName: exportName}
			}
		}

		namespaceRE := regexp.MustCompile(`(?m)^\s*import\s+\*\s+as\s+([A-Za-z_$][\w$]*)\s+from\s*["']([^"']+)["']`)
		for _, m := range namespaceRE.FindAllStringSubmatch(text, -1) {
			idx.Namespace[m[1]] = normalizeJSImportPath(m[2])
		}

		namedRE := regexp.MustCompile(`(?m)^\s*import\s*\{([^}]*)\}\s*from\s*["']([^"']+)["']`)
		for _, m := range namedRE.FindAllStringSubmatch(text, -1) {
			for _, part := range strings.Split(m[1], ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				exportName := part
				localName := part
				if left, right, ok := strings.Cut(part, " as "); ok {
					exportName = strings.TrimSpace(left)
					localName = strings.TrimSpace(right)
				}
				idx.Named[localName] = importedBinding{ModulePath: normalizeJSImportPath(m[2]), ExportName: exportName}
			}
		}

		defaultRE := regexp.MustCompile(`(?m)^\s*import\s+([A-Za-z_$][\w$]*)\s+from\s*["']([^"']+)["']`)
		for _, m := range defaultRE.FindAllStringSubmatch(text, -1) {
			idx.Default[m[1]] = importedBinding{ModulePath: normalizeJSImportPath(m[2]), ExportName: "default"}
		}
	case ".go":
		goImportRE := regexp.MustCompile(`(?m)^\s*(?:([A-Za-z_][\w]*)\s+)?"([^"]+)"`)
		for _, m := range goImportRE.FindAllStringSubmatch(text, -1) {
			modulePath := strings.TrimSpace(m[2])
			alias := strings.TrimSpace(m[1])
			if modulePath == "" {
				continue
			}
			if alias == "" {
				alias = pathBase(modulePath)
			}
			if alias == "." || alias == "_" || alias == "" {
				continue
			}
			idx.Namespace[alias] = modulePath
		}
	case ".java":
		// Java: import com.example.MyClass; or import static com.example.MyClass.method;
		javaImportRE := regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([\w.]+)\s*;`)
		for _, m := range javaImportRE.FindAllStringSubmatch(text, -1) {
			modulePath := strings.TrimSpace(m[1])
			if modulePath == "" {
				continue
			}
			alias := pathBase(strings.ReplaceAll(modulePath, ".", "/"))
			idx.Namespace[alias] = modulePath
		}
	case ".cs":
		// C#: using System.Collections.Generic; or using Alias = Namespace;
		csUsingRE := regexp.MustCompile(`(?m)^\s*using\s+(?:static\s+)?(?:(\w+)\s*=\s*)?([\w.]+)\s*;`)
		for _, m := range csUsingRE.FindAllStringSubmatch(text, -1) {
			alias := strings.TrimSpace(m[1])
			modulePath := strings.TrimSpace(m[2])
			if modulePath == "" {
				continue
			}
			if alias == "" {
				alias = pathBase(strings.ReplaceAll(modulePath, ".", "/"))
			}
			idx.Namespace[alias] = modulePath
		}
	case ".rs":
		// Rust: use std::collections::HashMap; or use crate::module::Type;
		rsUseRE := regexp.MustCompile(`(?m)^\s*use\s+([\w:]+(?:::\w+)*)\s*;`)
		for _, m := range rsUseRE.FindAllStringSubmatch(text, -1) {
			modulePath := strings.TrimSpace(m[1])
			if modulePath == "" {
				continue
			}
			alias := pathBase(strings.ReplaceAll(modulePath, "::", "/"))
			idx.Namespace[alias] = modulePath
		}
	}

	return idx
}

func pathBase(modulePath string) string {
	modulePath = strings.TrimSuffix(modulePath, "/")
	if idx := strings.LastIndex(modulePath, "/"); idx >= 0 {
		return modulePath[idx+1:]
	}
	return modulePath
}

func (v *symbolVisitor) qualifyEdgeTarget(target edgeTarget) edgeTarget {
	if target.TargetName == "" {
		return target
	}
	if target.TargetQualifier == "" {
		if binding, ok := v.imports.lookup(target.TargetName); ok {
			target.TargetModuleHint = binding.ModulePath
			if binding.ExportName != "" && binding.ExportName != "default" {
				target.TargetName = binding.ExportName
			}
		}
		return target
	}
	if modulePath, ok := v.imports.Namespace[target.TargetQualifier]; ok {
		target.TargetModuleHint = modulePath
	}
	return target
}

func (v *symbolVisitor) addEdge(fromFunc string, target edgeTarget, edgeType string) {
	if target.TargetName == "" {
		return
	}
	target = v.qualifyEdgeTarget(target)
	fromID := CodeChunkID(v.docPath, fromFunc)
	if strings.TrimSpace(fromFunc) == "" {
		fromID = CodeChunkID(v.docPath, "")
	}
	resolvedTo := target.TargetName
	if target.TargetModuleHint != "" {
		resolvedTo = target.TargetModuleHint + "::" + target.TargetName
	}
	v.edges = append(v.edges, CodeEdge{
		From:                 fromID,
		To:                   target.TargetName,
		Type:                 edgeType,
		FromPath:             v.docPath,
		ToPath:               "",
		RawTarget:            target.RawTarget,
		TargetName:           target.TargetName,
		TargetQualifier:      target.TargetQualifier,
		TargetModuleHint:     target.TargetModuleHint,
		ReceiverTypeHint:     target.ReceiverTypeHint,
		ResolutionStatus:     "unresolved",
		ResolutionConfidence: "low",
		ResolvedTo:           resolvedTo,
	})
}

// callExpressionTarget extracts structured callee information from a call_expression.
// Handles: foo(), obj.method(), pkg.Function()
func (v *symbolVisitor) callExpressionTarget(node sitter.Node) edgeTarget {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier":
			name := v.extractNodeText(child)
			return edgeTarget{RawTarget: name, TargetName: name}
		case "field_expression", "selector_expression", "member_expression":
			qualifier, name := v.fieldExpressionParts(child)
			if name != "" {
				raw := name
				if qualifier != "" {
					raw = qualifier + "." + name
				}
				return edgeTarget{RawTarget: raw, TargetName: name, TargetQualifier: qualifier}
			}
		}
	}
	return edgeTarget{}
}

func (v *symbolVisitor) instantiatedTarget(node sitter.Node) edgeTarget {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		if qualifier, name := v.typeExpressionParts(child); name != "" {
			raw := name
			if qualifier != "" {
				raw = qualifier + "." + name
			}
			return edgeTarget{RawTarget: raw, TargetName: name, TargetQualifier: qualifier}
		}
	}
	return edgeTarget{}
}

func (v *symbolVisitor) instantiatedType(node sitter.Node) string {
	return v.instantiatedTarget(node).TargetName
}

func (v *symbolVisitor) fieldExpressionParts(node *sitter.Node) (string, string) {
	if node == nil {
		return "", ""
	}
	var qualifier string
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier", "type_identifier", "field_identifier":
			text := v.extractNodeText(child)
			if qualifier == "" {
				qualifier = text
			} else {
				name = text
			}
		case "field_expression", "selector_expression", "member_expression":
			q, n := v.fieldExpressionParts(child)
			if n != "" {
				if qualifier == "" {
					qualifier = n
				} else if name == "" {
					name = n
				}
			}
			if qualifier == "" && q != "" {
				qualifier = q
			}
		}
	}
	if name == "" {
		name = qualifier
		qualifier = ""
	}
	return qualifier, name
}

func (v *symbolVisitor) typeExpressionParts(node *sitter.Node) (string, string) {
	if node == nil {
		return "", ""
	}
	kind := node.Kind()
	switch kind {
	case "type_identifier", "identifier", "field_identifier":
		name := v.extractNodeText(node)
		return "", name
	case "qualified_type", "selector_expression", "member_expression", "field_expression", "generic_type", "type_instantiation", "new_expression", "composite_literal":
		for i := int(node.ChildCount()) - 1; i >= 0; i-- {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			if q, name := v.typeExpressionParts(child); name != "" {
				return q, name
			}
		}
	}
	return "", ""
}

func (v *symbolVisitor) typeExpressionName(node *sitter.Node) string {
	_, name := v.typeExpressionParts(node)
	return name
}

func (v *symbolVisitor) fieldExpressionName(node *sitter.Node) string {
	_, name := v.fieldExpressionParts(node)
	return name
}

func (v *symbolVisitor) extractImport(node sitter.Node) string {
	kind := node.Kind()
	switch kind {
	case "import_specifier", "import_declaration", "import_clause", "import_statement":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			if child.Kind() == "string_literal" {
				return strings.Trim(v.extractNodeText(child), `"`)
			}
			if nested := v.extractImport(*child); nested != "" {
				return nested
			}
		}
	}
	return ""
}

func (v *symbolVisitor) importTarget(node sitter.Node) edgeTarget {
	modulePath := v.extractImport(node)
	if modulePath == "" {
		return edgeTarget{}
	}
	name := pathBase(modulePath)
	return edgeTarget{RawTarget: modulePath, TargetName: name, TargetModuleHint: modulePath}
}

func (v *symbolVisitor) addImportEdge(fromFunc string, node sitter.Node) {
	target := v.importTarget(node)
	if target.TargetName == "" {
		return
	}
	v.edges = append(v.edges, CodeEdge{
		From:                 CodeChunkID(v.docPath, fromFunc),
		To:                   target.TargetName,
		Type:                 "imports",
		FromPath:             v.docPath,
		ToPath:               "",
		RawTarget:            target.RawTarget,
		TargetName:           target.TargetName,
		TargetQualifier:      target.TargetQualifier,
		TargetModuleHint:     target.TargetModuleHint,
		ReceiverTypeHint:     "",
		ResolutionStatus:     "unresolved",
		ResolutionConfidence: "medium",
		ResolvedTo:           target.TargetModuleHint,
	})
}

func (v *symbolVisitor) importAnchor() string {
	if len(v.funcStack) > 0 {
		return v.funcStack[len(v.funcStack)-1]
	}
	if len(v.classStack) > 0 {
		return v.classStack[len(v.classStack)-1]
	}
	return ""
}
