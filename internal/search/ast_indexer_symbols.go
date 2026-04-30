package search

import (
	"bytes"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// extractSymbols walks the tree and extracts named symbols and edges.
func extractSymbols(docPath string, data []byte, root *sitter.Node) ([]CodeSymbol, []CodeEdge) {
	visitor := newSymbolVisitor(docPath, data)
	visitor.addFileSymbol()
	walkTree(root, visitor)
	visitor.addFileImportEdges()
	return visitor.Symbols(), visitor.Edges()
}

// symbolVisitor walks a tree-sitter tree collecting function/class/interface symbols.
type symbolVisitor struct {
	docPath    string
	data       []byte
	symbols    []CodeSymbol
	edges      []CodeEdge
	imports    fileImportIndex
	language   string
	funcStack  []string
	classStack []string
}

func newSymbolVisitor(docPath string, data []byte) *symbolVisitor {
	return &symbolVisitor{
		docPath:  docPath,
		data:     data,
		imports:  extractFileImports(docPath, data),
		language: strings.TrimPrefix(strings.ToLower(filepath.Ext(docPath)), "."),
	}
}

func (v *symbolVisitor) currentReceiverType() string {
	if len(v.classStack) == 0 {
		return ""
	}
	return v.classStack[len(v.classStack)-1]
}

func (v *symbolVisitor) currentPackageOrModule() string {
	switch v.language {
	case "go":
		base := strings.TrimSuffix(filepath.ToSlash(v.docPath), filepath.Ext(v.docPath))
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			return base[:idx]
		}
		return base
	default:
		return strings.TrimSuffix(filepath.ToSlash(v.docPath), filepath.Ext(v.docPath))
	}
}

func (v *symbolVisitor) Symbols() []CodeSymbol { return v.symbols }
func (v *symbolVisitor) Edges() []CodeEdge     { return v.edges }

func (v *symbolVisitor) addFileSymbol() {
	v.symbols = append(v.symbols, CodeSymbol{
		Name:    "",
		Kind:    "file",
		DocPath: v.docPath,
		Source:  strings.TrimSpace(string(v.data)),
	})
}

func (v *symbolVisitor) addFileImportEdges() {
	seen := make(map[string]bool)
	add := func(modulePath string) {
		modulePath = strings.TrimSpace(modulePath)
		if modulePath == "" || seen[modulePath] {
			return
		}
		seen[modulePath] = true
		target := edgeTarget{
			RawTarget:        modulePath,
			TargetName:       pathBase(modulePath),
			TargetModuleHint: modulePath,
		}
		v.addEdge("", target, "imports")
	}
	for _, binding := range v.imports.Named {
		add(binding.ModulePath)
	}
	for _, binding := range v.imports.Default {
		add(binding.ModulePath)
	}
	for _, modulePath := range v.imports.Namespace {
		add(modulePath)
	}
}

func (v *symbolVisitor) addContainsEdge(symbolName string) {
	if symbolName == "" {
		return
	}
	v.edges = append(v.edges, CodeEdge{
		From:     CodeChunkID(v.docPath, ""),
		To:       CodeChunkID(v.docPath, symbolName),
		Type:     "contains",
		FromPath: v.docPath,
		ToPath:   v.docPath,
	})
}

func (v *symbolVisitor) addOwnerEdge(symbolName string, kind string) {
	if symbolName == "" || kind != "method" || len(v.classStack) == 0 {
		return
	}
	ownerName := v.currentReceiverType()
	if ownerName == "" {
		return
	}
	v.edges = append(v.edges, CodeEdge{
		From:                 CodeChunkID(v.docPath, ownerName),
		To:                   CodeChunkID(v.docPath, symbolName),
		Type:                 "has_method",
		FromPath:             v.docPath,
		ToPath:               v.docPath,
		ResolutionStatus:     "resolved_internal",
		ResolutionConfidence: "high",
		ResolvedTo:           CodeChunkID(v.docPath, symbolName),
	})
}

func (v *symbolVisitor) addSymbol(name, kind string, node sitter.Node) {
	content := v.nodeContent(node)
	sig := extractSignature(node, v.data)
	v.symbols = append(v.symbols, CodeSymbol{
		Name:      name,
		Kind:      kind,
		DocPath:   v.docPath,
		Source:    content,
		Signature: sig,
	})
	v.addContainsEdge(name)
	v.addOwnerEdge(name, kind)
}

func (v *symbolVisitor) nodeContent(node sitter.Node) string {
	start := int(node.StartByte())
	end := int(node.EndByte())
	if start < 0 || end > len(v.data) || start >= end {
		return ""
	}
	return strings.TrimSpace(string(v.data[start:end]))
}

