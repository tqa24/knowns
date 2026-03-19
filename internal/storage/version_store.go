package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/howznguyen/knowns/internal/models"
)

// VersionStore reads and writes task version histories from .knowns/versions/.
type VersionStore struct {
	root string
}

func (vs *VersionStore) versionsDir() string { return filepath.Join(vs.root, "versions") }

func (vs *VersionStore) versionPath(taskID string) string {
	return filepath.Join(vs.versionsDir(), "task-"+taskID+".json")
}

// GetHistory returns the full version history for a task.
// Returns an empty history (not an error) if no history file exists.
func (vs *VersionStore) GetHistory(taskID string) (*models.TaskVersionHistory, error) {
	data, err := os.ReadFile(vs.versionPath(taskID))
	if os.IsNotExist(err) {
		return &models.TaskVersionHistory{
			TaskID:         taskID,
			CurrentVersion: 0,
			Versions:       []models.TaskVersion{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read version history %s: %w", taskID, err)
	}
	var h models.TaskVersionHistory
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse version history %s: %w", taskID, err)
	}
	if h.Versions == nil {
		h.Versions = []models.TaskVersion{}
	}
	return &h, nil
}

// SaveVersion appends a new version entry and updates CurrentVersion.
// version.Snapshot, Changes, and Timestamp should be populated by the caller
// (or use TrackChanges + TaskToSnapshot helpers).
func (vs *VersionStore) SaveVersion(taskID string, version models.TaskVersion) error {
	h, err := vs.GetHistory(taskID)
	if err != nil {
		return err
	}

	h.CurrentVersion++
	version.ID = fmt.Sprintf("v%d", h.CurrentVersion)
	version.Version = h.CurrentVersion
	version.TaskID = taskID
	if version.Timestamp.IsZero() {
		version.Timestamp = time.Now().UTC()
	}

	h.Versions = append(h.Versions, version)

	if err := os.MkdirAll(vs.versionsDir(), 0755); err != nil {
		return fmt.Errorf("mkdir versions: %w", err)
	}
	return writeJSON(vs.versionPath(taskID), h)
}

// TrackChanges compares two Task values and returns the list of changed fields.
// oldTask may be nil (for the initial creation version).
func (vs *VersionStore) TrackChanges(oldTask, newTask *models.Task) []models.TaskChange {
	var changes []models.TaskChange

	if oldTask == nil {
		// Creation: record non-zero new values as additions.
		if newTask.Status != "" {
			changes = append(changes, models.TaskChange{Field: "status", NewValue: newTask.Status})
		}
		if newTask.Priority != "" {
			changes = append(changes, models.TaskChange{Field: "priority", NewValue: newTask.Priority})
		}
		return changes
	}

	diff := func(field string, old, new any) {
		if !reflect.DeepEqual(old, new) {
			ch := models.TaskChange{Field: field}
			if !isZero(old) {
				ch.OldValue = old
			}
			if !isZero(new) {
				ch.NewValue = new
			}
			changes = append(changes, ch)
		}
	}

	diff("title", oldTask.Title, newTask.Title)
	diff("status", oldTask.Status, newTask.Status)
	diff("priority", oldTask.Priority, newTask.Priority)
	diff("assignee", oldTask.Assignee, newTask.Assignee)
	diff("labels", oldTask.Labels, newTask.Labels)
	diff("description", oldTask.Description, newTask.Description)
	diff("acceptanceCriteria", oldTask.AcceptanceCriteria, newTask.AcceptanceCriteria)
	diff("implementationPlan", oldTask.ImplementationPlan, newTask.ImplementationPlan)
	diff("implementationNotes", oldTask.ImplementationNotes, newTask.ImplementationNotes)

	return changes
}

// TaskToSnapshot converts a Task to the generic map snapshot stored in
// TaskVersion.Snapshot. This matches the TypeScript version history format.
func TaskToSnapshot(task *models.Task) map[string]any {
	snap := map[string]any{
		"title":    task.Title,
		"status":   task.Status,
		"priority": task.Priority,
	}
	if task.Description != "" {
		snap["description"] = task.Description
	}
	if task.Assignee != "" {
		snap["assignee"] = task.Assignee
	}
	if len(task.Labels) > 0 {
		snap["labels"] = task.Labels
	}
	if len(task.AcceptanceCriteria) > 0 {
		snap["acceptanceCriteria"] = task.AcceptanceCriteria
	}
	if task.ImplementationPlan != "" {
		snap["implementationPlan"] = task.ImplementationPlan
	}
	if task.ImplementationNotes != "" {
		snap["implementationNotes"] = task.ImplementationNotes
	}
	return snap
}

// --- Activity feed ---

// ActivityEntry is a denormalized version entry for the activity feed.
type ActivityEntry struct {
	TaskID    string              `json:"taskId"`
	TaskTitle string              `json:"taskTitle"`
	Version   int                 `json:"version"`
	Timestamp time.Time           `json:"timestamp"`
	Author    string              `json:"author,omitempty"`
	Changes   []models.TaskChange `json:"changes"`
}

// ListRecentActivities scans all task version histories and returns the most
// recent version entries across all tasks, sorted by timestamp descending.
// typeFilter optionally filters to entries containing changes of a certain
// category ("status", "assignee", "content").
func (vs *VersionStore) ListRecentActivities(limit int, typeFilter string) ([]ActivityEntry, error) {
	dir := vs.versionsDir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ActivityEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read versions dir: %w", err)
	}

	var all []ActivityEntry

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "task-") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var h models.TaskVersionHistory
		if err := json.Unmarshal(data, &h); err != nil {
			continue
		}

		for _, v := range h.Versions {
			if len(v.Changes) == 0 {
				continue
			}

			// Apply type filter
			if typeFilter != "" && !matchesTypeFilter(v.Changes, typeFilter) {
				continue
			}

			// Extract task title from snapshot
			title, _ := v.Snapshot["title"].(string)

			all = append(all, ActivityEntry{
				TaskID:    h.TaskID,
				TaskTitle: title,
				Version:   v.Version,
				Timestamp: v.Timestamp,
				Author:    v.Author,
				Changes:   v.Changes,
			})
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp.After(all[j].Timestamp)
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

// matchesTypeFilter checks if any change in the list matches the category.
func matchesTypeFilter(changes []models.TaskChange, category string) bool {
	for _, c := range changes {
		switch category {
		case "status":
			if c.Field == "status" {
				return true
			}
		case "assignee":
			if c.Field == "assignee" {
				return true
			}
		case "content":
			if c.Field == "title" || c.Field == "description" || c.Field == "acceptanceCriteria" ||
				c.Field == "implementationPlan" || c.Field == "implementationNotes" {
				return true
			}
		}
	}
	return false
}

// --- Doc version tracking ---

// docVersionPath returns the file path for a doc's version history.
// Slashes in the doc path are replaced with "--" to create a flat filename.
func (vs *VersionStore) docVersionPath(docPath string) string {
	safe := strings.ReplaceAll(docPath, "/", "--")
	return filepath.Join(vs.versionsDir(), "doc-"+safe+".json")
}

// GetDocHistory returns the full version history for a document.
// Returns an empty history (not an error) if no history file exists.
func (vs *VersionStore) GetDocHistory(docPath string) (*models.DocVersionHistory, error) {
	data, err := os.ReadFile(vs.docVersionPath(docPath))
	if os.IsNotExist(err) {
		return &models.DocVersionHistory{
			DocPath:        docPath,
			CurrentVersion: 0,
			Versions:       []models.DocVersion{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read doc version history %s: %w", docPath, err)
	}
	var h models.DocVersionHistory
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse doc version history %s: %w", docPath, err)
	}
	if h.Versions == nil {
		h.Versions = []models.DocVersion{}
	}
	return &h, nil
}

// SaveDocVersion appends a new version entry and updates CurrentVersion.
func (vs *VersionStore) SaveDocVersion(docPath string, version models.DocVersion) error {
	h, err := vs.GetDocHistory(docPath)
	if err != nil {
		return err
	}

	h.CurrentVersion++
	version.ID = fmt.Sprintf("v%d", h.CurrentVersion)
	version.Version = h.CurrentVersion
	version.DocPath = docPath
	if version.Timestamp.IsZero() {
		version.Timestamp = time.Now().UTC()
	}

	h.Versions = append(h.Versions, version)

	if err := os.MkdirAll(vs.versionsDir(), 0755); err != nil {
		return fmt.Errorf("mkdir versions: %w", err)
	}
	return writeJSON(vs.docVersionPath(docPath), h)
}

// TrackDocChanges compares two Doc values and returns the list of changed fields.
// oldDoc may be nil (for the initial creation version).
func (vs *VersionStore) TrackDocChanges(oldDoc, newDoc *models.Doc) []models.DocChange {
	var changes []models.DocChange

	if oldDoc == nil {
		if newDoc.Title != "" {
			changes = append(changes, models.DocChange{Field: "title", NewValue: newDoc.Title})
		}
		if newDoc.Description != "" {
			changes = append(changes, models.DocChange{Field: "description", NewValue: newDoc.Description})
		}
		if len(newDoc.Tags) > 0 {
			changes = append(changes, models.DocChange{Field: "tags", NewValue: newDoc.Tags})
		}
		return changes
	}

	diff := func(field string, old, newVal any) {
		if !reflect.DeepEqual(old, newVal) {
			ch := models.DocChange{Field: field}
			if !isZero(old) {
				ch.OldValue = old
			}
			if !isZero(newVal) {
				ch.NewValue = newVal
			}
			changes = append(changes, ch)
		}
	}

	diff("title", oldDoc.Title, newDoc.Title)
	diff("description", oldDoc.Description, newDoc.Description)
	diff("content", oldDoc.Content, newDoc.Content)
	diff("tags", oldDoc.Tags, newDoc.Tags)

	return changes
}

// DocToSnapshot converts a Doc to a generic map snapshot.
func DocToSnapshot(doc *models.Doc) map[string]any {
	snap := map[string]any{
		"title": doc.Title,
	}
	if doc.Description != "" {
		snap["description"] = doc.Description
	}
	if doc.Content != "" {
		snap["content"] = doc.Content
	}
	if len(doc.Tags) > 0 {
		snap["tags"] = doc.Tags
	}
	return snap
}

// isZero reports whether v is the zero value of its type.
func isZero(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return x == ""
	case int:
		return x == 0
	case bool:
		return !x
	case []string:
		return len(x) == 0
	case []models.AcceptanceCriterion:
		return len(x) == 0
	}
	return false
}
