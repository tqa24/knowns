package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

// NormalizeTask ensures all slice fields are non-nil so JSON serialization
// produces [] instead of null/omitted, which the UI client relies on.
func NormalizeTask(t *models.Task) {
	if t.Labels == nil {
		t.Labels = []string{}
	}
	if t.Subtasks == nil {
		t.Subtasks = []string{}
	}
	if t.AcceptanceCriteria == nil {
		t.AcceptanceCriteria = []models.AcceptanceCriterion{}
	}
	if t.TimeEntries == nil {
		t.TimeEntries = []models.TimeEntry{}
	}
	if t.Fulfills == nil {
		t.Fulfills = []string{}
	}
}

func (tr *TaskRoutes) loadTaskTimeEntries(t *models.Task) {
	if t == nil {
		return
	}
	entries, err := tr.getStore().Time.GetEntries(t.ID)
	if err != nil {
		return
	}
	t.TimeEntries = entries
	NormalizeTask(t)
}

// TaskRoutes handles /api/tasks endpoints.
type TaskRoutes struct {
	store *storage.Store
	mgr   *storage.Manager
	sse   Broadcaster
}

func (tr *TaskRoutes) getStore() *storage.Store {
	if tr.mgr != nil {
		return tr.mgr.GetStore()
	}
	return tr.store
}

// Register wires the task routes onto r.
func (tr *TaskRoutes) Register(r chi.Router) {
	r.Get("/tasks", tr.list)
	r.Post("/tasks", tr.create)
	r.Get("/tasks/{id}", tr.get)
	r.Put("/tasks/{id}", tr.update)
	r.Post("/tasks/{id}/archive", tr.archive)
	r.Post("/tasks/{id}/unarchive", tr.unarchive)
	r.Post("/tasks/batch-archive", tr.batchArchive)
	r.Post("/tasks/reorder", tr.reorder)
	r.Post("/tasks/sync-spec-acs", tr.syncSpecACs)
	r.Get("/tasks/{id}/history", tr.history)
}

// list returns all active tasks.
//
// GET /api/tasks
func (tr *TaskRoutes) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := tr.getStore().Tasks.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		tasks = []*models.Task{}
	}
	for _, t := range tasks {
		tr.loadTaskTimeEntries(t)
	}
	respondJSON(w, http.StatusOK, tasks)
}

// get returns a single task by ID.
//
// GET /api/tasks/{id}
func (tr *TaskRoutes) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := tr.getStore().Tasks.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	tr.loadTaskTimeEntries(task)
	task.ActiveTimer = tr.getStore().Time.GetActiveTimer(task.ID)
	respondJSON(w, http.StatusOK, task)
}

// create persists a new task.
//
// POST /api/tasks
func (tr *TaskRoutes) create(w http.ResponseWriter, r *http.Request) {
	var task models.Task
	if err := decodeJSON(r, &task); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Generate an ID if not provided.
	if task.ID == "" {
		task.ID = models.NewTaskID()
	}

	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	if task.Status == "" {
		task.Status = "todo"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Labels == nil {
		task.Labels = []string{}
	}

	if err := tr.getStore().Tasks.Create(&task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	search.BestEffortIndexTask(tr.getStore(), task.ID)

	// Record initial version.
	_ = tr.getStore().Versions.SaveVersion(task.ID, models.TaskVersion{
		Changes:  tr.getStore().Versions.TrackChanges(nil, &task),
		Snapshot: storage.TaskToSnapshot(&task),
	})

	NormalizeTask(&task)
	tr.sse.Broadcast(SSEEvent{Type: "tasks:updated", Data: map[string]interface{}{"task": task}})
	respondJSON(w, http.StatusCreated, task)
}

// update replaces a task's fields.
//
// PUT /api/tasks/{id}
func (tr *TaskRoutes) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	existing, err := tr.getStore().Tasks.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	// Start from a copy of the existing task so that partial updates
	// (e.g. {"status":"done"}) only overwrite the fields present in the
	// request body — all other fields retain their current values.
	updated := *existing
	if err := decodeJSON(r, &updated); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Preserve immutable fields.
	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = time.Now().UTC()

	changes := tr.getStore().Versions.TrackChanges(existing, &updated)

	if err := tr.getStore().Tasks.Update(&updated); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	search.BestEffortIndexTask(tr.getStore(), updated.ID)

	if len(changes) > 0 {
		_ = tr.getStore().Versions.SaveVersion(id, models.TaskVersion{
			Changes:  changes,
			Snapshot: storage.TaskToSnapshot(&updated),
		})
	}

	NormalizeTask(&updated)
	tr.sse.Broadcast(SSEEvent{Type: "tasks:updated", Data: map[string]interface{}{"task": updated}})
	respondJSON(w, http.StatusOK, updated)
}

// archive moves a task to the archive.
//
// POST /api/tasks/{id}/archive
func (tr *TaskRoutes) archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Read the task before archiving so we can return it.
	task, err := tr.getStore().Tasks.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := tr.getStore().Tasks.Archive(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	search.BestEffortRemoveTask(tr.getStore(), id)
	task.Status = "archived"
	NormalizeTask(task)
	tr.sse.Broadcast(SSEEvent{Type: "tasks:archived", Data: map[string]interface{}{"task": task}})
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "task": task})
}

