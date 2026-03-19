package search

import (
	"fmt"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
)

// IndexService orchestrates chunking, embedding, and storage of vectors.
type IndexService struct {
	store    *storage.Store
	embedder *Embedder
	vecStore VectorStore
}

// NewIndexService creates an IndexService.
func NewIndexService(store *storage.Store, embedder *Embedder, vecStore VectorStore) *IndexService {
	return &IndexService{
		store:    store,
		embedder: embedder,
		vecStore: vecStore,
	}
}

// ReindexProgress is called during reindexing to report progress.
type ReindexProgress func(phase string, current, total int)

// Reindex clears the vector store and re-indexes all tasks and docs.
func (s *IndexService) Reindex(progress ReindexProgress) error {
	if err := s.vecStore.Clear(); err != nil {
		return fmt.Errorf("clear vecstore: %w", err)
	}

	tasks, err := s.store.Tasks.List()
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}
	docs, err := s.store.Docs.List()
	if err != nil {
		return fmt.Errorf("list docs: %w", err)
	}

	total := len(tasks) + len(docs)
	current := 0

	// Index tasks.
	for _, task := range tasks {
		current++
		if progress != nil {
			progress("tasks", current, total)
		}
		if err := s.embedAndStoreTask(task); err != nil {
			continue // non-fatal
		}
	}

	// Index docs.
	for _, doc := range docs {
		current++
		if progress != nil {
			progress("docs", current, total)
		}
		fullDoc, err := s.store.Docs.Get(doc.Path)
		if err != nil {
			continue
		}
		if err := s.embedAndStoreDoc(fullDoc); err != nil {
			continue
		}
	}

	return s.vecStore.Save()
}

// IndexTask incrementally indexes a single task (removes old chunks first).
func (s *IndexService) IndexTask(taskID string) error {
	s.vecStore.RemoveByPrefix(fmt.Sprintf("task:%s:", taskID))

	task, err := s.store.Tasks.Get(taskID)
	if err != nil {
		return err
	}
	if err := s.embedAndStoreTask(task); err != nil {
		return err
	}
	return s.vecStore.Save()
}

// IndexDoc incrementally indexes a single doc (removes old chunks first).
func (s *IndexService) IndexDoc(docPath string) error {
	s.vecStore.RemoveByPrefix(fmt.Sprintf("doc:%s:", docPath))

	doc, err := s.store.Docs.Get(docPath)
	if err != nil {
		return err
	}
	if err := s.embedAndStoreDoc(doc); err != nil {
		return err
	}
	return s.vecStore.Save()
}

// RemoveTask removes all chunks for a task from the vector store.
func (s *IndexService) RemoveTask(taskID string) error {
	s.vecStore.RemoveByPrefix(fmt.Sprintf("task:%s:", taskID))
	return s.vecStore.Save()
}

// RemoveDoc removes all chunks for a doc from the vector store.
func (s *IndexService) RemoveDoc(docPath string) error {
	s.vecStore.RemoveByPrefix(fmt.Sprintf("doc:%s:", docPath))
	return s.vecStore.Save()
}

func (s *IndexService) embedAndStoreTask(task *models.Task) error {
	result := ChunkTask(task)
	return s.embedAndStore(result.Chunks)
}

func (s *IndexService) embedAndStoreDoc(doc *models.Doc) error {
	maxTokens := 512
	if cfg, ok := EmbeddingModels[s.vecStore.Model()]; ok {
		maxTokens = cfg.MaxTokens
	}
	result := ChunkDocument(doc.Content, doc.Path, doc.Title, doc.Description, maxTokens)
	return s.embedAndStore(result.Chunks)
}

func (s *IndexService) embedAndStore(chunks []Chunk) error {
	for i := range chunks {
		vec, err := s.embedder.Embed(chunks[i].Content)
		if err != nil {
			return fmt.Errorf("embed chunk %s: %w", chunks[i].ID, err)
		}
		chunks[i].Embedding = vec
	}
	s.vecStore.AddChunks(chunks)
	return nil
}
