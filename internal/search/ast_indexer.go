package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
)

// CodeSymbol represents a parsed symbol from a code file.
type CodeSymbol struct {
	Name      string
	Kind      string // "function", "method", "class", "interface", "file"
	DocPath   string // relative file path
	Content   string // natural-language description
	Signature string
	Source    string
}

// CodeChunkID returns the chunk ID for a code symbol.
func CodeChunkID(docPath, symbolName string) string {
	if symbolName == "" {
		return fmt.Sprintf("code::%s::__file__", docPath)
	}
	return fmt.Sprintf("code::%s::%s", docPath, symbolName)
}

// ToChunk converts a CodeSymbol to a search Chunk.
func (s *CodeSymbol) ToChunk() Chunk {
	id := CodeChunkID(s.DocPath, s.Name)
	content := s.Content
	if content == "" {
		if s.Name == "" {
			content = fmt.Sprintf("%s %s", s.Kind, s.DocPath)
		} else {
			content = fmt.Sprintf("%s %s — file: %s", s.Kind, s.Name, s.DocPath)
		}
	}
	return Chunk{
		ID:        id,
		Type:      ChunkTypeCode,
		Content:   content,
		DocPath:   s.DocPath,
		Field:     s.Kind,
		Name:      s.Name,
		Signature: s.Signature,
	}
}

// IndexAllFiles walks projectRoot, indexes all supported code files, returns symbols and edges.
// Does NOT persist to the vector store — caller embeds and saves.
func IndexAllFiles(projectRoot string, includeTests bool) ([]CodeSymbol, []CodeEdge, error) {
	return indexAllFiles(projectRoot, includeTests, nil)
}

// IndexAllFilesWithProgress walks projectRoot and invokes onFile after each candidate file is attempted.
func IndexAllFilesWithProgress(projectRoot string, includeTests bool, onFile func(string)) ([]CodeSymbol, []CodeEdge, error) {
	return indexAllFiles(projectRoot, includeTests, onFile)
}

func indexAllFiles(projectRoot string, includeTests bool, onFile func(string)) ([]CodeSymbol, []CodeEdge, error) {
	files, err := listCodeCandidateFiles(projectRoot, includeTests)
	if err != nil {
		return nil, nil, err
	}

	var symbols []CodeSymbol
	var edges []CodeEdge
	for _, rel := range files {
		absPath := filepath.Join(projectRoot, filepath.FromSlash(rel))
		syms, eds, parseErr := parseRawFile(rel, absPath)
		if onFile != nil {
			onFile(rel)
		}
		if parseErr != nil {
			continue
		}
		symbols = append(symbols, syms...)
		edges = append(edges, eds...)
	}
	return finalizeCodeParse(symbols, edges)
}

// IndexFile parses a single code file, returns symbols and edges found.
func IndexFile(docPath, absPath string) ([]CodeSymbol, []CodeEdge, error) {
	syms, eds, err := parseRawFile(docPath, absPath)
	if err != nil {
		return nil, nil, err
	}
	return finalizeCodeParse(syms, eds)
}

// CodeEdge represents an edge between two code symbols.
type CodeEdge struct {
	From     string // "code::<filepath>::<symbol>"
	To       string
	Type     string // "calls", "imports", "contains", "has_method", "instantiates", "implements", "extends"
	FromPath string
	ToPath   string

	RawTarget            string
	TargetName           string
	TargetQualifier      string
	TargetModuleHint     string
	ReceiverTypeHint     string
	ResolutionStatus     string
	ResolutionConfidence string
	ResolvedTo           string
}

// HasCodeIndex returns true if the code_edges table has any rows.
func HasCodeIndex(db *sql.DB) bool {
	if db == nil {
		return false
	}
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM code_edges").Scan(&count)
	return err == nil && count > 0
}

