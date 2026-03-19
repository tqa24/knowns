package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTaskTools registers all task-related MCP tools.
func RegisterTaskTools(s *server.MCPServer, getStore func() *storage.Store) {
	// create_task
	s.AddTool(
		mcp.NewTool("create_task",
			mcp.WithDescription("Create a new task with title and optional description, status, priority, labels, and assignee."),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("Task title"),
			),
			mcp.WithString("description",
				mcp.Description("Task description"),
			),
			mcp.WithString("status",
				mcp.Description("Task status (todo, in-progress, in-review, done, blocked)"),
				mcp.Enum("todo", "in-progress", "in-review", "done", "blocked", "on-hold", "urgent"),
			),
			mcp.WithString("priority",
				mcp.Description("Task priority (low, medium, high)"),
				mcp.Enum("low", "medium", "high"),
			),
			mcp.WithString("assignee",
				mcp.Description("Task assignee"),
			),
			mcp.WithArray("labels",
				mcp.Description("Task labels"),
				mcp.WithStringItems(),
			),
			mcp.WithString("parent",
				mcp.Description("Parent task ID for subtasks"),
			),
			mcp.WithString("spec",
				mcp.Description("Spec document path (e.g., 'specs/user-auth')"),
			),
			mcp.WithArray("fulfills",
				mcp.Description("Spec ACs this task fulfills (e.g., ['AC-1', 'AC-2'])"),
				mcp.WithStringItems(),
			),
			mcp.WithNumber("order",
				mcp.Description("Display order (lower = first)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			title, err := req.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			args := req.GetArguments()

			// Load config for defaults.
			priority := "medium"
			status := "todo"
			if proj, err := store.Config.Load(); err == nil {
				if proj.Settings.DefaultPriority != "" {
					priority = proj.Settings.DefaultPriority
				}
			}

			if s, ok := stringArg(args, "status"); ok {
				status = s
			}
			if p, ok := stringArg(args, "priority"); ok {
				priority = p
			}

			task := &models.Task{
				ID:        models.NewTaskID(),
				Title:     title,
				Status:    status,
				Priority:  priority,
				Labels:    []string{},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			if v, ok := stringArg(args, "description"); ok {
				task.Description = v
			}
			if v, ok := stringArg(args, "assignee"); ok {
				task.Assignee = v
			}
			if v, ok := stringArg(args, "parent"); ok {
				task.Parent = v
			}
			if v, ok := stringArg(args, "spec"); ok {
				task.Spec = v
			}

			if v, ok := stringSliceArg(args, "labels"); ok {
				task.Labels = v
			}
			if v, ok := stringSliceArg(args, "fulfills"); ok {
				task.Fulfills = v
			}

			if v, ok := intArg(args, "order"); ok {
				task.Order = &v
			}

			if err := store.Tasks.Create(task); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create task: %s", err.Error())), nil
			}

			search.BestEffortIndexTask(store, task.ID)

			// Notify server for real-time UI updates.
			go notifyTaskUpdated(store, task.ID)

			out, _ := json.MarshalIndent(task, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_task
	s.AddTool(
		mcp.NewTool("get_task",
			mcp.WithDescription("Get a task by ID. Returns full task details including acceptance criteria, plan, and notes."),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("Task ID to retrieve"),
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

			// Load time entries.
			if entries, err := store.Time.GetEntries(task.ID); err == nil {
				task.TimeEntries = entries
			}

			out, _ := json.MarshalIndent(task, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// update_task
	s.AddTool(
		mcp.NewTool("update_task",
			mcp.WithDescription("Update task fields including title, description, status, priority, acceptance criteria, plan, and notes."),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("Task ID to update"),
			),
			mcp.WithString("title",
				mcp.Description("New title"),
			),
			mcp.WithString("description",
				mcp.Description("New description"),
			),
			mcp.WithString("status",
				mcp.Description("New status"),
				mcp.Enum("todo", "in-progress", "in-review", "done", "blocked", "on-hold", "urgent"),
			),
			mcp.WithString("priority",
				mcp.Description("New priority"),
				mcp.Enum("low", "medium", "high"),
			),
			mcp.WithString("assignee",
				mcp.Description("New assignee"),
			),
			mcp.WithArray("labels",
				mcp.Description("New labels (replaces existing)"),
				mcp.WithStringItems(),
			),
			mcp.WithString("spec",
				mcp.Description("Spec document path (set to empty string to remove)"),
			),
			mcp.WithArray("fulfills",
				mcp.Description("Spec ACs this task fulfills"),
				mcp.WithStringItems(),
			),
			mcp.WithNumber("order",
				mcp.Description("Display order (lower = first)"),
			),
			mcp.WithArray("addAc",
				mcp.Description("Add new acceptance criteria"),
				mcp.WithStringItems(),
			),
			mcp.WithArray("checkAc",
				mcp.Description("Check AC by index (1-based)"),
				mcp.WithNumberItems(),
			),
			mcp.WithArray("uncheckAc",
				mcp.Description("Uncheck AC by index (1-based)"),
				mcp.WithNumberItems(),
			),
			mcp.WithArray("removeAc",
				mcp.Description("Remove AC by index (1-based, processed in reverse order)"),
				mcp.WithNumberItems(),
			),
			mcp.WithString("plan",
				mcp.Description("Set implementation plan"),
			),
			mcp.WithString("notes",
				mcp.Description("Set implementation notes (replaces existing)"),
			),
			mcp.WithString("appendNotes",
				mcp.Description("Append to implementation notes"),
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

			args := req.GetArguments()

			if v, ok := stringArg(args, "title"); ok {
				task.Title = v
			}
			if v, ok := stringArg(args, "description"); ok {
				task.Description = v
			}
			if v, ok := stringArg(args, "status"); ok {
				task.Status = v
			}
			if v, ok := stringArg(args, "priority"); ok {
				task.Priority = v
			}
			if v, ok := stringArg(args, "assignee"); ok {
				task.Assignee = v
			}
			if _, ok := args["labels"]; ok {
				if v, ok := stringSliceArg(args, "labels"); ok {
					task.Labels = v
				} else {
					task.Labels = []string{}
				}
			}
			if v, ok := stringArg(args, "spec"); ok {
				task.Spec = v
			}
			if _, ok := args["fulfills"]; ok {
				if v, ok := stringSliceArg(args, "fulfills"); ok {
					task.Fulfills = v
				} else {
					task.Fulfills = nil
				}
			}
			if _, ok := args["order"]; ok {
				if v, ok := intArg(args, "order"); ok {
					task.Order = &v
				}
			}

			// Acceptance criteria operations.
			if v, ok := stringSliceArg(args, "addAc"); ok {
				for _, text := range v {
					task.AcceptanceCriteria = append(task.AcceptanceCriteria, models.AcceptanceCriterion{
						Text:      text,
						Completed: false,
					})
				}
			}
			if v, ok := intSliceArg(args, "checkAc"); ok {
				for _, idx := range v {
					i := idx - 1
					if i >= 0 && i < len(task.AcceptanceCriteria) {
						task.AcceptanceCriteria[i].Completed = true
					}
				}
			}
			if v, ok := intSliceArg(args, "uncheckAc"); ok {
				for _, idx := range v {
					i := idx - 1
					if i >= 0 && i < len(task.AcceptanceCriteria) {
						task.AcceptanceCriteria[i].Completed = false
					}
				}
			}
			if v, ok := intSliceArg(args, "removeAc"); ok {
				// Sort descending so we remove from the end first.
				sorted := make([]int, len(v))
				copy(sorted, v)
				for i := 0; i < len(sorted); i++ {
					for j := i + 1; j < len(sorted); j++ {
						if sorted[j] > sorted[i] {
							sorted[i], sorted[j] = sorted[j], sorted[i]
						}
					}
				}
				for _, idx := range sorted {
					i := idx - 1
					if i >= 0 && i < len(task.AcceptanceCriteria) {
						task.AcceptanceCriteria = append(
							task.AcceptanceCriteria[:i],
							task.AcceptanceCriteria[i+1:]...,
						)
					}
				}
			}

			if v, ok := stringArg(args, "plan"); ok {
				task.ImplementationPlan = v
			}
			if v, ok := stringArg(args, "notes"); ok {
				task.ImplementationNotes = v
			}
			if v, ok := stringArg(args, "appendNotes"); ok && v != "" {
				if task.ImplementationNotes == "" {
					task.ImplementationNotes = v
				} else {
					task.ImplementationNotes += "\n" + v
				}
			}

			task.UpdatedAt = time.Now().UTC()

			if err := store.Tasks.Update(task); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update task: %s", err.Error())), nil
			}

			search.BestEffortIndexTask(store, task.ID)

			// Notify server for real-time UI updates.
			go notifyTaskUpdated(store, task.ID)

			out, _ := json.MarshalIndent(task, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// list_tasks
	s.AddTool(
		mcp.NewTool("list_tasks",
			mcp.WithDescription("List tasks with optional filters by status, priority, assignee, label, or spec."),
			mcp.WithString("status",
				mcp.Description("Filter by status"),
			),
			mcp.WithString("priority",
				mcp.Description("Filter by priority"),
			),
			mcp.WithString("assignee",
				mcp.Description("Filter by assignee"),
			),
			mcp.WithString("label",
				mcp.Description("Filter by label"),
			),
			mcp.WithString("spec",
				mcp.Description("Filter by spec document path"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			tasks, err := store.Tasks.List()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list tasks: %s", err.Error())), nil
			}

			args := req.GetArguments()
			statusFilter, _ := stringArg(args, "status")
			priorityFilter, _ := stringArg(args, "priority")
			assigneeFilter, _ := stringArg(args, "assignee")
			labelFilter, _ := stringArg(args, "label")
			specFilter, _ := stringArg(args, "spec")

			filtered := tasks[:0]
			for _, t := range tasks {
				if statusFilter != "" && t.Status != statusFilter {
					continue
				}
				if priorityFilter != "" && t.Priority != priorityFilter {
					continue
				}
				if assigneeFilter != "" && t.Assignee != assigneeFilter {
					continue
				}
				if labelFilter != "" && !containsString(t.Labels, labelFilter) {
					continue
				}
				if specFilter != "" && t.Spec != specFilter {
					continue
				}
				filtered = append(filtered, t)
			}

			out, _ := json.MarshalIndent(filtered, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// get_task_history
	s.AddTool(
		mcp.NewTool("get_task_history",
			mcp.WithDescription("Get the version history of a task, showing all changes over time."),
			mcp.WithString("taskId",
				mcp.Required(),
				mcp.Description("Task ID to get history for"),
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

			history, err := store.Versions.GetHistory(taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get task history: %s", err.Error())), nil
			}

			out, _ := json.MarshalIndent(history, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	// delete_task
	s.AddTool(
		mcp.NewTool("delete_task",
			mcp.WithDescription("Delete a task permanently. Runs in dry-run mode by default (preview only). Set dryRun: false to actually delete."),
			mcp.WithString("taskId",
				mcp.Description("Task ID to delete"),
				mcp.Required(),
			),
			mcp.WithBoolean("dryRun",
				mcp.Description("Preview only without deleting (default: true for safety)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			store := getStore()
			if store == nil {
				return mcp.NewToolResultError("No project set. Call set_project first."), nil
			}

			args := req.GetArguments()
			taskID, ok := stringArg(args, "taskId")
			if !ok || taskID == "" {
				return mcp.NewToolResultError("taskId is required"), nil
			}

			// Default to dry-run for safety.
			dryRun := true
			if v, exists := args["dryRun"]; exists {
				if b, ok := v.(bool); ok {
					dryRun = b
				}
			}

			task, err := store.Tasks.Get(taskID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Task not found: %s", err.Error())), nil
			}

			if dryRun {
				out, _ := json.MarshalIndent(map[string]any{
					"dryRun":  true,
					"message": fmt.Sprintf("Would delete task %s: %s", task.ID, task.Title),
					"task":    task,
				}, "", "  ")
				return mcp.NewToolResultText(string(out)), nil
			}

			if err := store.Tasks.Delete(taskID); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to delete task: %s", err.Error())), nil
			}

			search.BestEffortRemoveTask(store, taskID)

			// Notify server for real-time UI updates (task refresh since task was deleted).
			go notifyServer(store, "notify/refresh")

			out, _ := json.MarshalIndent(map[string]any{
				"deleted": true,
				"message": fmt.Sprintf("Deleted task %s: %s", task.ID, task.Title),
			}, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

// stringArg safely extracts a string argument from the args map.
func stringArg(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// intArg safely extracts an int argument from the args map (JSON numbers come as float64).
func intArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

// stringSliceArg extracts a []string from an args map value (which may be []any).
func stringSliceArg(args map[string]any, key string) ([]string, bool) {
	v, ok := args[key]
	if !ok {
		return nil, false
	}
	switch arr := v.(type) {
	case []string:
		return arr, true
	case []any:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, true
	}
	return nil, false
}

// intSliceArg extracts a []int from an args map value (JSON arrays of numbers come as []any of float64).
func intSliceArg(args map[string]any, key string) ([]int, bool) {
	v, ok := args[key]
	if !ok {
		return nil, false
	}
	switch arr := v.(type) {
	case []int:
		return arr, true
	case []any:
		result := make([]int, 0, len(arr))
		for _, item := range arr {
			switch n := item.(type) {
			case float64:
				result = append(result, int(n))
			case int:
				result = append(result, n)
			}
		}
		return result, true
	}
	return nil, false
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