func walkTree(node *sitter.Node, v *symbolVisitor) {
	if node == nil {
		return
	}
	v.visit(*node)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		childKind := child.Kind()
		switch childKind {
		case "function_declaration", "function_definition",
			"function_item", "local_function_statement":
			if name := v.functionName(*child); name != "" {
				v.funcStack = append(v.funcStack, name)
			}
		case "method_declaration", "method_definition",
			"constructor_declaration":
			if name := v.methodName(*child); name != "" {
				v.funcStack = append(v.funcStack, name)
			}
		case "variable_declarator":
			if name := v.functionVariableName(*child); name != "" {
				v.funcStack = append(v.funcStack, name)
			}
		case "class_declaration", "class_definition",
			"struct_declaration", "struct_item",
			"record_declaration", "enum_declaration",
			"enum_item":
			if name := v.className(*child); name != "" {
				v.classStack = append(v.classStack, name)
			}
		case "interface_declaration", "interface_specifier",
			"trait_item":
			if name := v.interfaceName(*child); name != "" {
				v.classStack = append(v.classStack, name)
			}
		case "impl_item":
			if name := v.implItemTypeName(*child); name != "" {
				v.classStack = append(v.classStack, name)
			}
		}
		walkTree(child, v)
		switch childKind {
		case "function_declaration", "function_definition", "method_declaration", "method_definition",
			"variable_declarator", "function_item", "constructor_declaration", "local_function_statement":
			if len(v.funcStack) > 0 {
				v.funcStack = v.funcStack[:len(v.funcStack)-1]
			}
		case "class_declaration", "class_definition", "interface_declaration", "interface_specifier",
			"struct_declaration", "struct_item", "record_declaration", "enum_declaration",
			"enum_item", "trait_item", "impl_item":
			if len(v.classStack) > 0 {
				v.classStack = v.classStack[:len(v.classStack)-1]
			}
		}
	}
}

func (v *symbolVisitor) visit(node sitter.Node) {
	switch node.Kind() {
	case "function_declaration", "function_definition",
		"function_item", "local_function_statement":
		if name := v.functionName(node); name != "" {
			v.addSymbol(name, "function", node)
		}
	case "method_declaration", "method_definition":
		if name := v.methodName(node); name != "" {
			v.addSymbol(name, "method", node)
		}
	case "constructor_declaration":
		if name := v.methodName(node); name != "" {
			v.addSymbol(name, "method", node)
		}
	case "class_declaration", "class_definition":
		if name := v.className(node); name != "" {
			v.addSymbol(name, "class", node)
		}
	case "struct_declaration", "struct_item":
		if name := v.className(node); name != "" {
			v.addSymbol(name, "class", node)
		}
	case "record_declaration":
		if name := v.className(node); name != "" {
			v.addSymbol(name, "class", node)
		}
	case "enum_declaration", "enum_item":
		if name := v.className(node); name != "" {
			v.addSymbol(name, "class", node)
		}
	case "interface_declaration", "interface_specifier":
		if name := v.interfaceName(node); name != "" {
			v.addSymbol(name, "interface", node)
		}
	case "trait_item":
		if name := v.interfaceName(node); name != "" {
			v.addSymbol(name, "interface", node)
		}
	case "impl_item":
		// Rust impl blocks: extract methods inside, track as class context
		// The impl block itself is not a symbol, but its methods are.
	case "type_declaration", "type_spec":
		name, symbolKind := v.typeDeclarationSymbol(node)
		if name != "" && symbolKind != "" {
			v.addSymbol(name, symbolKind, node)
		}
	case "lexical_declaration", "variable_declaration":
		for _, fnName := range v.functionVariableNames(node) {
			v.addSymbol(fnName, "function", node)
		}
	case "variable_declarator":
		if name := v.functionVariableName(node); name != "" {
			v.addSymbol(name, "function", node)
		}
	case "class_heritage", "extends_clause", "base_list", "superclass":
		if len(v.classStack) > 0 {
			if target := v.extendsTarget(node); target.TargetName != "" {
				target.ReceiverTypeHint = v.currentReceiverType()
				v.addEdge(v.currentReceiverType(), target, "extends")
			}
		}
	case "call_expression", "invocation_expression":
		if len(v.funcStack) > 0 {
			if target := v.callExpressionTarget(node); target.TargetName != "" {
				target.ReceiverTypeHint = v.currentReceiverType()
				v.addEdge(v.funcStack[len(v.funcStack)-1], target, "calls")
			}
		}
	case "new_expression", "composite_literal", "object_creation_expression":
		if len(v.funcStack) > 0 {
			if target := v.instantiatedTarget(node); target.TargetName != "" {
				target.ReceiverTypeHint = v.currentReceiverType()
				v.addEdge(v.funcStack[len(v.funcStack)-1], target, "instantiates")
			}
		}
	case "import_specifier", "import_declaration", "import_clause", "import_statement":
		if target := v.importTarget(node); target.TargetName != "" {
			v.addImportEdge(v.importAnchor(), node)
		}
	}
}

func (v *symbolVisitor) functionName(node sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child != nil && child.Kind() == "identifier" {
			return v.extractNodeText(child)
		}
	}
	return v.extractNodeText(node.ChildByFieldName("name"))
}

