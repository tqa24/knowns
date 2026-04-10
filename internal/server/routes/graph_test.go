package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"

	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

func TestCodeGraph_ReturnsCodeNodesWithoutSemanticEdges(t *testing.T) {
	store := newGraphRouteTestStore(t)
	seedCodeGraphTestData(t, store)

	r := chi.NewRouter()
	(&GraphRoutes{store: store}).Register(r)

	req := httptest.NewRequest(http.MethodGet, "/graph/code", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /graph/code status = %d, want 200", w.Code)
	}

	var resp struct {
		Nodes []GraphNode `json:"nodes"`
		Edges []GraphEdge `json:"edges"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Nodes) != 6 {
		t.Fatalf("node count = %d, want 6", len(resp.Nodes))
	}
	if len(resp.Edges) != 9 {
		t.Fatalf("edge count = %d, want 9", len(resp.Edges))
	}

	kindByID := map[string]string{}
	for _, node := range resp.Nodes {
		kind, _ := node.Data["kind"].(string)
		kindByID[node.ID] = kind
	}

	assertKind := func(id, want string) {
		t.Helper()
		if got := kindByID[id]; got != want {
			t.Fatalf("node kind for %s = %q, want %q", id, got, want)
		}
	}

	assertKind("code::sample.go::__file__", "file")
	assertKind("code::sample.go::Hello", "function")
	assertKind("code::sample.go::Save", "method")
	assertKind("code::sample.go::User", "class")
	assertKind("code::sample.go::BaseUser", "class")
	assertKind("code::sample.go::Store", "interface")

	hasCall := false
	hasImplements := false
	hasMethod := false
	hasExtends := false
	for _, edge := range resp.Edges {
		if edge.Type == "calls" && edge.Source == "code::sample.go::Hello" && edge.Target == "code::sample.go::Save" {
			hasCall = true
		}
		if edge.Type == "has_method" && edge.Source == "code::sample.go::User" && edge.Target == "code::sample.go::Save" {
			hasMethod = true
		}
		if edge.Type == "extends" && edge.Source == "code::sample.go::User" && edge.Target == "code::sample.go::BaseUser" {
			hasExtends = true
		}
		if edge.Type == "implements" && edge.Source == "code::sample.go::User" && edge.Target == "code::sample.go::Store" {
			hasImplements = true
		}
	}
	if !hasCall {
		t.Fatal("expected calls edge in graph payload")
	}
	if !hasImplements {
		t.Fatal("expected implements edge in graph payload")
	}
	if !hasMethod {
		t.Fatal("expected has_method edge in graph payload")
	}
	if !hasExtends {
		t.Fatal("expected extends edge in graph payload")
	}
}

func newGraphRouteTestStore(t *testing.T) *storage.Store {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".knowns")
	store := storage.NewStore(root)
	if err := store.Init("graph-route-test"); err != nil {
		t.Fatalf("Init store: %v", err)
	}
	return store
}

func seedCodeGraphTestData(t *testing.T, store *storage.Store) {
	t.Helper()
	vecStore := search.NewSQLiteVectorStore(filepath.Join(store.Root, ".search"), "test", 1)
	if err := vecStore.Load(); err != nil {
		t.Fatalf("load vec store: %v", err)
	}
	defer vecStore.Close()

	chunks := []search.Chunk{
		{ID: "code::sample.go::__file__", Type: search.ChunkTypeCode, Content: "file sample.go", DocPath: "sample.go", Field: "file", Embedding: []float32{1}},
		{ID: "code::sample.go::Hello", Type: search.ChunkTypeCode, Content: "function Hello", DocPath: "sample.go", Field: "function", Name: "Hello", Embedding: []float32{1}},
		{ID: "code::sample.go::Save", Type: search.ChunkTypeCode, Content: "method Save", DocPath: "sample.go", Field: "method", Name: "Save", Embedding: []float32{1}},
		{ID: "code::sample.go::User", Type: search.ChunkTypeCode, Content: "class User", DocPath: "sample.go", Field: "class", Name: "User", Embedding: []float32{1}},
		{ID: "code::sample.go::BaseUser", Type: search.ChunkTypeCode, Content: "class BaseUser", DocPath: "sample.go", Field: "class", Name: "BaseUser", Embedding: []float32{1}},
		{ID: "code::sample.go::Store", Type: search.ChunkTypeCode, Content: "interface Store", DocPath: "sample.go", Field: "interface", Name: "Store", Embedding: []float32{1}},
	}
	vecStore.AddChunks(chunks)
	if err := vecStore.Save(); err != nil {
		t.Fatalf("save vec store: %v", err)
	}

	dbPath := filepath.Join(store.Root, ".search", "index.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open semantic db: %v", err)
	}
	defer db.Close()
	if err := search.SaveCodeEdges(db, []search.CodeEdge{
		{From: "code::sample.go::__file__", To: "code::sample.go::Hello", Type: "contains", FromPath: "sample.go", ToPath: "sample.go"},
		{From: "code::sample.go::__file__", To: "code::sample.go::Save", Type: "contains", FromPath: "sample.go", ToPath: "sample.go"},
		{From: "code::sample.go::__file__", To: "code::sample.go::User", Type: "contains", FromPath: "sample.go", ToPath: "sample.go"},
		{From: "code::sample.go::__file__", To: "code::sample.go::BaseUser", Type: "contains", FromPath: "sample.go", ToPath: "sample.go"},
		{From: "code::sample.go::__file__", To: "code::sample.go::Store", Type: "contains", FromPath: "sample.go", ToPath: "sample.go"},
		{From: "code::sample.go::Hello", To: "code::sample.go::Save", Type: "calls", FromPath: "sample.go", ToPath: "sample.go", ResolutionStatus: "resolved_internal", ResolutionConfidence: "high", ResolvedTo: "code::sample.go::Save"},
		{From: "code::sample.go::User", To: "code::sample.go::Save", Type: "has_method", FromPath: "sample.go", ToPath: "sample.go", ResolutionStatus: "resolved_internal", ResolutionConfidence: "high", ResolvedTo: "code::sample.go::Save"},
		{From: "code::sample.go::User", To: "code::sample.go::BaseUser", Type: "extends", FromPath: "sample.go", ToPath: "sample.go", ResolutionStatus: "resolved_internal", ResolutionConfidence: "high", ResolvedTo: "code::sample.go::BaseUser"},
		{From: "code::sample.go::User", To: "code::sample.go::Store", Type: "implements", FromPath: "sample.go", ToPath: "sample.go", ResolutionStatus: "resolved_internal", ResolutionConfidence: "medium", ResolvedTo: "code::sample.go::Store"},
	}); err != nil {
		t.Fatalf("save code edges: %v", err)
	}
}
