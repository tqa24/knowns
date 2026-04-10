package search

import (
	"path/filepath"
	"regexp"
	"strings"
)

func buildImplementsEdges(symbols []CodeSymbol) []CodeEdge {
	fileMethodSets := make(map[string]map[string]bool)
	packageMethodSets := make(map[string]map[string]map[string]bool)
	interfaces := make([]CodeSymbol, 0)
	classes := make([]CodeSymbol, 0)

	for _, sym := range symbols {
		switch sym.Kind {
		case "method":
			methods := fileMethodSets[sym.DocPath]
			if methods == nil {
				methods = make(map[string]bool)
				fileMethodSets[sym.DocPath] = methods
			}
			methods[sym.Name] = true

			if filepath.Ext(sym.DocPath) == ".go" {
				pkgKey := goPackageKey(sym.DocPath)
				receiverType := goMethodReceiverType(codeSymbolSource(sym))
				if pkgKey != "" && receiverType != "" {
					if packageMethodSets[pkgKey] == nil {
						packageMethodSets[pkgKey] = make(map[string]map[string]bool)
					}
					if packageMethodSets[pkgKey][receiverType] == nil {
						packageMethodSets[pkgKey][receiverType] = make(map[string]bool)
					}
					packageMethodSets[pkgKey][receiverType][sym.Name] = true
				}
			}
		case "interface":
			interfaces = append(interfaces, sym)
		case "class":
			classes = append(classes, sym)
		}
	}

	edges := make([]CodeEdge, 0)
	seen := make(map[string]bool)
	for _, classSym := range classes {
		for _, ifaceSym := range interfaces {
			if ifaceSym.Name == classSym.Name {
				continue
			}
			ifaceMethods := interfaceMethodNames(codeSymbolSource(ifaceSym))
			if len(ifaceMethods) == 0 {
				continue
			}

			classMethods := methodsForImplementsClass(classSym, ifaceSym, fileMethodSets, packageMethodSets)
			if len(classMethods) == 0 {
				continue
			}

			matched := true
			for _, methodName := range ifaceMethods {
				if !classMethods[methodName] {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
			key := CodeChunkID(classSym.DocPath, classSym.Name) + "->" + CodeChunkID(ifaceSym.DocPath, ifaceSym.Name)
			if seen[key] {
				continue
			}
			seen[key] = true
			edges = append(edges, CodeEdge{
				From:                 CodeChunkID(classSym.DocPath, classSym.Name),
				To:                   CodeChunkID(ifaceSym.DocPath, ifaceSym.Name),
				Type:                 "implements",
				FromPath:             classSym.DocPath,
				ToPath:               ifaceSym.DocPath,
				ResolutionStatus:     "resolved_internal",
				ResolutionConfidence: "medium",
				ResolvedTo:           CodeChunkID(ifaceSym.DocPath, ifaceSym.Name),
			})
		}
	}

	return edges
}

func methodsForImplementsClass(classSym, ifaceSym CodeSymbol, fileMethodSets map[string]map[string]bool, packageMethodSets map[string]map[string]map[string]bool) map[string]bool {
	if filepath.Ext(classSym.DocPath) == ".go" && filepath.Ext(ifaceSym.DocPath) == ".go" {
		classPkg := goPackageKey(classSym.DocPath)
		ifacePkg := goPackageKey(ifaceSym.DocPath)
		if classPkg != ifacePkg {
			return nil
		}
		if pkgMethods := packageMethodSets[classPkg]; pkgMethods != nil {
			return pkgMethods[classSym.Name]
		}
		return nil
	}
	if classSym.DocPath != ifaceSym.DocPath {
		return nil
	}
	return fileMethodSets[classSym.DocPath]
}

func goPackageKey(docPath string) string {
	return filepath.ToSlash(filepath.Dir(docPath))
}

func goMethodReceiverType(content string) string {
	match := regexp.MustCompile(`(?m)^func\s*\(([^)]*)\)`).FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	receiver := strings.TrimSpace(match[1])
	if receiver == "" {
		return ""
	}
	parts := strings.Fields(receiver)
	typeName := parts[len(parts)-1]
	typeName = strings.TrimLeft(typeName, "*[]")
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		typeName = typeName[idx+1:]
	}
	if idx := strings.Index(typeName, "["); idx >= 0 {
		typeName = typeName[:idx]
	}
	return strings.TrimSpace(typeName)
}

func interfaceMethodNames(content string) []string {
	matches := regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(`).FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	names := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		name := strings.TrimSpace(match[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func codeSymbolSource(sym CodeSymbol) string {
	if strings.TrimSpace(sym.Source) != "" {
		return sym.Source
	}
	return sym.Content
}
