package search

import (
	"errors"
	"fmt"
	"log"

	"github.com/howznguyen/knowns/internal/storage"
)

func BestEffortIndexTask(store *storage.Store, taskID string) {
	runBestEffort(store, "index task "+taskID, func(svc *IndexService) error {
		return svc.IndexTask(taskID)
	})
}

func BestEffortIndexDoc(store *storage.Store, docPath string) {
	runBestEffort(store, "index doc "+docPath, func(svc *IndexService) error {
		return svc.IndexDoc(docPath)
	})
}

func BestEffortRemoveTask(store *storage.Store, taskID string) {
	runBestEffort(store, "remove task "+taskID, func(svc *IndexService) error {
		return svc.RemoveTask(taskID)
	})
}

func BestEffortRemoveDoc(store *storage.Store, docPath string) {
	runBestEffort(store, "remove doc "+docPath, func(svc *IndexService) error {
		return svc.RemoveDoc(docPath)
	})
}

func BestEffortIndexMemory(store *storage.Store, memoryID string) {
	runBestEffort(store, "index memory "+memoryID, func(svc *IndexService) error {
		return svc.IndexMemory(memoryID)
	})
}

func BestEffortRemoveMemory(store *storage.Store, memoryID string) {
	runBestEffort(store, "remove memory "+memoryID, func(svc *IndexService) error {
		return svc.RemoveMemory(memoryID)
	})
}

// BestEffortIndexAll runs a full code index of all files in projectRoot.
// Safe to call from a goroutine; errors are logged, not returned.
// No-op if semantic search is not configured.
func BestEffortIndexAll(store *storage.Store, projectRoot string) {
	runBestEffortCode(store, "index all files", func(embedder *Embedder, vecStore VectorStore) error {
		syms, edges, err := IndexAllFiles(projectRoot, false)
		if err != nil {
			return err
		}
		if len(syms) == 0 {
			return nil
		}

		vecStore.RemoveByPrefix("code::")

		var chunks []Chunk
		for _, sym := range syms {
			chunk := sym.ToChunk()
			vec, embedErr := embedder.EmbedDocument(chunk.Content)
			if embedErr != nil {
				continue
			}
			chunk.Embedding = vec
			chunks = append(chunks, chunk)
		}
		vecStore.AddChunks(chunks)
		if err := vecStore.Save(); err != nil {
			return err
		}

		if len(edges) == 0 {
			return nil
		}
		db := store.SemanticDB()
		if db == nil {
			return nil
		}
		defer db.Close()

		resolvedEdges := ResolveCodeEdges(syms, edges)
		_ = SaveCodeEdges(db, resolvedEdges)
		return nil
	})
}

// BestEffortIndexFile parses and indexes a single code file.
func BestEffortIndexFile(store *storage.Store, docPath, absPath string) {
	runBestEffortCode(store, "index file "+docPath, func(embedder *Embedder, vecStore VectorStore) error {
		syms, _, err := IndexFile(docPath, absPath)
		if err != nil || len(syms) == 0 {
			return err
		}
		// Remove old chunks for this file
		prefix := fmt.Sprintf("code::%s::", docPath)
		vecStore.RemoveByPrefix(prefix)
		// Convert and embed
		var chunks []Chunk
		for _, sym := range syms {
			chunk := sym.ToChunk()
			vec, err := embedder.EmbedDocument(chunk.Content)
			if err != nil {
				continue
			}
			chunk.Embedding = vec
			chunks = append(chunks, chunk)
		}
		vecStore.AddChunks(chunks)
		return vecStore.Save()
	})
}

// BestEffortRemoveFile removes all code chunks for a file from the vector store.
func BestEffortRemoveFile(store *storage.Store, docPath string) {
	runBestEffortCode(store, "remove file "+docPath, func(embedder *Embedder, vecStore VectorStore) error {
		prefix := fmt.Sprintf("code::%s::", docPath)
		vecStore.RemoveByPrefix(prefix)
		return vecStore.Save()
	})
}

func runBestEffortCode(store *storage.Store, action string, fn func(*Embedder, VectorStore) error) {
	if store == nil {
		return
	}
	embedder, vecStore, err := InitSemantic(store)
	if err != nil {
		if !errors.Is(err, ErrSemanticNotConfigured) {
			log.Printf("[search] could not %s: %v", action, err)
		}
		return
	}
	defer embedder.Close()
	defer vecStore.Close()
	if err := fn(embedder, vecStore); err != nil {
		log.Printf("[search] could not %s: %v", action, err)
	}
}

func runBestEffort(store *storage.Store, action string, fn func(*IndexService) error) {
	if store == nil {
		return
	}

	embedder, vecStore, err := InitSemantic(store)
	if err != nil {
		if !errors.Is(err, ErrSemanticNotConfigured) {
			log.Printf("[search] could not %s: %v", action, err)
		}
		return
	}
	defer embedder.Close()
	defer vecStore.Close()

	if err := fn(NewIndexService(store, embedder, vecStore)); err != nil {
		log.Printf("[search] could not %s: %v", action, err)
	}
}
