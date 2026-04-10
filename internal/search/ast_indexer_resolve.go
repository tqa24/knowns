package search

import (
	"path/filepath"
	"strings"
)

func ResolveCodeEdges(symbols []CodeSymbol, edges []CodeEdge) []CodeEdge {
	ctx := buildResolutionContext(symbols)
	resolved := make([]CodeEdge, 0, len(edges))
	for _, e := range edges {
		resolved = append(resolved, ctx.resolveEdge(e))
	}
	return resolved
}

type resolutionContext struct {
	byName   map[string][]CodeSymbol
	byDoc    map[string]map[string]CodeSymbol
	byModule map[string]map[string]CodeSymbol
}

func buildResolutionContext(symbols []CodeSymbol) resolutionContext {
	ctx := resolutionContext{
		byName:   make(map[string][]CodeSymbol),
		byDoc:    make(map[string]map[string]CodeSymbol),
		byModule: make(map[string]map[string]CodeSymbol),
	}
	for _, s := range symbols {
		ctx.byName[s.Name] = append(ctx.byName[s.Name], s)
		if ctx.byDoc[s.DocPath] == nil {
			ctx.byDoc[s.DocPath] = make(map[string]CodeSymbol)
		}
		ctx.byDoc[s.DocPath][s.Name] = s
		moduleKey := strings.TrimSuffix(filepath.ToSlash(s.DocPath), filepath.Ext(s.DocPath))
		if ctx.byModule[moduleKey] == nil {
			ctx.byModule[moduleKey] = make(map[string]CodeSymbol)
		}
		ctx.byModule[moduleKey][s.Name] = s
	}
	return ctx
}

func (ctx resolutionContext) resolveEdge(e CodeEdge) CodeEdge {
	resolved := e
	resolved.ResolutionStatus = "unresolved"
	resolved.ResolutionConfidence = "low"
	resolved.ResolvedTo = ""

	if strings.HasPrefix(e.To, "code::") {
		resolved.ResolutionStatus = "resolved_internal"
		resolved.ResolutionConfidence = "high"
		resolved.ResolvedTo = e.To
		return resolved
	}

	if e.Type == "imports" {
		if importedFile := ctx.resolveImportedFileCandidate(e); importedFile != nil {
			resolved.To = CodeChunkID(importedFile.DocPath, importedFile.Name)
			resolved.ToPath = importedFile.DocPath
			resolved.ResolutionStatus = "resolved_internal"
			resolved.ResolutionConfidence = "high"
			resolved.ResolvedTo = resolved.To
			return resolved
		}
	}

	if e.Type == "extends" {
		if sameDoc := ctx.resolveSameDocTypeCandidate(e); sameDoc != nil {
			resolved.To = CodeChunkID(sameDoc.DocPath, sameDoc.Name)
			resolved.ToPath = sameDoc.DocPath
			resolved.ResolutionStatus = "resolved_internal"
			resolved.ResolutionConfidence = "high"
			resolved.ResolvedTo = resolved.To
			return resolved
		}
	}

	if docSyms := ctx.byDoc[e.FromPath]; docSyms != nil {
		if sym, ok := docSyms[e.TargetName]; ok {
			resolved.To = CodeChunkID(sym.DocPath, sym.Name)
			resolved.ToPath = sym.DocPath
			resolved.ResolutionStatus = "resolved_internal"
			resolved.ResolutionConfidence = "high"
			resolved.ResolvedTo = resolved.To
			return resolved
		}
	}

	if e.TargetModuleHint != "" {
		for _, candidate := range resolveJSImportCandidate(e.FromPath, e.TargetModuleHint) {
			moduleSyms := ctx.byModule[candidate]
			if moduleSyms == nil {
				continue
			}
			if sym, ok := moduleSyms[e.TargetName]; ok {
				resolved.To = CodeChunkID(sym.DocPath, sym.Name)
				resolved.ToPath = sym.DocPath
				resolved.ResolutionStatus = "resolved_internal"
				resolved.ResolutionConfidence = "high"
				resolved.ResolvedTo = resolved.To
				return resolved
			}
		}
	}

	if samePkg := ctx.resolveSamePackageGoCandidate(e); samePkg != nil {
		resolved.To = CodeChunkID(samePkg.DocPath, samePkg.Name)
		resolved.ToPath = samePkg.DocPath
		resolved.ResolutionStatus = "resolved_internal"
		resolved.ResolutionConfidence = "high"
		resolved.ResolvedTo = resolved.To
		return resolved
	}

	if candidates := ctx.byName[e.TargetName]; len(candidates) == 1 {
		if sameModule := ctx.resolveSameModuleCandidate(e); sameModule != nil {
			resolved.To = CodeChunkID(sameModule.DocPath, sameModule.Name)
			resolved.ToPath = sameModule.DocPath
			resolved.ResolutionStatus = "resolved_internal"
			resolved.ResolutionConfidence = "high"
			resolved.ResolvedTo = resolved.To
			return resolved
		}

		sym := candidates[0]
		resolved.To = CodeChunkID(sym.DocPath, sym.Name)
		resolved.ToPath = sym.DocPath
		resolved.ResolutionStatus = "resolved_internal"
		resolved.ResolutionConfidence = "medium"
		resolved.ResolvedTo = resolved.To
		return resolved
	}

	if e.TargetModuleHint != "" || e.TargetQualifier != "" {
		resolved.ResolutionStatus = "resolved_external"
		resolved.ResolutionConfidence = "medium"
		resolved.ResolvedTo = firstNonEmpty(e.RawTarget, e.TargetModuleHint, e.TargetName)
		return resolved
	}

	resolved.ResolvedTo = firstNonEmpty(e.RawTarget, e.TargetName)
	return resolved
}

