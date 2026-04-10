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
	Verdict        string
	Observed       []string
	Why            string
}

func main() {
	limit := flag.Int("limit", 3, "number of top results to show per query")
	jsonOut := flag.Bool("json", false, "emit report as JSON")
	flag.Parse()

	cases := []benchmarkCase{
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

	for i := range cases {
		results, err := runSearch(cases[i].Query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchmark %q failed: %v\n", cases[i].Query, err)
			os.Exit(1)
		}
		filtered := filterResults(results, cases[i].IgnoreDocPaths)
		cases[i].Observed = topResultKeys(filtered, *limit)
		cases[i].Verdict, cases[i].Why = evaluateCase(cases[i], filtered)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(cases)
		return
	}

	printReport(cases, *limit)
}

func runSearch(query string) ([]searchResult, error) {
	cmd := exec.Command("go", "run", "./cmd/knowns", "search", query, "--keyword", "--json")
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

func printReport(cases []benchmarkCase, limit int) {
	passes, partials, fails := 0, 0, 0
	for _, tc := range cases {
		switch tc.Verdict {
		case "pass":
			passes++
		case "partial":
			partials++
		default:
			fails++
		}
	}

	fmt.Printf("Keyword Search Benchmark\n")
	fmt.Printf("========================\n")
	fmt.Printf("Total: %d  Pass: %d  Partial: %d  Fail: %d\n\n", len(cases), passes, partials, fails)

	for _, tc := range cases {
		fmt.Printf("Query: %s\n", tc.Query)
		fmt.Printf("Verdict: %s\n", strings.ToUpper(tc.Verdict))
		fmt.Printf("Expected: %s\n", strings.Join(tc.Expected, ", "))
		fmt.Printf("Observed top %d: %s\n", limit, strings.Join(tc.Observed, ", "))
		if tc.Why != "" {
			fmt.Printf("Why: %s\n", tc.Why)
		}
		if tc.Notes != "" {
			fmt.Printf("Notes: %s\n", tc.Notes)
		}
		fmt.Println()
	}
}

func resultKey(r searchResult) string {
	if r.Type == "doc" {
		return r.ID
	}
	return r.ID
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(wd) == "scripts" {
		return filepath.Dir(wd)
	}
	return wd
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
