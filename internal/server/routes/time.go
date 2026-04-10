package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/howznguyen/knowns/internal/storage"
)

// TimeRoutes handles /api/time endpoints.
type TimeRoutes struct {
	store *storage.Store
	mgr   *storage.Manager
	sse   Broadcaster
}

func (tr *TimeRoutes) getStore() *storage.Store {
	if tr.mgr != nil {
		return tr.mgr.GetStore()
	}
	return tr.store
}

// Register wires the time-tracking routes onto r.
func (tr *TimeRoutes) Register(r chi.Router) {
	r.Get("/time/status", tr.status)
	r.Post("/time/start", tr.start)
	r.Post("/time/stop", tr.stop)
	r.Post("/time/pause", tr.pause)
	r.Post("/time/resume", tr.resume)
}

// status lists all active timers.
//
// GET /api/time/status
func (tr *TimeRoutes) status(w http.ResponseWriter, r *http.Request) {
	state, err := tr.getStore().Time.GetState()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, state)
}

// startRequest is the body for POST /api/time/start.
type startRequest struct {
	TaskID string `json:"taskId"`
}

// start begins a timer for a task.
//
// POST /api/time/start
func (tr *TimeRoutes) start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.TaskID == "" {
		respondError(w, http.StatusBadRequest, "taskId is required")
		return
	}

	task, err := tr.getStore().Tasks.Get(req.TaskID)
	if err != nil {
		respondError(w, http.StatusNotFound, "task not found: "+err.Error())
		return
	}

	if err := tr.getStore().Time.Start(req.TaskID, task.Title); err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	state, _ := tr.getStore().Time.GetState()
	tr.sse.Broadcast(SSEEvent{Type: "time:updated", Data: state})
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"taskId": req.TaskID,
		"status": "started",
		"active": state.Active,
	})
}

// stopRequest is the body for POST /api/time/stop.
type stopRequest struct {
	TaskID string `json:"taskId"`
	All    bool   `json:"all"`
}

// stop terminates one or all active timers.
//
// POST /api/time/stop
func (tr *TimeRoutes) stop(w http.ResponseWriter, r *http.Request) {
	var req stopRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.All {
		state, err := tr.getStore().Time.GetState()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		var stopped []string
		for _, a := range state.Active {
			entry, err := tr.getStore().Time.Stop(a.TaskID)
			if err == nil {
				stopped = append(stopped, a.TaskID)
				// Update task's timeSpent
				if task, taskErr := tr.getStore().Tasks.Get(a.TaskID); taskErr == nil {
					task.TimeSpent += entry.Duration
					task.UpdatedAt = time.Now().UTC()
					_ = tr.getStore().Tasks.Update(task)
				}
			}
		}
		newState, _ := tr.getStore().Time.GetState()
		tr.sse.Broadcast(SSEEvent{Type: "time:updated", Data: newState})
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"stopped": stopped,
			"active":  newState.Active,
		})
		return
	}

	if req.TaskID == "" {
		respondError(w, http.StatusBadRequest, "taskId is required (or set all:true)")
		return
	}
	entry, err := tr.getStore().Time.Stop(req.TaskID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	// Update task's timeSpent
	if task, taskErr := tr.getStore().Tasks.Get(req.TaskID); taskErr == nil {
		task.TimeSpent += entry.Duration
		task.UpdatedAt = time.Now().UTC()
		_ = tr.getStore().Tasks.Update(task)
	}
	state, _ := tr.getStore().Time.GetState()
	tr.sse.Broadcast(SSEEvent{Type: "time:updated", Data: state})
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stopped": []interface{}{entry},
		"active":  state.Active,
	})
}

// pauseResumeRequest is the body for /api/time/pause and /api/time/resume.
type pauseResumeRequest struct {
	TaskID string `json:"taskId"`
}

// pause pauses the timer for a task.
//
// POST /api/time/pause
func (tr *TimeRoutes) pause(w http.ResponseWriter, r *http.Request) {
	var req pauseResumeRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.TaskID == "" {
		respondError(w, http.StatusBadRequest, "taskId is required")
		return
	}
	if err := tr.getStore().Time.Pause(req.TaskID); err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	state, _ := tr.getStore().Time.GetState()
	tr.sse.Broadcast(SSEEvent{Type: "time:updated", Data: state})
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"taskId": req.TaskID,
		"status": "paused",
		"active": state.Active,
	})
}

// resume resumes a paused timer.
//
// POST /api/time/resume
func (tr *TimeRoutes) resume(w http.ResponseWriter, r *http.Request) {
	var req pauseResumeRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.TaskID == "" {
		respondError(w, http.StatusBadRequest, "taskId is required")
		return
	}
	if err := tr.getStore().Time.Resume(req.TaskID); err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	state, _ := tr.getStore().Time.GetState()
	tr.sse.Broadcast(SSEEvent{Type: "time:updated", Data: state})
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"taskId": req.TaskID,
		"status": "resumed",
		"active": state.Active,
	})
}
