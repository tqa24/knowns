package models

import "time"

// TaskVersion represents a snapshot of a task at a specific point in time.
// Versions are identified by an incrementing integer wrapped in a "v" prefix
// string (e.g., "v1", "v2").
type TaskVersion struct {
	// ID is the human-readable version label (e.g., "v1").
	ID string `json:"id"`

	TaskID    string    `json:"taskId"`
	Version   int       `json:"version"`
	Timestamp time.Time `json:"timestamp"`

	// Author records who triggered the change (username or "rollback").
	Author string `json:"author,omitempty"`

	// Changes is the list of individual field mutations in this version.
	Changes []TaskChange `json:"changes"`

	// Snapshot is a full copy of the task state at this version, stored as a
	// generic map so that the version store remains decoupled from Task field
	// additions over time.
	Snapshot map[string]any `json:"snapshot"`
}

// TaskChange describes a mutation of a single field between two task versions.
type TaskChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"oldValue"`
	NewValue any    `json:"newValue"`
}

// TaskVersionHistory is the complete audit trail for one task.
type TaskVersionHistory struct {
	TaskID         string        `json:"taskId"`
	CurrentVersion int           `json:"currentVersion"`
	Versions       []TaskVersion `json:"versions"`
}

// TaskSnapshot is a typed representation of the fields captured in a version
// snapshot.  It mirrors the TRACKED_FIELDS list from the TypeScript model.
// Storage uses map[string]any on TaskVersion.Snapshot for forward
// compatibility; this struct can be used for typed access after unmarshaling.
type TaskSnapshot struct {
	Title               string                `json:"title"`
	Description         string                `json:"description,omitempty"`
	Status              string                `json:"status"`
	Priority            string                `json:"priority"`
	Assignee            string                `json:"assignee,omitempty"`
	Labels              []string              `json:"labels,omitempty"`
	AcceptanceCriteria  []AcceptanceCriterion `json:"acceptanceCriteria,omitempty"`
	ImplementationPlan  string                `json:"implementationPlan,omitempty"`
	ImplementationNotes string                `json:"implementationNotes,omitempty"`
}

// DocVersion represents a snapshot of a doc at a specific point in time.
type DocVersion struct {
	ID        string         `json:"id"`
	DocPath   string         `json:"docPath"`
	Version   int            `json:"version"`
	Timestamp time.Time      `json:"timestamp"`
	Author    string         `json:"author,omitempty"`
	Changes   []DocChange    `json:"changes"`
	Snapshot  map[string]any `json:"snapshot"`
}

// DocChange describes a mutation of a single field between two doc versions.
type DocChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"oldValue"`
	NewValue any    `json:"newValue"`
}

// DocVersionHistory is the complete audit trail for one document.
type DocVersionHistory struct {
	DocPath        string       `json:"docPath"`
	CurrentVersion int          `json:"currentVersion"`
	Versions       []DocVersion `json:"versions"`
}
