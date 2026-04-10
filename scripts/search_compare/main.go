package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type searchResult struct {
	Type  string  `json:"type"`
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Path  string  `json:"path,omitempty"`
	Score float64 `json:"score"`
}

type benchmarkCase struct {
	Query          string
	Expected       []string
	FailureHints   []string
	Notes          string
	IgnoreDocPaths []string
}

type modeReport struct {
	Mode    string       `json:"mode"`
	Cases   []caseReport `json:"cases"`
	Summary modeSummary  `json:"summary"`
	Skipped bool         `json:"skipped,omitempty"`
	SkipWhy string       `json:"skipWhy,omitempty"`
}

type caseReport struct {
	Query    string   `json:"query"`
	Verdict  string   `json:"verdict"`
	Expected []string `json:"expected"`
	Observed []string `json:"observed"`
	Why      string   `json:"why"`
	Notes    string   `json:"notes,omitempty"`
}

type modeSummary struct {
	Total    int `json:"total"`
	Pass     int `json:"pass"`
	Partial  int `json:"partial"`
	Fail     int `json:"fail"`
	Skipped  int `json:"skipped"`
	TopKUsed int `json:"topKUsed"`
}

func main() {
	limit := flag.Int("limit", 3, "number of top results to show per query")
	jsonOut := flag.Bool("json", false, "emit report as JSON")
	modesFlag := flag.String("modes", "keyword,hybrid,bm25", "comma-separated modes to compare")
	flag.Parse()

	modes := splitCSV(*modesFlag)
	if len(modes) == 0 {
		modes = []string{"keyword", "hybrid", "bm25"}
	}

	var reports []modeReport
	for _, mode := range modes {
		reports = append(reports, runMode(mode, benchmarkCases(), *limit))
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(reports)
		return
	}

	printReports(reports, *limit)
}

func runMode(mode string, cases []benchmarkCase, limit int) modeReport {
	report := modeReport{Mode: mode}
	if !modeAvailable(mode) {
		report.Skipped = true
		report.SkipWhy = "mode not implemented yet"
		report.Summary = modeSummary{Total: len(cases), Skipped: len(cases), TopKUsed: limit}
		return report
	}

	for _, tc := range cases {
		results, err := runSearch(mode, tc.Query)
		if err != nil {
			report.Cases = append(report.Cases, caseReport{
				Query:    tc.Query,
				Verdict:  "fail",
				Expected: tc.Expected,
				Observed: nil,
				Why:      err.Error(),
				Notes:    tc.Notes,
			})
			continue
		}
		filtered := filterResults(results, tc.IgnoreDocPaths)
		observed := topResultKeys(filtered, limit)
		verdict, why := evaluateCase(tc, filtered)
		report.Cases = append(report.Cases, caseReport{
			Query:    tc.Query,
			Verdict:  verdict,
			Expected: tc.Expected,
			Observed: observed,
			Why:      why,
			Notes:    tc.Notes,
		})
	}

	report.Summary = summarize(report.Cases, limit)
	return report
}