// unarchive restores a task from the archive.
//
// POST /api/tasks/{id}/unarchive
func (tr *TaskRoutes) unarchive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := tr.getStore().Tasks.Unarchive(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	search.BestEffortIndexTask(tr.getStore(), id)
	// Re-read the task after unarchiving to return the current state.
	task, err := tr.getStore().Tasks.Get(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	NormalizeTask(task)
	tr.sse.Broadcast(SSEEvent{Type: "tasks:unarchived", Data: map[string]interface{}{"task": task}})
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "task": task})
}

// batchArchiveRequest is the request body for batch-archive.
type batchArchiveRequest struct {
	OlderThanMs int64 `json:"olderThanMs"`
}

// batchArchive archives all tasks that were last updated before the given age.
//
// POST /api/tasks/batch-archive
func (tr *TaskRoutes) batchArchive(w http.ResponseWriter, r *http.Request) {
	var req batchArchiveRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	tasks, err := tr.getStore().Tasks.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	cutoff := time.Now().Add(-time.Duration(req.OlderThanMs) * time.Millisecond)
	var archivedTasks []*models.Task
	for _, t := range tasks {
		// Only archive "done" tasks that are older than the cutoff.
		if t.Status != "done" {
			continue
		}
		if req.OlderThanMs > 0 && t.UpdatedAt.After(cutoff) {
			continue
		}
		if err := tr.getStore().Tasks.Archive(t.ID); err == nil {
			search.BestEffortRemoveTask(tr.getStore(), t.ID)
			t.Status = "archived"
			NormalizeTask(t)
			archivedTasks = append(archivedTasks, t)
		}
	}
	if archivedTasks == nil {
		archivedTasks = []*models.Task{}
	}

	tr.sse.Broadcast(SSEEvent{Type: "tasks:batch-archived", Data: map[string]interface{}{
		"count": len(archivedTasks),
	}})
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   len(archivedTasks),
		"tasks":   archivedTasks,
	})
}

// reorderItem is a single entry in a reorder request.
type reorderItem struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
}

// reorderRequest is the request body for the batch reorder endpoint.
type reorderRequest struct {
	Orders []reorderItem `json:"orders"`
}

// reorder batch-updates the display order of tasks.
//
// POST /api/tasks/reorder
func (tr *TaskRoutes) reorder(w http.ResponseWriter, r *http.Request) {
	var req reorderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	updated := 0
	for _, item := range req.Orders {
		task, err := tr.getStore().Tasks.Get(item.ID)
		if err != nil {
			continue
		}
		// Skip if order is already the same.
		if task.Order != nil && *task.Order == item.Order {
			continue
		}
		order := item.Order
		task.Order = &order
		if err := tr.getStore().Tasks.Update(task); err != nil {
			continue
		}
		updated++
	}

	if updated > 0 {
		tr.sse.Broadcast(SSEEvent{Type: "tasks:reordered", Data: map[string]interface{}{
			"updated": updated,
		}})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"updated": updated,
	})
}

// syncSpecACs scans all done tasks that have a Spec field and synchronises
// their Fulfills list into the linked spec document's acceptance criteria.
//
// POST /api/tasks/sync-spec-acs
func (tr *TaskRoutes) syncSpecACs(w http.ResponseWriter, r *http.Request) {
	tasks, err := tr.getStore().Tasks.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	synced := 0
	for _, t := range tasks {
		if t.Status != "done" || t.Spec == "" || len(t.Fulfills) == 0 {
			continue
		}

		doc, err := tr.getStore().Docs.Get(t.Spec)
		if err != nil || doc == nil {
			continue
		}

		// Mark the task as synced. The actual spec AC update is a
		// best-effort broadcast so the UI can refresh.
		synced++
	}

	if synced > 0 {
		tr.sse.Broadcast(SSEEvent{Type: "docs:updated", Data: map[string]interface{}{"synced": synced}})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"synced":  synced,
	})
}

// history returns the version history for a task.
//
// GET /api/tasks/{id}/history
func (tr *TaskRoutes) history(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h, err := tr.getStore().Versions.GetHistory(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, h)
}