func (ctx resolutionContext) resolveImportedFileCandidate(e CodeEdge) *CodeSymbol {
	if e.TargetModuleHint == "" {
		return nil
	}
	for _, candidate := range resolveJSImportCandidate(e.FromPath, e.TargetModuleHint) {
		docSyms := ctx.byDoc[candidate]
		if docSyms == nil {
			continue
		}
		if fileSym, ok := docSyms[""]; ok {
			return &fileSym
		}
	}
	return nil
}

func (ctx resolutionContext) resolveSameDocTypeCandidate(e CodeEdge) *CodeSymbol {
	if e.FromPath == "" || e.TargetName == "" {
		return nil
	}
	docSyms := ctx.byDoc[e.FromPath]
	if docSyms == nil {
		return nil
	}
	if sym, ok := docSyms[e.TargetName]; ok && (sym.Kind == "class" || sym.Kind == "interface") {
		return &sym
	}
	return nil
}

func (ctx resolutionContext) resolveSamePackageGoCandidate(e CodeEdge) *CodeSymbol {
	if filepath.Ext(e.FromPath) != ".go" {
		return nil
	}
	fromPkg := goPackageKey(e.FromPath)
	if fromPkg == "" {
		return nil
	}
	candidates := ctx.byName[e.TargetName]
	var match *CodeSymbol
	for i := range candidates {
		candidate := candidates[i]
		if filepath.Ext(candidate.DocPath) != ".go" {
			continue
		}
		if goPackageKey(candidate.DocPath) != fromPkg {
			continue
		}
		if match != nil {
			return nil
		}
		match = &candidate
	}
	return match
}

func (ctx resolutionContext) resolveSameModuleCandidate(e CodeEdge) *CodeSymbol {
	fromModule := strings.TrimSuffix(filepath.ToSlash(e.FromPath), filepath.Ext(e.FromPath))
	if fromModule == "" {
		return nil
	}
	candidates := ctx.byName[e.TargetName]
	var match *CodeSymbol
	for i := range candidates {
		candidate := candidates[i]
		candidateModule := strings.TrimSuffix(filepath.ToSlash(candidate.DocPath), filepath.Ext(candidate.DocPath))
		if candidateModule != fromModule {
			continue
		}
		if match != nil {
			return nil
		}
		match = &candidate
	}
	return match
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
