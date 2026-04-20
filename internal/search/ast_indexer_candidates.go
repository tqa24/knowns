package search

import (
	"os"
	"path/filepath"
	"strings"
)

func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py":
		return true
	}
	return false
}

func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(path, "_test.go") ||
		strings.HasSuffix(path, ".spec.ts") ||
		strings.HasSuffix(path, ".test.ts") ||
		strings.HasSuffix(path, ".spec.js") ||
		strings.HasSuffix(path, ".test.js") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".test.js")
}

func listCodeCandidateFiles(projectRoot string, includeTests bool) ([]string, error) {
	patterns := loadCodeIgnorePatterns(projectRoot)
	matcher := newCodeMatcher(patterns)

	var files []string
	walkErr := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(projectRoot, path)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if info.IsDir() {
			base := filepath.Base(rel)
			if base == "node_modules" || base == "__pycache__" || base == ".git" {
				return filepath.SkipDir
			}
			if matcher.ShouldIgnore(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if matcher.ShouldIgnore(rel) {
			return nil
		}
		if !isCodeFile(rel) {
			return nil
		}
		if !includeTests && isTestFile(rel) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return files, nil
}

// ListCodeCandidateFiles returns the relative code file paths that code ingest will attempt to parse.
func ListCodeCandidateFiles(projectRoot string, includeTests bool) ([]string, error) {
	return listCodeCandidateFiles(projectRoot, includeTests)
}
