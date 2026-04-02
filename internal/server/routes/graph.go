package routes

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/storage"
)

// GraphRoutes handles /api/graph endpoints.
type GraphRoutes struct {
	store *storage.Store
}

// Register wires the graph routes onto r.
func (gr *GraphRoutes) Register(r chi.Router) {
	r.Get("/graph", gr.graph)
}

// GraphNode represents a single entity in the knowledge graph.
type GraphNode struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`  // "task", "doc", "template"
	Label string                 `json:"label"`
	Data  map[string]interface{} `json:"data"`
}

// GraphEdge represents a relationship between two nodes.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "parent", "spec", "template-doc", "mention"
}

// Reference-detection regexes (same as validate package).
var (
	graphTaskRefRE   = regexp.MustCompile(`@task-([a-z0-9]+)`)
	graphDocRefRE    = regexp.MustCompile(`@doc/([^\s\)]+)`)
	graphMemoryRefRE = regexp.MustCompile(`@memory-([a-z0-9]+)`)
)

// graph returns the full knowledge graph of tasks, docs, and templates.
//
// GET /api/graph
func (gr *GraphRoutes) graph(w http.ResponseWriter, r *http.Request) {
	tasks, err := gr.store.Tasks.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	docs, err := gr.store.Docs.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build lookup sets for validating references.
	taskIDs := make(map[string]bool, len(tasks))
	docPaths := make(map[string]bool, len(docs))

	// Load memory entries.
	memories, _ := gr.store.Memory.List("")
	memoryIDs := make(map[string]bool, len(memories))

	var nodes []GraphNode
	var edges []GraphEdge

	// --- Task nodes ---
	for _, t := range tasks {
		taskIDs[t.ID] = true
		nodes = append(nodes, GraphNode{
			ID:    "task:" + t.ID,
			Type:  "task",
			Label: t.Title,
			Data: map[string]interface{}{
				"status":   t.Status,
				"priority": t.Priority,
				"labels":   t.Labels,
				"assignee": t.Assignee,
			},
		})
	}

	// --- Doc nodes ---
	for _, d := range docs {
		docPaths[d.Path] = true
		nodes = append(nodes, GraphNode{
			ID:    "doc:" + d.Path,
			Type:  "doc",
			Label: d.Title,
			Data: map[string]interface{}{
				"tags":        d.Tags,
				"description": d.Description,
			},
		})
	}

	// --- Memory nodes ---
	for _, m := range memories {
		memoryIDs[m.ID] = true
		nodes = append(nodes, GraphNode{
			ID:    "memory:" + m.ID,
			Type:  "memory",
			Label: m.Title,
			Data: map[string]interface{}{
				"layer":    m.Layer,
				"category": m.Category,
				"tags":     m.Tags,
			},
		})
	}

	// --- Edges from task fields ---
	for _, t := range tasks {
		src := "task:" + t.ID

		// Parent-child
		if t.Parent != "" && taskIDs[t.Parent] {
			edges = append(edges, GraphEdge{Source: src, Target: "task:" + t.Parent, Type: "parent"})
		}

		// Spec link
		if t.Spec != "" && docPaths[t.Spec] {
			edges = append(edges, GraphEdge{Source: src, Target: "doc:" + t.Spec, Type: "spec"})
		}

		// Cross-references in content
		content := strings.Join([]string{t.Description, t.ImplementationPlan, t.ImplementationNotes}, "\n")
		edges = append(edges, extractMentions(src, content, taskIDs, docPaths, memoryIDs)...)
	}

	// --- Edges from doc content ---
	for _, d := range docs {
		src := "doc:" + d.Path
		edges = append(edges, extractMentions(src, d.Content, taskIDs, docPaths, memoryIDs)...)
	}

	// --- Edges from memory content ---
	for _, m := range memories {
		src := "memory:" + m.ID
		edges = append(edges, extractMentions(src, m.Content, taskIDs, docPaths, memoryIDs)...)
	}

	// Ensure non-nil slices for JSON.
	if nodes == nil {
		nodes = []GraphNode{}
	}
	if edges == nil {
		edges = []GraphEdge{}
	}

	// Deduplicate edges.
	edges = deduplicateEdges(edges)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

// extractMentions scans content for @task-<id> and @doc/<path> references and
// returns edges from src to each valid target.
func extractMentions(src, content string, taskIDs map[string]bool, docPaths map[string]bool, memoryIDs map[string]bool) []GraphEdge {
	var edges []GraphEdge

	for _, m := range graphTaskRefRE.FindAllStringSubmatch(content, -1) {
		id := m[1]
		target := "task:" + id
		if taskIDs[id] && target != src {
			edges = append(edges, GraphEdge{Source: src, Target: target, Type: "mention"})
		}
	}

	for _, m := range graphDocRefRE.FindAllStringSubmatch(content, -1) {
		path := m[1]
		target := "doc:" + path
		if docPaths[path] && target != src {
			edges = append(edges, GraphEdge{Source: src, Target: target, Type: "mention"})
		}
	}

	for _, m := range graphMemoryRefRE.FindAllStringSubmatch(content, -1) {
		id := m[1]
		target := "memory:" + id
		if memoryIDs[id] && target != src {
			edges = append(edges, GraphEdge{Source: src, Target: target, Type: "mention"})
		}
	}

	return edges
}

// deduplicateEdges removes duplicate edges (same source+target+type).
func deduplicateEdges(edges []GraphEdge) []GraphEdge {
	type key struct{ s, t, ty string }
	seen := make(map[key]bool, len(edges))
	out := make([]GraphEdge, 0, len(edges))
	for _, e := range edges {
		k := key{e.Source, e.Target, e.Type}
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	return out
}
