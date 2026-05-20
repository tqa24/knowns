package search

import (
	"errors"
	"fmt"

	"github.com/howznguyen/knowns/internal/runtimequeue"
	"github.com/howznguyen/knowns/internal/storage"
)

// ExecuteRuntimeJob runs a queued runtime job synchronously inside the shared runtime.
func ExecuteRuntimeJob(storeRoot string, job runtimequeue.Job) error {
	store := storage.NewStore(storeRoot)
	switch job.Kind {
	case runtimequeue.JobIndexTask:
		return executeRuntimeEntity(store, string(job.Kind)+" "+job.Target, func(svc *IndexService) error {
			return svc.IndexTask(job.Target)
		})
	case runtimequeue.JobIndexDoc:
		return executeRuntimeEntity(store, string(job.Kind)+" "+job.Target, func(svc *IndexService) error {
			return svc.IndexDoc(job.Target)
		})
	case runtimequeue.JobRemoveTask:
		return executeRuntimeEntity(store, string(job.Kind)+" "+job.Target, func(svc *IndexService) error {
			return svc.RemoveTask(job.Target)
		})
	case runtimequeue.JobRemoveDoc:
		return executeRuntimeEntity(store, string(job.Kind)+" "+job.Target, func(svc *IndexService) error {
			return svc.RemoveDoc(job.Target)
		})
	case runtimequeue.JobIndexMemory:
		return executeRuntimeEntity(store, string(job.Kind)+" "+job.Target, func(svc *IndexService) error {
			return svc.IndexMemory(job.Target)
		})
	case runtimequeue.JobRemoveMemory:
		return executeRuntimeEntity(store, string(job.Kind)+" "+job.Target, func(svc *IndexService) error {
			return svc.RemoveMemory(job.Target)
		})
	case runtimequeue.JobReindex:
		embedder, vecStore, err := InitSemantic(store)
		if err != nil {
			if errors.Is(err, ErrSemanticNotConfigured) {
				return nil
			}
			return err
		}
		if embedder == nil || vecStore == nil {
			return nil
		}
		defer embedder.Close()
		defer vecStore.Close()
		return NewEngine(store, embedder, vecStore).Reindex(func(phase string, current, total int) {
			_ = runtimequeue.ReportProgress(storeRoot, job.ID, phase, current, total)
		})
	case runtimequeue.JobIndexFile, runtimequeue.JobRemoveFile, runtimequeue.JobIndexAll:
		return nil
	default:
		return fmt.Errorf("unsupported runtime job kind: %s", job.Kind)
	}
}

func executeRuntimeEntity(store *storage.Store, action string, fn func(*IndexService) error) error {
	if store == nil {
		return nil
	}
	embedder, vecStore, err := InitSemantic(store)
	if err != nil {
		if errors.Is(err, ErrSemanticNotConfigured) {
			return nil
		}
		return fmt.Errorf("could not %s: %w", action, err)
	}
	defer embedder.Close()
	defer vecStore.Close()
	return fn(NewIndexService(store, embedder, vecStore))
}
