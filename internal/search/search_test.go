package search

import (
	"testing"

	"github.com/howznguyen/knowns/internal/models"
)

// --- Model Prefix Config Tests ---

func TestEmbeddingModelPrefixes(t *testing.T) {
	tests := []struct {
		model       string
		queryPrefix string
		docPrefix   string
	}{
		{"gte-small", "", ""},
		{"all-MiniLM-L6-v2", "", ""},
		{"gte-base", "", ""},
		{"bge-small-en-v1.5", "Represent this sentence: ", "Represent this sentence: "},
		{"bge-base-en-v1.5", "Represent this sentence: ", "Represent this sentence: "},
		{"nomic-embed-text-v1.5", "search_query: ", "search_document: "},
		{"multilingual-e5-small", "query: ", "passage: "},
	}

	for _, tt := range tests {
		cfg, ok := EmbeddingModels[tt.model]
		if !ok {
			t.Errorf("model %q not found in EmbeddingModels", tt.model)
			continue
		}
		if cfg.QueryPrefix != tt.queryPrefix {
			t.Errorf("model %q: QueryPrefix = %q, want %q", tt.model, cfg.QueryPrefix, tt.queryPrefix)
		}
		if cfg.DocPrefix != tt.docPrefix {
			t.Errorf("model %q: DocPrefix = %q, want %q", tt.model, cfg.DocPrefix, tt.docPrefix)
		}
	}
}

// --- ChunkVersion Tests ---

func TestChunkVersionConstant(t *testing.T) {
	if ChunkVersion < 2 {
		t.Errorf("ChunkVersion = %d, want >= 2", ChunkVersion)
	}
}

// --- Code Block Detection Tests ---

func TestExtractHeadings_CodeBlockIgnored(t *testing.T) {
	md := "## Setup\nSome text\n```bash\n# Install deps\nnpm install\n```\n## Usage\nMore text"

	headings := extractHeadings(md)

	titles := make([]string, len(headings))
	for i, h := range headings {
		titles[i] = h.Title
	}

	if len(headings) != 2 {
		t.Fatalf("expected 2 headings, got %d: %v", len(headings), titles)
	}
	if headings[0].Title != "Setup" {
		t.Errorf("heading[0].Title = %q, want %q", headings[0].Title, "Setup")
	}
	if headings[1].Title != "Usage" {
		t.Errorf("heading[1].Title = %q, want %q", headings[1].Title, "Usage")
	}
}

func TestExtractHeadings_NoCodeBlock(t *testing.T) {
	md := "## First\nContent\n## Second\nMore"

	headings := extractHeadings(md)
	if len(headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(headings))
	}
}

// --- Header Path Tests ---

func TestExtractHeadings_HeaderPath(t *testing.T) {
	md := "## API Reference\n### Endpoints\n#### GET /users\nReturns users.\n#### POST /users\nCreates user.\n### Authentication\n#### API Keys\nKey info."

	headings := extractHeadings(md)

	expected := []struct {
		title      string
		headerPath string
	}{
		{"API Reference", "API Reference"},
		{"Endpoints", "API Reference/Endpoints"},
		{"GET /users", "API Reference/Endpoints/GET /users"},
		{"POST /users", "API Reference/Endpoints/POST /users"},
		{"Authentication", "API Reference/Authentication"},
		{"API Keys", "API Reference/Authentication/API Keys"},
	}

	if len(headings) != len(expected) {
		t.Fatalf("expected %d headings, got %d", len(expected), len(headings))
	}

	for i, exp := range expected {
		if headings[i].Title != exp.title {
			t.Errorf("heading[%d].Title = %q, want %q", i, headings[i].Title, exp.title)
		}
		if headings[i].HeaderPath != exp.headerPath {
			t.Errorf("heading[%d].HeaderPath = %q, want %q", i, headings[i].HeaderPath, exp.headerPath)
		}
	}
}

