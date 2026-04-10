package search

import (
	"testing"

	"github.com/howznguyen/knowns/internal/models"
)

func TestSelectRankedNeighborCandidates_PrioritizesEdgeTypeAndDedupe(t *testing.T) {
	match := models.SearchResult{ID: "code::a::target", Path: "a.ts"}
	candidates := []codeNeighborCandidate{
		buildCodeNeighborCandidate(match, CodeNeighborEdge{From: "code::a::Owner", To: "code::a::m1", Type: "has_method", FromPath: "a.ts", ToPath: "a.ts"}, Chunk{ID: "code::a::m1", DocPath: "a.ts"}),
		buildCodeNeighborCandidate(match, CodeNeighborEdge{From: "code::a::Owner", To: "code::a::m2", Type: "has_method", FromPath: "a.ts", ToPath: "a.ts"}, Chunk{ID: "code::a::m2", DocPath: "a.ts"}),
		buildCodeNeighborCandidate(match, CodeNeighborEdge{From: "code::a::Owner", To: "code::a::m3", Type: "has_method", FromPath: "a.ts", ToPath: "a.ts"}, Chunk{ID: "code::a::m3", DocPath: "a.ts"}),
		buildCodeNeighborCandidate(match, CodeNeighborEdge{From: "code::a::Owner", To: "code::a::m4", Type: "has_method", FromPath: "a.ts", ToPath: "a.ts"}, Chunk{ID: "code::a::m4", DocPath: "a.ts"}),
		buildCodeNeighborCandidate(match, CodeNeighborEdge{From: "code::a::target", To: "code::a::callee", Type: "calls", FromPath: "a.ts", ToPath: "a.ts"}, Chunk{ID: "code::a::callee", DocPath: "a.ts"}),
		buildCodeNeighborCandidate(match, CodeNeighborEdge{From: "code::a::__file__", To: "code::a::target", Type: "contains", FromPath: "a.ts", ToPath: "a.ts"}, Chunk{ID: "code::a::__file__", DocPath: "a.ts"}),
	}

	selected, truncated := selectRankedNeighborCandidates(candidates, 10)
	if !truncated {
		t.Fatal("expected truncation due to bucket dedupe")
	}
	if len(selected) != 5 {
		t.Fatalf("selected = %d, want 5", len(selected))
	}
	if selected[0].edge.Type != "calls" {
		t.Fatalf("first edge type = %s, want calls", selected[0].edge.Type)
	}
	hasMethodCount := 0
	for _, item := range selected {
		if item.edge.Type == "has_method" {
			hasMethodCount++
		}
	}
	if hasMethodCount != 3 {
		t.Fatalf("has_method count = %d, want 3", hasMethodCount)
	}
}