func (v *symbolVisitor) methodName(node sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child != nil && (child.Kind() == "field_identifier" || child.Kind() == "identifier" || child.Kind() == "property_identifier" || child.Kind() == "private_property_identifier") {
			return v.extractNodeText(child)
		}
	}
	return v.namedChildText(node, "property_identifier", "private_property_identifier", "field_identifier", "identifier")
}

func (v *symbolVisitor) className(node sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child != nil && (child.Kind() == "type_identifier" || child.Kind() == "identifier") {
			return v.extractNodeText(child)
		}
	}
	return ""
}

func (v *symbolVisitor) interfaceName(node sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child != nil && (child.Kind() == "type_identifier" || child.Kind() == "identifier") {
			return v.extractNodeText(child)
		}
	}
	return ""
}

// implItemTypeName extracts the type name from a Rust impl block.
// e.g. `impl MyStruct { ... }` → "MyStruct"
func (v *symbolVisitor) implItemTypeName(node sitter.Node) string {
	// In Rust tree-sitter, impl_item has a "type" field
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		return v.extractNodeText(typeNode)
	}
	// Fallback: look for type_identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child != nil && child.Kind() == "type_identifier" {
			return v.extractNodeText(child)
		}
	}
	return ""
}

func (v *symbolVisitor) typeName(node sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child != nil && (child.Kind() == "type_identifier" || child.Kind() == "identifier") {
			return v.extractNodeText(child)
		}
	}
	return ""
}

func (v *symbolVisitor) typeDeclarationSymbol(node sitter.Node) (string, string) {
	name := v.typeName(node)
	if name == "" {
		return "", ""
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "struct_type", "class_declaration", "class_definition":
			return name, "class"
		case "interface_type", "interface_declaration", "interface_specifier":
			return name, "interface"
		case "type_spec":
			if nestedName, nestedKind := v.typeDeclarationSymbol(*child); nestedName != "" && nestedKind != "" {
				return nestedName, nestedKind
			}
		}
	}
	return "", ""
}

func (v *symbolVisitor) functionVariableName(node sitter.Node) string {
	if node.Kind() != "variable_declarator" {
		return ""
	}
	value := node.ChildByFieldName("value")
	if value == nil {
		value = v.firstNamedChildByKind(node, "arrow_function", "function_expression")
	}
	if value == nil {
		return ""
	}
	switch value.Kind() {
	case "arrow_function", "function_expression":
		name := node.ChildByFieldName("name")
		if name == nil {
			name = v.firstNamedChildByKind(node, "identifier")
		}
		if name != nil && name.Kind() == "identifier" {
			return v.extractNodeText(name)
		}
	}
	return ""
}

func (v *symbolVisitor) firstNamedChildByKind(node sitter.Node, kinds ...string) *sitter.Node {
	if node.ChildCount() == 0 {
		return nil
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		for _, kind := range kinds {
			if child.Kind() == kind {
				return child
			}
		}
		if nested := v.firstNamedChildByKind(*child, kinds...); nested != nil {
			return nested
		}
	}
	return nil
}

func (v *symbolVisitor) namedChildText(node sitter.Node, kinds ...string) string {
	child := v.firstNamedChildByKind(node, kinds...)
	if child == nil {
		return ""
	}
	return v.extractNodeText(child)
}

func (v *symbolVisitor) functionVariableNames(node sitter.Node) []string {
	if node.Kind() != "lexical_declaration" && node.Kind() != "variable_declaration" {
		return nil
	}
	var names []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil || child.Kind() != "variable_declarator" {
			continue
		}
		if name := v.functionVariableName(*child); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (v *symbolVisitor) extendsTarget(node sitter.Node) edgeTarget {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		if qualifier, name := v.typeExpressionParts(child); name != "" && name != v.currentReceiverType() {
			raw := name
			if qualifier != "" {
				raw = qualifier + "." + name
			}
			return edgeTarget{RawTarget: raw, TargetName: name, TargetQualifier: qualifier}
		}
	}
	return edgeTarget{}
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

func (v *symbolVisitor) extractNodeText(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	start := int(node.StartByte())
	end := int(node.EndByte())
	if start < 0 || end > len(v.data) || start >= end {
		return ""
	}
	return strings.TrimSpace(string(v.data[start:end]))
}

func extractSignature(node sitter.Node, data []byte) string {
	var buf bytes.Buffer
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "identifier" || kind == "field_identifier" || kind == "type_identifier" {
			if buf.Len() > 0 {
				buf.WriteString(".")
			}
			start := int(child.StartByte())
			end := int(child.EndByte())
			if start >= 0 && end <= len(data) {
				buf.Write(data[start:end])
			}
		}
		if kind == "parameter_list" || kind == "typed_parameter_list" {
			start := int(child.StartByte())
			end := int(child.EndByte())
			if start >= 0 && end <= len(data) {
				buf.Write(data[start:end])
			}
		}
	}
	return strings.TrimSpace(buf.String())
}
