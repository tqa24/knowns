package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTimeTools registers all time-tracking MCP tools.
func RegisterTimeTools(s *server.MCPServer, getStore func() *storage.Store) {
	// start_time
	s.AddTool(
		mcp.NewTool("start_time",
			mcp.WithDescription("Start time tracking for a task."),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("Task ID to track time for"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			taskID, err := req.RequireString("taskId")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			task, err := store.Tasks.Get(taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", err.Error())), nil
			}

			if err := store.Time.Start(task.ID, task.Title); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to start timer: %s", err.Error())), nil
			}

			// Notify server for real-time UI updates.
			go notifyTimeUpdated(store)

			result := map[string]any{
				"success": true,
				"taskId":  task.ID,
				"message": fmt.Sprintf("Timer started for task '%s'", task.Title),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// stop_time
	s.AddTool(
		mcp.NewTool("stop_time",
			mcp.WithDescription("Stop time tracking for a task."),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("Task ID to stop tracking"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			taskID, err := req.RequireString("taskId")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			entry, err := store.Time.Stop(taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to stop timer: %s", err.Error())), nil
			}

			// Update task's timeSpent.
			task, taskErr := store.Tasks.Get(taskID)
			if taskErr == nil {
				task.TimeSpent += entry.Duration
				task.UpdatedAt = time.Now().UTC()
				_ = store.Tasks.Update(task)
			}

			// Notify server for real-time UI updates.
			go notifyTimeUpdated(store)

			result := map[string]any{
				"success":  true,
				"taskId":   taskID,
				"duration": entry.Duration,
				"entry":    entry,
				"message":  fmt.Sprintf("Timer stopped. Duration: %s", formatDuration(entry.Duration)),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// add_time
	s.AddTool(
		mcp.NewTool("add_time",
			mcp.WithDescription("Manually add a time entry to a task."),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("Task ID"),
			),
			mcp.WithString("duration",
				mcp.Required(),
				mcp.Description("Duration (e.g., '2h', '30m', '1h30m')"),
			),
			mcp.WithString("note",
				mcp.Description("Optional note for this entry"),
			),
			mcp.WithString("date",
				mcp.Description("Optional date (YYYY-MM-DD, defaults to now)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			taskID, err := req.RequireString("taskId")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			durationStr, err := req.RequireString("duration")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			durationSecs, err := models.ParseDuration(durationStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid duration %q: %s", durationStr, err.Error())), nil
			}

			task, err := store.Tasks.Get(taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", err.Error())), nil
			}

			args := req.GetArguments()

			startedAt := time.Now().UTC()
			if dateStr, ok := stringArg(args, "date"); ok && dateStr != "" {
				parsed, parseErr := time.Parse("2006-01-02", dateStr)
				if parseErr == nil {
					startedAt = parsed.UTC()
				}
			}
			endedAt := startedAt.Add(time.Duration(durationSecs) * time.Second)

			entryID := fmt.Sprintf("te-%d-%s", time.Now().UnixMilli(), taskID)
			entry := models.TimeEntry{
				ID:        entryID,
				StartedAt: startedAt,
				EndedAt:   &endedAt,
				Duration:  durationSecs,
			}
			if note, ok := stringArg(args, "note"); ok {
				entry.Note = note
			}

			if err := store.Time.SaveEntry(taskID, entry); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to save time entry: %s", err.Error())), nil
			}

			// Update task's timeSpent.
			task.TimeSpent += durationSecs
			task.UpdatedAt = time.Now().UTC()
			_ = store.Tasks.Update(task)

			// Notify server for real-time UI updates.
			go notifyTimeUpdated(store)

			result := map[string]any{
				"success":  true,
				"taskId":   taskID,
				"duration": durationSecs,
				"entry":    entry,
				"message":  fmt.Sprintf("Added %s to task '%s'", durationStr, task.Title),
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_time_report
	s.AddTool(
		mcp.NewTool("get_time_report",
			mcp.WithDescription("Get a time tracking report with optional filtering and grouping."),
			mcp.WithString("from",
				mcp.Description("Start date (YYYY-MM-DD)"),
			),
			mcp.WithString("to",
				mcp.Description("End date (YYYY-MM-DD)"),
			),
			mcp.WithString("groupBy",
				mcp.Description("Group results by task, label, or status"),
				mcp.Enum("task", "label", "status"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			args := req.GetArguments()
			fromStr, _ := stringArg(args, "from")
			toStr, _ := stringArg(args, "to")
			groupBy, _ := stringArg(args, "groupBy")

			var fromTime, toTime time.Time
			if fromStr != "" {
				fromTime, _ = time.Parse("2006-01-02", fromStr)
			}
			if toStr != "" {
				t, err := time.Parse("2006-01-02", toStr)
				if err == nil {
					// Include end of day.
					toTime = t.Add(24*time.Hour - time.Second)
				}
			}

			allEntries, err := store.Time.GetAllEntries()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to load time entries: %s", err.Error())), nil
			}

			tasks, _ := store.Tasks.List()
			taskMap := make(map[string]*models.Task)
			for _, t := range tasks {
				taskMap[t.ID] = t
			}

			type reportEntry struct {
				TaskID    string `json:"taskId"`
				TaskTitle string `json:"taskTitle,omitempty"`
				Duration  int    `json:"duration"`
				Note      string `json:"note,omitempty"`
				Date      string `json:"date,omitempty"`
			}

			var entries []reportEntry
			totalSecs := 0

			for taskID, taskEntries := range allEntries {
				for _, e := range taskEntries {
					if !fromTime.IsZero() && e.StartedAt.Before(fromTime) {
						continue
					}
					if !toTime.IsZero() && e.StartedAt.After(toTime) {
						continue
					}

					re := reportEntry{
						TaskID:   taskID,
						Duration: e.Duration,
						Note:     e.Note,
						Date:     e.StartedAt.Format("2006-01-02"),
					}
					if t, ok := taskMap[taskID]; ok {
						re.TaskTitle = t.Title
					}
					entries = append(entries, re)
					totalSecs += e.Duration
				}
			}

			// Also include active timers' elapsed time.
			state, _ := store.Time.GetState()
			if state != nil {
				for _, at := range state.Active {
					// Include active timers if they fall in the range.
					startedAt, err := time.Parse("2006-01-02T15:04:05.000Z", at.StartedAt)
					if err != nil {
						continue
					}
					if !fromTime.IsZero() && startedAt.Before(fromTime) {
						continue
					}
					elapsed := int(time.Since(startedAt).Seconds()) - int(at.TotalPausedMs/1000)
					if elapsed < 0 {
						elapsed = 0
					}
					re := reportEntry{
						TaskID:   at.TaskID,
						Duration: elapsed,
						Note:     "(active timer)",
						Date:     startedAt.Format("2006-01-02"),
					}
					if t, ok := taskMap[at.TaskID]; ok {
						re.TaskTitle = t.Title
					}
					entries = append(entries, re)
					totalSecs += elapsed
				}
			}

			var reportData any

			if groupBy != "" {
				type group struct {
					Key      string `json:"key"`
					Duration int    `json:"duration"`
					Count    int    `json:"count"`
				}
				groupMap := make(map[string]*group)

				for _, e := range entries {
					var key string
					switch groupBy {
					case "task":
						key = e.TaskID
						if e.TaskTitle != "" {
							key = e.TaskTitle + " (" + e.TaskID + ")"
						}
					case "label":
						task := taskMap[e.TaskID]
						if task != nil && len(task.Labels) > 0 {
							for _, label := range task.Labels {
								if g, ok := groupMap[label]; ok {
									g.Duration += e.Duration
									g.Count++
								} else {
									groupMap[label] = &group{Key: label, Duration: e.Duration, Count: 1}
								}
							}
							continue
						}
						key = "(no label)"
					case "status":
						task := taskMap[e.TaskID]
						if task != nil {
							key = task.Status
						} else {
							key = "(unknown)"
						}
					}
					if g, ok := groupMap[key]; ok {
						g.Duration += e.Duration
						g.Count++
					} else {
						groupMap[key] = &group{Key: key, Duration: e.Duration, Count: 1}
					}
				}

				var groups []*group
				for _, g := range groupMap {
					groups = append(groups, g)
				}
				reportData = map[string]any{
					"groups":    groups,
					"total":     totalSecs,
					"formatted": formatDuration(totalSecs),
					"groupBy":   groupBy,
					"from":      fromStr,
					"to":        toStr,
				}
			} else {
				reportData = map[string]any{
					"entries":   entries,
					"total":     totalSecs,
					"formatted": formatDuration(totalSecs),
					"from":      fromStr,
					"to":        toStr,
				}
			}

			out, _ := json.MarshalIndent(reportData, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

// formatDuration formats a duration in seconds to a human-readable string.
func formatDuration(secs int) string {
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		if m > 0 || s > 0 {
			return fmt.Sprintf("%dh%02dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	if m > 0 {
		if s > 0 {
			return fmt.Sprintf("%dm%02ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", s)
}
