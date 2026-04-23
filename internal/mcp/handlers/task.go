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

// RegisterTaskTool registers the consolidated task management MCP tool.
// This includes the board view (previously in board.go).
func RegisterTaskTool(s *server.MCPServer, getStore func() *storage.Store) {
	s.AddTool(
		mcp.NewTool("tasks",
			mcp.WithDescription("Task management operations. Use 'action' to specify: create, get, update, delete, list, history, board."),
			mcp.WithString("action",
				mcp.Required(),
				mcp.Description("Action to perform"),
				mcp.Enum("create", "get", "update", "delete", "list", "history", "board"),
			),
			mcp.WithString("taskId",
				mcp.Description("Task ID (required for get, update, delete, history)"),
			),
			mcp.WithString("title",
				mcp.Description("Task title (required for create, optional for update)"),
			),
			mcp.WithString("description",
				mcp.Description("Task description (create, update)"),
			),
			mcp.WithString("status",
				mcp.Description("Task status (create, update, list)"),
				mcp.Enum("todo", "in-progress", "in-review", "done", "blocked", "on-hold", "urgent"),
			),
			mcp.WithString("priority",
				mcp.Description("Task priority (create, update, list)"),
				mcp.Enum("low", "medium", "high"),
			),
			mcp.WithString("assignee",
				mcp.Description("Task assignee (create, update, list)"),
			),
			mcp.WithArray("labels",
				mcp.Description("Task labels (create, update)"),
				mcp.WithStringItems(),
			),
			mcp.WithString("parent",
				mcp.Description("Parent task ID for subtasks (create)"),
			),
			mcp.WithString("spec",
				mcp.Description("Spec document path (create, update, list)"),
			),
			mcp.WithArray("fulfills",
				mcp.Description("Spec ACs this task fulfills (create, update)"),
				mcp.WithStringItems(),
			),
			mcp.WithNumber("order",
				mcp.Description("Display order (create, update)"),
			),
			mcp.WithArray("addAc",
				mcp.Description("Add new acceptance criteria (update)"),
				mcp.WithStringItems(),
			),
			mcp.WithArray("checkAc",
				mcp.Description("Check AC by index, 1-based (update)"),
				mcp.WithNumberItems(),
			),
			mcp.WithArray("uncheckAc",
				mcp.Description("Uncheck AC by index, 1-based (update)"),
				mcp.WithNumberItems(),
			),
			mcp.WithArray("removeAc",
				mcp.Description("Remove AC by index, 1-based (update)"),
				mcp.WithNumberItems(),
			),
			mcp.WithString("plan",
				mcp.Description("Implementation plan (update)"),
			),
			mcp.WithString("notes",
				mcp.Description("Implementation notes, replaces existing (update)"),
			),
			mcp.WithString("appendNotes",
				mcp.Description("Append to implementation notes (update)"),
			),
			mcp.WithArray("clear",
				mcp.Description("Clear string fields (update)"),
				mcp.WithStringItems(),
			),
			mcp.WithString("label",
				mcp.Description("Filter by label (list)"),
			),
			mcp.WithBoolean("dryRun",
				mcp.Description("Preview only without deleting (default: true) (delete)"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			action, err := req.RequireString("action")
			if err != nil {
				return errResult("action is required")
			}
			switch action {
			case "create":
				return handleTaskCreate(getStore, req)
			case "get":
				return handleTaskGet(getStore, req)
			case "update":
				return handleTaskUpdate(getStore, req)
			case "delete":
				return handleTaskDelete(getStore, req)
			case "list":
				return handleTaskList(getStore, req)
			case "history":
				return handleTaskHistory(getStore, req)
			case "board":
				return handleTaskBoard(getStore, req)
			default:
				return errResultf("unknown tasks action: %s", action)
			}
		},
	)
}

func handleTaskCreate(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	title, err := req.RequireString("title")
	if err != nil {
		return errResult(err.Error())
	}

	args := req.GetArguments()

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
	if v, ok := stringArg(args, "plan"); ok {
		task.ImplementationPlan = v
	}
	if v, ok := stringArg(args, "notes"); ok {
		task.ImplementationNotes = v
	}

	if err := store.Tasks.Create(task); err != nil {
		return errFailed("create task", err)
	}

	_ = store.Versions.SaveVersion(task.ID, models.TaskVersion{
		Changes:  store.Versions.TrackChanges(nil, task),
		Snapshot: storage.TaskToSnapshot(task),
	})

	search.BestEffortIndexTask(store, task.ID)
	go notifyTaskUpdated(store, task.ID)

	out, _ := json.MarshalIndent(task, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleTaskGet(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	taskID, err := req.RequireString("taskId")
	if err != nil {
		return errResult(ErrTaskIDReq)
	}

	task, err := store.Tasks.Get(taskID)
	if err != nil {
		return errNotFound("Task", err)
	}

	if entries, err := store.Time.GetEntries(task.ID); err == nil {
		task.TimeEntries = entries
	}
	task.ActiveTimer = store.Time.GetActiveTimer(task.ID)

	out, _ := json.MarshalIndent(task, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleTaskUpdate(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	taskID, err := req.RequireString("taskId")
	if err != nil {
		return errResult(ErrTaskIDReq)
	}

	task, err := store.Tasks.Get(taskID)
	if err != nil {
		return errNotFound("Task", err)
	}

	oldTask := *task
	args := req.GetArguments()
	clearFields := stringSetArg(args, "clear")

	if clearFields["title"] {
		task.Title = ""
	} else if v, ok := stringArg(args, "title"); ok && v != "" {
		task.Title = v
	}
	if clearFields["description"] {
		task.Description = ""
	} else if v, ok := stringArg(args, "description"); ok && v != "" {
		task.Description = v
	}
	if v, ok := stringArg(args, "status"); ok {
		task.Status = v
	}
	if v, ok := stringArg(args, "priority"); ok {
		task.Priority = v
	}
	if clearFields["assignee"] {
		task.Assignee = ""
	} else if v, ok := stringArg(args, "assignee"); ok && v != "" {
		task.Assignee = v
	}
	if _, ok := args["labels"]; ok {
		if v, ok := stringSliceArg(args, "labels"); ok {
			task.Labels = v
		} else {
			task.Labels = []string{}
		}
	}
	if clearFields["spec"] {
		task.Spec = ""
	} else if v, ok := stringArg(args, "spec"); ok && v != "" {
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

	if clearFields["plan"] {
		task.ImplementationPlan = ""
	} else if v, ok := stringArg(args, "plan"); ok && v != "" {
		task.ImplementationPlan = v
	}
	if clearFields["notes"] {
		task.ImplementationNotes = ""
	} else if v, ok := stringArg(args, "notes"); ok && v != "" {
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
		return errFailed("update task", err)
	}

	if changes := store.Versions.TrackChanges(&oldTask, task); len(changes) > 0 {
		_ = store.Versions.SaveVersion(task.ID, models.TaskVersion{
			Changes:  changes,
			Snapshot: storage.TaskToSnapshot(task),
		})
	}

	search.BestEffortIndexTask(store, task.ID)
	go notifyTaskUpdated(store, task.ID)

	task.ActiveTimer = store.Time.GetActiveTimer(task.ID)

	out, _ := json.MarshalIndent(task, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleTaskList(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	tasks, err := store.Tasks.List()
	if err != nil {
		return errFailed("list tasks", err)
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

	// If results are empty, include project context for diagnostics.
	if len(filtered) == 0 {
		wrapper := map[string]any{
			"results":      filtered,
			"_projectRoot": store.Root,
			"_hint":        "No tasks found. Verify the active project is correct via project({ action: \"current\" }).",
		}
		out, _ := json.MarshalIndent(wrapper, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	}

	out, _ := json.MarshalIndent(filtered, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleTaskHistory(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	taskID, err := req.RequireString("taskId")
	if err != nil {
		return errResult(ErrTaskIDReq)
	}

	history, err := store.Versions.GetHistory(taskID)
	if err != nil {
		return errFailed("get task history", err)
	}

	out, _ := json.MarshalIndent(history, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleTaskDelete(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	args := req.GetArguments()
	taskID, ok := stringArg(args, "taskId")
	if !ok || taskID == "" {
		return errResult(ErrTaskIDReq)
	}

	dryRun := true
	if v, exists := args["dryRun"]; exists {
		if b, ok := v.(bool); ok {
			dryRun = b
		}
	}

	task, err := store.Tasks.Get(taskID)
	if err != nil {
		return errNotFound("Task", err)
	}

	if dryRun {
		out, _ := json.MarshalIndent(map[string]any{
			"dryRun":  true,
			"message": fmt.Sprintf(MsgWouldDeleteTask, task.ID, task.Title),
			"task":    task,
		}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	}

	if err := store.Tasks.Delete(taskID); err != nil {
		return errFailed("delete task", err)
	}

	search.BestEffortRemoveTask(store, taskID)
	go notifyServer(store, "notify/refresh")

	out, _ := json.MarshalIndent(map[string]any{
		"deleted": true,
		"message": fmt.Sprintf(MsgDeletedTask, task.ID, task.Title),
	}, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

// handleTaskBoard returns the board view with tasks grouped by status.
// Previously in board.go as get_board tool.
func handleTaskBoard(getStore func() *storage.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	store := getStore()
	if store == nil {
		return noProjectError()
	}

	tasks, err := store.Tasks.List()
	if err != nil {
		return errFailed("list tasks", err)
	}

	columns := models.DefaultStatuses()
	if proj, err := store.Config.Load(); err == nil {
		if len(proj.Settings.VisibleColumns) > 0 {
			columns = proj.Settings.VisibleColumns
		} else if len(proj.Settings.Statuses) > 0 {
			columns = proj.Settings.Statuses
		}
	}

	type column struct {
		Status string         `json:"status"`
		Tasks  []*models.Task `json:"tasks"`
		Count  int            `json:"count"`
	}

	columnMap := make(map[string]*column)
	for _, status := range columns {
		columnMap[status] = &column{
			Status: status,
			Tasks:  []*models.Task{},
		}
	}

	extraStatuses := make(map[string]bool)
	for _, t := range tasks {
		if _, ok := columnMap[t.Status]; !ok {
			extraStatuses[t.Status] = true
			if _, ok2 := columnMap[t.Status]; !ok2 {
				columnMap[t.Status] = &column{
					Status: t.Status,
					Tasks:  []*models.Task{},
				}
			}
		}
		columnMap[t.Status].Tasks = append(columnMap[t.Status].Tasks, t)
		columnMap[t.Status].Count++
	}

	var ordered []*column
	for _, status := range columns {
		if c, ok := columnMap[status]; ok {
			ordered = append(ordered, c)
		}
	}
	for status := range extraStatuses {
		if c, ok := columnMap[status]; ok {
			ordered = append(ordered, c)
		}
	}

	totalCount := 0
	for _, c := range ordered {
		totalCount += c.Count
	}

	board := map[string]any{
		"columns": ordered,
		"total":   totalCount,
	}

	out, _ := json.MarshalIndent(board, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}
