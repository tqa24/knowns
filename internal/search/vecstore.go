package search

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// VectorStore is the interface for storing and searching embedding vectors.
type VectorStore interface {
	Load() error
	Save() error
	Clear() error
	AddChunks(chunks []Chunk)
	RemoveByPrefix(prefix string)
	Search(queryVec []float32, opts VectorSearchOpts) []ScoredChunk
	Count() int
	NeedsRebuild(model string) bool
	Stats() (chunkCount int, model string, indexedAt time.Time)
	Close() error
	Model() string
}

// FileVectorStore stores embeddings in a flat binary file with a JSON index.
// Deprecated: Use SQLiteVectorStore instead.
//
// Layout in .knowns/.search/:
//   - embeddings.bin  – contiguous float32 vectors
//   - index.json      – chunk metadata with byte offsets into the bin file
//   - version.json    – model info, dimension count, timestamp
type FileVectorStore struct {
	dir        string
	dimensions int
	model      string

	mu     sync.RWMutex
	index  []indexEntry
	vecs   []float32 // all vectors loaded into memory (flat)
	loaded bool
}

type indexEntry struct {
	ID         string    `json:"id"`
	Type       ChunkType `json:"type"`
	Offset     int       `json:"offset"` // index into vecs (in float32 units, not bytes)
	TokenCount int       `json:"tokenCount,omitempty"`

	// Doc fields.
	DocPath       string `json:"docPath,omitempty"`
	Section       string `json:"section,omitempty"`
	HeadingLevel  int    `json:"headingLevel,omitempty"`
	ParentSection string `json:"parentSection,omitempty"`
	Position      int    `json:"position,omitempty"`

	// Task fields.
	TaskID   string   `json:"taskId,omitempty"`
	Field    string   `json:"field,omitempty"`
	Status   string   `json:"status,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

type versionInfo struct {
	Model      string    `json:"model"`
	Dimensions int       `json:"dimensions"`
	IndexedAt  time.Time `json:"indexedAt"`
	ChunkCount int       `json:"chunkCount"`
}

// NewFileVectorStore creates (or opens) a vector store in dir.
func NewFileVectorStore(dir string, model string, dimensions int) *FileVectorStore {
	return &FileVectorStore{
		dir:        dir,
		model:      model,
		dimensions: dimensions,
	}
}

func (s *FileVectorStore) embeddingsPath() string { return filepath.Join(s.dir, "embeddings.bin") }
func (s *FileVectorStore) indexPath() string       { return filepath.Join(s.dir, "index.json") }
func (s *FileVectorStore) versionPath() string     { return filepath.Join(s.dir, "version.json") }

// Load reads the store from disk into memory. Safe to call multiple times.
func (s *FileVectorStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("vecstore: mkdir: %w", err)
	}

	// Read index.
	indexData, err := os.ReadFile(s.indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			s.index = nil
			s.vecs = nil
			s.loaded = true
			return nil
		}
		return fmt.Errorf("vecstore: read index: %w", err)
	}

	var entries []indexEntry
	if err := json.Unmarshal(indexData, &entries); err != nil {
		return fmt.Errorf("vecstore: parse index: %w", err)
	}

	// Read embeddings binary.
	binData, err := os.ReadFile(s.embeddingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			s.index = entries
			s.vecs = nil
			s.loaded = true
			return nil
		}
		return fmt.Errorf("vecstore: read embeddings: %w", err)
	}

	// Convert bytes to float32 slice.
	floatCount := len(binData) / 4
	vecs := make([]float32, floatCount)
	for i := 0; i < floatCount; i++ {
		bits := binary.LittleEndian.Uint32(binData[i*4 : (i+1)*4])
		vecs[i] = math.Float32frombits(bits)
	}

	s.index = entries
	s.vecs = vecs
	s.loaded = true
	return nil
}

// Save writes the current in-memory state to disk.
func (s *FileVectorStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	// Write index.
	indexData, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return fmt.Errorf("vecstore: marshal index: %w", err)
	}
	if err := os.WriteFile(s.indexPath(), indexData, 0644); err != nil {
		return fmt.Errorf("vecstore: write index: %w", err)
	}

	// Write embeddings binary.
	binData := make([]byte, len(s.vecs)*4)
	for i, v := range s.vecs {
		binary.LittleEndian.PutUint32(binData[i*4:], math.Float32bits(v))
	}
	if err := os.WriteFile(s.embeddingsPath(), binData, 0644); err != nil {
		return fmt.Errorf("vecstore: write embeddings: %w", err)
	}

	// Write version info.
	ver := versionInfo{
		Model:      s.model,
		Dimensions: s.dimensions,
		IndexedAt:  time.Now(),
		ChunkCount: len(s.index),
	}
	verData, _ := json.MarshalIndent(ver, "", "  ")
	if err := os.WriteFile(s.versionPath(), verData, 0644); err != nil {
		return fmt.Errorf("vecstore: write version: %w", err)
	}

	return nil
}

// Clear removes all data from the store (in-memory and on disk).
func (s *FileVectorStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.index = nil
	s.vecs = nil

	os.Remove(s.embeddingsPath())
	os.Remove(s.indexPath())
	os.Remove(s.versionPath())
	return nil
}

// AddChunks appends embedded chunks to the store. Each chunk must have a
// non-nil Embedding of length == s.dimensions.
func (s *FileVectorStore) AddChunks(chunks []Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range chunks {
		if len(c.Embedding) != s.dimensions {
			continue
		}

		offset := len(s.vecs)
		s.vecs = append(s.vecs, c.Embedding...)

		s.index = append(s.index, indexEntry{
			ID:            c.ID,
			Type:          c.Type,
			Offset:        offset,
			TokenCount:    c.TokenCount,
			DocPath:       c.DocPath,
			Section:       c.Section,
			HeadingLevel:  c.HeadingLevel,
			ParentSection: c.ParentSection,
			Position:      c.Position,
			TaskID:        c.TaskID,
			Field:         c.Field,
			Status:        c.Status,
			Priority:      c.Priority,
			Labels:        c.Labels,
		})
	}
}

// RemoveByPrefix removes all chunks whose ID starts with prefix. Rebuilds
// vectors and index in-place.
func (s *FileVectorStore) RemoveByPrefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newIndex []indexEntry
	var newVecs []float32

	for _, entry := range s.index {
		if len(entry.ID) >= len(prefix) && entry.ID[:len(prefix)] == prefix {
			continue // skip this entry
		}
		// Copy the embedding.
		start := entry.Offset
		end := start + s.dimensions
		if end > len(s.vecs) {
			continue // corrupted entry
		}

		newOffset := len(newVecs)
		newVecs = append(newVecs, s.vecs[start:end]...)
		entry.Offset = newOffset
		newIndex = append(newIndex, entry)
	}

	s.index = newIndex
	s.vecs = newVecs
}

// Search performs brute-force cosine similarity search, returning up to topK
// results above the threshold.
func (s *FileVectorStore) Search(queryVec []float32, opts VectorSearchOpts) []ScoredChunk {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(queryVec) != s.dimensions || len(s.index) == 0 {
		return nil
	}

	if opts.TopK <= 0 {
		opts.TopK = 20
	}
	if opts.Threshold <= 0 {
		opts.Threshold = 0.3
	}

	type scored struct {
		idx   int
		score float64
	}
	var candidates []scored

	for i, entry := range s.index {
		start := entry.Offset
		end := start + s.dimensions
		if end > len(s.vecs) {
			continue
		}
		vec := s.vecs[start:end]
		sim := CosineSimilarity(queryVec, vec)
		if sim >= opts.Threshold {
			candidates = append(candidates, scored{i, sim})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if len(candidates) > opts.TopK {
		candidates = candidates[:opts.TopK]
	}

	results := make([]ScoredChunk, len(candidates))
	for i, c := range candidates {
		entry := s.index[c.idx]
		results[i] = ScoredChunk{
			Chunk: Chunk{
				ID:            entry.ID,
				Type:          entry.Type,
				TokenCount:    entry.TokenCount,
				DocPath:       entry.DocPath,
				Section:       entry.Section,
				HeadingLevel:  entry.HeadingLevel,
				ParentSection: entry.ParentSection,
				Position:      entry.Position,
				TaskID:        entry.TaskID,
				Field:         entry.Field,
				Status:        entry.Status,
				Priority:      entry.Priority,
				Labels:        entry.Labels,
			},
			Score: c.score,
		}
	}
	return results
}

// Count returns the number of indexed chunks.
func (s *FileVectorStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index)
}

// NeedsRebuild returns true if the stored model differs from the current one.
func (s *FileVectorStore) NeedsRebuild(model string) bool {
	data, err := os.ReadFile(s.versionPath())
	if err != nil {
		return true
	}
	var ver versionInfo
	if err := json.Unmarshal(data, &ver); err != nil {
		return true
	}
	return ver.Model != model
}

// Stats returns basic statistics about the vector store.
func (s *FileVectorStore) Stats() (chunkCount int, model string, indexedAt time.Time) {
	data, err := os.ReadFile(s.versionPath())
	if err != nil {
		return 0, "", time.Time{}
	}
	var ver versionInfo
	if err := json.Unmarshal(data, &ver); err != nil {
		return 0, "", time.Time{}
	}
	return ver.ChunkCount, ver.Model, ver.IndexedAt
}

// Close is a no-op for FileVectorStore (no persistent connection).
func (s *FileVectorStore) Close() error { return nil }

// Model returns the embedding model name.
func (s *FileVectorStore) Model() string { return s.model }

// Compile-time check that FileVectorStore implements VectorStore.
var _ VectorStore = (*FileVectorStore)(nil)