// GetCodeEdgeCount returns the number of code_edges rows.
func GetCodeEdgeCount(db *sql.DB) int {
	if db == nil {
		return 0
	}
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM code_edges").Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// SaveCodeEdges persists code edges to the code_edges table.
func SaveCodeEdges(db *sql.DB, edges []CodeEdge) error {
	if db == nil {
		return fmt.Errorf("no db")
	}
	_, err := db.Exec("DELETE FROM code_edges")
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO code_edges (
		from_id, to_id, edge_type, from_path, to_path,
		raw_target, target_name, target_qualifier, target_module_hint,
		receiver_type_hint, resolution_status, resolution_confidence, resolved_to
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, e := range edges {
		_, err := stmt.Exec(
			e.From, e.To, e.Type, e.FromPath, e.ToPath,
			e.RawTarget, e.TargetName, e.TargetQualifier, e.TargetModuleHint,
			e.ReceiverTypeHint, e.ResolutionStatus, e.ResolutionConfidence, e.ResolvedTo,
		)
		if err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()
	return tx.Commit()
}

// CodeChunkIDPattern returns a regex to match code chunk IDs.
func CodeChunkIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^code::(.+)::(.+)$`)
}

func loadCodeIgnorePatterns(projectRoot string) []string {
	var patterns []string

	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	if data, err := os.ReadFile(gitignorePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			patterns = append(patterns, strings.TrimSpace(line))
		}
	}

	configPath := filepath.Join(projectRoot, ".knowns", "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg models.Project
		if err := json.Unmarshal(data, &cfg); err == nil {
			patterns = append(patterns, cfg.Settings.CodeIntelligenceIgnore...)
		}
	}

	return patterns
}

func enrichCodeSymbolContent(symbols []CodeSymbol, edges []CodeEdge) []CodeSymbol {
	type edgeLists struct {
		calls        []string
		imports      []string
		instantiates []string
		implements   []string
	}

	edgeSummaryByID := make(map[string]*edgeLists, len(symbols))
	for _, edge := range edges {
		if !strings.HasPrefix(edge.From, "code::") {
			continue
		}
		lists := edgeSummaryByID[edge.From]
		if lists == nil {
			lists = &edgeLists{}
			edgeSummaryByID[edge.From] = lists
		}
		target := summarizeEdgeTarget(edge)
		if target == "" {
			continue
		}
		switch edge.Type {
		case "calls":
			lists.calls = append(lists.calls, target)
		case "imports":
			lists.imports = append(lists.imports, target)
		case "instantiates":
			lists.instantiates = append(lists.instantiates, target)
		case "implements":
			lists.implements = append(lists.implements, target)
		}
	}

	enriched := make([]CodeSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		updated := symbol
		id := CodeChunkID(symbol.DocPath, symbol.Name)
		lists := edgeSummaryByID[id]

		var parts []string
		if symbol.Name == "" {
			parts = append(parts, "file "+symbol.DocPath)
		} else if strings.TrimSpace(symbol.Signature) != "" {
			parts = append(parts, symbol.Signature)
		} else {
			parts = append(parts, symbol.Kind+" "+symbol.Name)
		}
		parts = append(parts, "file: "+symbol.DocPath)
		if lists != nil {
			if values := uniqueSorted(lists.calls); len(values) > 0 {
				parts = append(parts, "calls: "+strings.Join(values, ", "))
			}
			if values := uniqueSorted(lists.instantiates); len(values) > 0 {
				parts = append(parts, "instantiates: "+strings.Join(values, ", "))
			}
			if values := uniqueSorted(lists.imports); len(values) > 0 {
				parts = append(parts, "imports: "+strings.Join(values, ", "))
			}
			if values := uniqueSorted(lists.implements); len(values) > 0 {
				parts = append(parts, "implements: "+strings.Join(values, ", "))
			}
		}

		summary := strings.Join(parts, " - ")
		if symbol.Kind == "file" || strings.TrimSpace(symbol.Source) == "" {
			updated.Content = summary
		} else {
			updated.Content = summary + "\n\n" + strings.TrimSpace(symbol.Source)
		}
		enriched = append(enriched, updated)
	}

	return enriched
}

func summarizeEdgeTarget(edge CodeEdge) string {
	if strings.HasPrefix(edge.To, "code::") {
		if matches := CodeChunkIDPattern().FindStringSubmatch(edge.To); len(matches) == 3 {
			if matches[2] == "__file__" {
				return matches[1]
			}
			return matches[2]
		}
	}
	return firstNonEmpty(edge.TargetName, edge.RawTarget, edge.TargetModuleHint, edge.ResolvedTo)
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}
