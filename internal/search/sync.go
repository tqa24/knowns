package search

import (
	"errors"
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