func TestExtractHeadings_HeaderStackPop(t *testing.T) {
	// H2 -> H3 -> H2 should pop H3 and H2 before pushing new H2
	md := "## A\n### B\nContent\n## C\nContent"

	headings := extractHeadings(md)

	if len(headings) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(headings))
	}
	if headings[2].HeaderPath != "C" {
		t.Errorf("heading[2].HeaderPath = %q, want %q", headings[2].HeaderPath, "C")
	}
}

// --- H1 Content Loss Fix Tests ---

func TestChunkDocument_H1ContentCaptured(t *testing.T) {
	content := "# My API\nThis API handles user management.\n## Endpoints\nGET /users"

	result := ChunkDocument(content, "test", "My API", "Description", 512, nil)

	if len(result.Chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(result.Chunks))
	}

	metaChunk := result.Chunks[0]
	if metaChunk.Section != "# Metadata" {
		t.Errorf("chunk[0].Section = %q, want %q", metaChunk.Section, "# Metadata")
	}

	// Metadata chunk should contain H1 body content.
	if !contains(metaChunk.Content, "This API handles user management.") {
		t.Errorf("metadata chunk should contain H1 body content, got: %q", metaChunk.Content)
	}
}

func TestChunkDocument_H1NoBody(t *testing.T) {
	content := "# Title\n## Section\nContent"

	result := ChunkDocument(content, "test", "Title", "", 512, nil)

	// Metadata chunk should just have title, no extra content.
	metaChunk := result.Chunks[0]
	if contains(metaChunk.Content, "\n\n\n") {
		t.Errorf("metadata chunk should not have extra blank lines: %q", metaChunk.Content)
	}
}

func TestChunkDocument_NoHeadings(t *testing.T) {
	content := "Just plain text without any headings."

	result := ChunkDocument(content, "test", "Title", "Desc", 512, nil)

	// Should have metadata chunk + content chunk.
	if len(result.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result.Chunks))
	}
}

// --- Token Counting Tests ---

func TestCountTokens_NilTokenizer(t *testing.T) {
	// Should fall back to EstimateTokens.
	result := countTokens("hello world", nil)
	expected := EstimateTokens("hello world")
	if result != expected {
		t.Errorf("countTokens(nil) = %d, want %d", result, expected)
	}
}

// mockTokenizer implements Tokenizer for testing.
type mockTokenizer struct {
	tokenCount int
}

func (m *mockTokenizer) Encode(text string, maxLength int) TokenizerOutput {
	ids := make([]int64, m.tokenCount)
	mask := make([]int64, m.tokenCount)
	types := make([]int64, m.tokenCount)
	for i := range ids {
		ids[i] = int64(i + 1)
		mask[i] = 1
	}
	return TokenizerOutput{
		InputIDs:      ids,
		AttentionMask: mask,
		TokenTypeIDs:  types,
	}
}

func TestCountTokens_WithTokenizer(t *testing.T) {
	tok := &mockTokenizer{tokenCount: 42}
	result := countTokens("any text", tok)
	if result != 42 {
		t.Errorf("countTokens(mockTokenizer) = %d, want 42", result)
	}
}

func TestChunkDocument_WithTokenizer(t *testing.T) {
	content := "## Section\nSome content here."
	tok := &mockTokenizer{tokenCount: 10}

	result := ChunkDocument(content, "test", "Title", "Desc", 512, tok)

	// Should use tokenizer for counting — all chunks should have tokenCount=10.
	for i, c := range result.Chunks {
		if c.TokenCount != 10 {
			t.Errorf("chunk[%d].TokenCount = %d, want 10", i, c.TokenCount)
		}
	}
}

// --- Task Chunk Splitting Tests ---

func TestChunkTask_ShortFields(t *testing.T) {
	task := &models.Task{
		ID:          "t1",
		Title:       "Test task",
		Description: "Short description",
		Status:      "todo",
		Priority:    "medium",
	}

	result := ChunkTask(task, 512, nil)

	// Should produce 1 chunk (description only, since others are empty).
	if len(result.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
	}
	if result.Chunks[0].Field != "description" {
		t.Errorf("chunk[0].Field = %q, want %q", result.Chunks[0].Field, "description")
	}
}