func runSearch(mode, query string) ([]searchResult, error) {
	args := []string{"run", "./cmd/knowns", "search", query, "--json"}
	switch mode {
	case "keyword":
		args = append(args, "--keyword")
	case "hybrid":
		// default CLI mode is hybrid when semantic is available, keyword fallback otherwise
	case "bm25":
		return nil, fmt.Errorf("bm25 backend not implemented")
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var results []searchResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results, nil
}

func printReports(reports []modeReport, limit int) {
	fmt.Println("Search Mode Comparison")
	fmt.Println("======================")
	for _, report := range reports {
		fmt.Printf("\nMode: %s\n", strings.ToUpper(report.Mode))
		if report.Skipped {
			fmt.Printf("Skipped: %s\n", report.SkipWhy)
			continue
		}
		fmt.Printf("Total: %d  Pass: %d  Partial: %d  Fail: %d\n", report.Summary.Total, report.Summary.Pass, report.Summary.Partial, report.Summary.Fail)
		for _, c := range report.Cases {
			fmt.Printf("- %-32s %-7s top %d: %s\n", c.Query, strings.ToUpper(c.Verdict), limit, strings.Join(c.Observed, ", "))
		}
	}
}

func summarize(cases []caseReport, limit int) modeSummary {
	s := modeSummary{Total: len(cases), TopKUsed: limit}
	for _, c := range cases {
		switch c.Verdict {
		case "pass":
			s.Pass++
		case "partial":
			s.Partial++
		case "skip":
			s.Skipped++
		default:
			s.Fail++
		}
	}
	return s
}

func modeAvailable(mode string) bool {
	switch mode {
	case "keyword", "hybrid":
		return true
	case "bm25":
		return false
	default:
		return false
	}
}

func benchmarkCases() []benchmarkCase {
	return []benchmarkCase{
		{Query: "semantic search", Expected: []string{"specs/semantic-search", "guides/semantic-search-guide"}, FailureHints: []string{"title underweight", "common-term inflation"}, Notes: "exact title/spec-guide lookup", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "rag retrieval foundation", Expected: []string{"specs/rag-retrieval-foundation"}, FailureHints: []string{"title underweight", "multi-word weakness"}, Notes: "exact spec lookup", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "3 layer memory system", Expected: []string{"specs/3-layer-memory-system", "learnings/learning-3-layer-memory-system"}, FailureHints: []string{"multi-word weakness", "common-term inflation"}, Notes: "phrase variation without punctuation", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "guides/semantic-search-guide", Expected: []string{"guides/semantic-search-guide"}, FailureHints: []string{"path underweight"}, Notes: "path lookup", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "knowns-go rewrite", Expected: []string{"specs/knowns-go-rewrite"}, FailureHints: []string{"substring noise", "title underweight"}, Notes: "dashed title/path lookup", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "search", Expected: []string{"specs/semantic-search", "guides/semantic-search-guide"}, FailureHints: []string{"common-term inflation", "long-doc bias"}, Notes: "common term stress test", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "task", Expected: []string{"guides/workflow-guide", "features/global-task-modal"}, FailureHints: []string{"common-term inflation", "task/doc skew"}, Notes: "generic task term", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "memory", Expected: []string{"specs/3-layer-memory-system", "learnings/learning-3-layer-memory-system"}, FailureHints: []string{"common-term inflation", "memory under-rank"}, Notes: "generic memory term", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "spec", Expected: []string{"specs/semantic-search", "specs/rag-retrieval-foundation"}, FailureHints: []string{"common-term inflation", "long-doc bias"}, Notes: "generic spec term", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "search quality improvements", Expected: []string{"specs/semantic-search-quality-improvements"}, FailureHints: []string{"multi-word weakness", "title underweight"}, Notes: "multi-word lexical intent", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "memory entries search integration", Expected: []string{"szd42a"}, FailureHints: []string{"task/doc skew", "multi-word weakness"}, Notes: "descriptive task query", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "heuristic reranker search pipeline", Expected: []string{"x6db7x"}, FailureHints: []string{"task/doc skew", "title underweight"}, Notes: "implementation task query", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "chunking strategies research", Expected: []string{"research/rag-chunking-strategies-research"}, FailureHints: []string{"multi-word weakness", "path underweight"}, Notes: "research doc specificity", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "retrieval orchestration", Expected: []string{"511yfk", "specs/rag-retrieval-foundation"}, FailureHints: []string{"task/doc skew"}, Notes: "doc/task balance", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
		{Query: "plan", Expected: []string{"guides/workflow-guide", "specs/sdd-spec-driven-development"}, FailureHints: []string{"substring noise", "long-doc bias"}, Notes: "short token boundary risk", IgnoreDocPaths: []string{"research/keyword-search-benchmark-matrix"}},
	}
}

func filterResults(results []searchResult, ignoreDocPaths []string) []searchResult {
	ignore := make(map[string]bool, len(ignoreDocPaths))
	for _, path := range ignoreDocPaths {
		ignore[path] = true
	}
	filtered := make([]searchResult, 0, len(results))
	for _, result := range results {
		if result.Type == "doc" && ignore[result.ID] {
			continue
		}
		filtered = append(filtered, result)
	}
	return filtered
}

func topResultKeys(results []searchResult, limit int) []string {
	if limit > len(results) {
		limit = len(results)
	}
	keys := make([]string, 0, limit)
	for _, result := range results[:limit] {
		keys = append(keys, resultKey(result))
	}
	return keys
}

func evaluateCase(tc benchmarkCase, results []searchResult) (string, string) {
	if len(results) == 0 {
		return "fail", "no results"
	}

	maxCheck := min(3, len(results))
	matchedTop3 := 0
	for _, expected := range tc.Expected {
		for _, result := range results[:maxCheck] {
			if resultKey(result) == expected {
				matchedTop3++
				break
			}
		}
	}

	if matchedTop3 == len(tc.Expected) || (len(tc.Expected) > 0 && resultKey(results[0]) == tc.Expected[0]) {
		return "pass", "expected result is in top results"
	}

	for _, expected := range tc.Expected {
		for _, result := range results {
			if resultKey(result) == expected {
				return "partial", "expected result found but ranked below top 3"
			}
		}
	}

	if len(tc.FailureHints) > 0 {
		return "fail", tc.FailureHints[0]
	}
	return "fail", "expected result missing"
}

func resultKey(r searchResult) string {
	return r.ID
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(wd) == "search_compare" {
		return filepath.Dir(filepath.Dir(wd))
	}
	return wd
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
