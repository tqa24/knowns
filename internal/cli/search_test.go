package cli
import (
	"strings"
	"testing"

	"github.com/howznguyen/knowns/internal/models"
)

func TestSprintPlainRetrieval(t *testing.T) {
	resp := &models.RetrievalResponse{
		Query: "rag retrieval foundation",
		Mode:  "keyword",
		Candidates: []models.RetrievalCandidate{
			{
				Type:        "doc",
				ID:          "specs/rag-retrieval-foundation",
				Title:       "RAG Retrieval Foundation",
				Score:       1,
				DirectMatch: true,
				Citation: models.Citation{
					Type: "doc",
					Path: "specs/rag-retrieval-foundation",
				},
				Snippet: "Specification for retrieval foundation across docs, tasks, and memories",
			},
		},
		ContextPack: models.ContextPack{
			Items: []models.ContextItem{
				{
					Type:        "doc",
					ID:          "specs/rag-retrieval-foundation",
					Title:       "RAG Retrieval Foundation",
					DirectMatch: true,
					Citation: models.Citation{
						Type: "doc",
						Path: "specs/rag-retrieval-foundation",
					},
					Content: "Build a shared retrieval foundation for Knowns.",
				},
			},
		},
	}

	got := sprintPlainRetrieval(resp)
	for _, want := range []string{
		"Query: rag retrieval foundation",
		"Candidates: 1",
		"[DOC] RAG Retrieval Foundation (specs/rag-retrieval-foundation)",
		"Citation: doc:specs/rag-retrieval-foundation",
		"Context Pack:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}