func TestChunkTask_LongFieldSplit(t *testing.T) {
	// Create a task with a very long description that exceeds maxTokens.
	// With EstimateTokens (~4 chars/token), 100 tokens = ~400 chars.
	longPara1 := repeatStr("Word ", 100) // ~500 chars = ~125 tokens
	longPara2 := repeatStr("Text ", 100) // ~500 chars = ~125 tokens

	task := &models.Task{
		ID:          "t2",
		Title:       "Long task",
		Description: longPara1 + "\n\n" + longPara2,
		Status:      "todo",
		Priority:    "high",
	}

	result := ChunkTask(task, 150, nil) // maxTokens=150, each para ~125+title tokens

	// Should split into multiple chunks.
	if len(result.Chunks) < 2 {
		t.Errorf("expected at least 2 chunks for long task, got %d", len(result.Chunks))
	}

	// All chunks should be for the description field.
	for i, c := range result.Chunks {
		if c.Field != "description" {
			t.Errorf("chunk[%d].Field = %q, want %q", i, c.Field, "description")
		}
		if c.TaskID != "t2" {
			t.Errorf("chunk[%d].TaskID = %q, want %q", i, c.TaskID, "t2")
		}
	}
}

func TestChunkTask_PreservesMetadata(t *testing.T) {
	task := &models.Task{
		ID:       "t3",
		Title:    "Meta task",
		Status:   "in-progress",
		Priority: "high",
		Labels:   []string{"bug", "urgent"},
		Description: "Some description",
	}

	result := ChunkTask(task, 512, nil)

	for _, c := range result.Chunks {
		if c.Status != "in-progress" {
			t.Errorf("chunk.Status = %q, want %q", c.Status, "in-progress")
		}
		if c.Priority != "high" {
			t.Errorf("chunk.Priority = %q, want %q", c.Priority, "high")
		}
		if len(c.Labels) != 2 || c.Labels[0] != "bug" {
			t.Errorf("chunk.Labels = %v, want [bug urgent]", c.Labels)
		}
	}
}

// --- HeaderPath in ChunkDocument ---

func TestChunkDocument_HeaderPathInChunks(t *testing.T) {
	content := "## API\n### Users\nUser content.\n### Auth\nAuth content."

	result := ChunkDocument(content, "test", "Title", "", 512, nil)

	// Find chunks by section.
	var usersChunk, authChunk *Chunk
	for i := range result.Chunks {
		if result.Chunks[i].Section == "### Users" {
			usersChunk = &result.Chunks[i]
		}
		if result.Chunks[i].Section == "### Auth" {
			authChunk = &result.Chunks[i]
		}
	}

	if usersChunk == nil {
		t.Fatal("Users chunk not found")
	}
	if usersChunk.HeaderPath != "API/Users" {
		t.Errorf("Users chunk HeaderPath = %q, want %q", usersChunk.HeaderPath, "API/Users")
	}

	if authChunk == nil {
		t.Fatal("Auth chunk not found")
	}
	if authChunk.HeaderPath != "API/Auth" {
		t.Errorf("Auth chunk HeaderPath = %q, want %q", authChunk.HeaderPath, "API/Auth")
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// --- Content Hash Tests ---

func TestContentHash_Deterministic(t *testing.T) {
	h1 := contentHash("hello world")
	h2 := contentHash("hello world")
	if h1 != h2 {
		t.Errorf("same input should produce same hash: %q != %q", h1, h2)
	}
}

func TestContentHash_DifferentInputs(t *testing.T) {
	h1 := contentHash("hello")
	h2 := contentHash("world")
	if h1 == h2 {
		t.Errorf("different inputs should produce different hashes")
	}
}

func TestTaskContentForHash_IncludesAllFields(t *testing.T) {
	task := &models.Task{
		Title:       "Test",
		Description: "Desc",
		Status:      "todo",
		Priority:    "high",
		AcceptanceCriteria: []models.AcceptanceCriterion{
			{Text: "AC1"},
			{Text: "AC2"},
		},
		ImplementationPlan:  "Plan",
		ImplementationNotes: "Notes",
	}

	hash := taskContentForHash(task)
	for _, expected := range []string{"Test", "Desc", "todo", "high", "AC1", "AC2", "Plan", "Notes"} {
		if !contains(hash, expected) {
			t.Errorf("taskContentForHash missing %q", expected)
		}
	}
}

func TestTaskContentForHash_ChangesOnUpdate(t *testing.T) {
	task := &models.Task{
		Title:       "Test",
		Description: "Original",
		Status:      "todo",
	}
	h1 := contentHash(taskContentForHash(task))

	task.Description = "Updated"
	h2 := contentHash(taskContentForHash(task))

	if h1 == h2 {
		t.Errorf("hash should change when task content changes")
	}
}

// --- SQLite VectorStore Content Hash Tests ---

func TestSQLiteVectorStore_ContentHashes(t *testing.T) {
	dir := t.TempDir()
	store := NewSQLiteVectorStore(dir, "test-model", 384)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer store.Close()

	// Initially empty.
	if h := store.GetContentHash("task:abc"); h != "" {
		t.Errorf("expected empty hash, got %q", h)
	}

	// Set and get.
	store.SetContentHash("task:abc", "hash123")
	if h := store.GetContentHash("task:abc"); h != "hash123" {
		t.Errorf("expected %q, got %q", "hash123", h)
	}

	// Update.
	store.SetContentHash("task:abc", "hash456")
	if h := store.GetContentHash("task:abc"); h != "hash456" {
		t.Errorf("expected %q, got %q", "hash456", h)
	}

	// List.
	store.SetContentHash("doc:readme", "hashXYZ")
	hashes := store.ListContentHashes()
	if len(hashes) != 2 {
		t.Fatalf("expected 2 hashes, got %d", len(hashes))
	}
	if hashes["task:abc"] != "hash456" {
		t.Errorf("task:abc hash = %q, want %q", hashes["task:abc"], "hash456")
	}
	if hashes["doc:readme"] != "hashXYZ" {
		t.Errorf("doc:readme hash = %q, want %q", hashes["doc:readme"], "hashXYZ")
	}

	// Delete.
	store.DeleteContentHash("task:abc")
	if h := store.GetContentHash("task:abc"); h != "" {
		t.Errorf("expected empty after delete, got %q", h)
	}
	hashes = store.ListContentHashes()
	if len(hashes) != 1 {
		t.Errorf("expected 1 hash after delete, got %d", len(hashes))
	}
}

func TestSQLiteVectorStore_ClearRemovesHashes(t *testing.T) {
	dir := t.TempDir()
	store := NewSQLiteVectorStore(dir, "test-model", 384)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	defer store.Close()

	store.SetContentHash("task:abc", "hash123")
	store.SetContentHash("doc:readme", "hash456")

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	hashes := store.ListContentHashes()
	if len(hashes) != 0 {
		t.Errorf("expected 0 hashes after Clear, got %d", len(hashes))
	}
}

// --- FileVectorStore Content Hash No-ops ---

func TestFileVectorStore_ContentHashNoOps(t *testing.T) {
	dir := t.TempDir()
	store := NewFileVectorStore(dir, "test-model", 384)

	// All should be no-ops / return empty.
	if h := store.GetContentHash("task:abc"); h != "" {
		t.Errorf("expected empty, got %q", h)
	}
	store.SetContentHash("task:abc", "hash123") // no-op
	store.DeleteContentHash("task:abc")          // no-op
	if hashes := store.ListContentHashes(); hashes != nil {
		t.Errorf("expected nil, got %v", hashes)
	}
}
