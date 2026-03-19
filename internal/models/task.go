package models

import "time"

// Task represents a task in the .knowns/ system.
// The ID uses a 6-character base36 format (e.g., "abc123").
// Legacy sequential integer IDs are also supported for backward compatibility.
type Task struct {
	ID          string   `json:"id"                    yaml:"id"`
	Title       string   `json:"title"                 yaml:"title"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Status      string   `json:"status"                yaml:"status"`
	Priority    string   `json:"priority"              yaml:"priority"` // "low", "medium", "high"
	Assignee    string   `json:"assignee,omitempty"    yaml:"assignee,omitempty"`
	Labels      []string `json:"labels"                yaml:"labels"`

	// Parent task ID for subtasks (e.g., "abc123").
	Parent string `json:"parent,omitempty" yaml:"parent,omitempty"`

	// Subtasks holds child task IDs. Derived at load time; not persisted in the
	// task file itself.
	Subtasks []string `json:"subtasks,omitempty" yaml:"-"`

	// Spec is the linked spec document path (e.g., "specs/user-auth").
	Spec string `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Fulfills lists spec acceptance-criteria IDs this task satisfies
	// (e.g., ["AC-1", "AC-2"]).
	Fulfills []string `json:"fulfills,omitempty" yaml:"fulfills,omitempty"`

	// Order is an optional manual display order (lower = first).
	Order *int `json:"order,omitempty" yaml:"order,omitempty"`

	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" yaml:"updatedAt"`

	// AcceptanceCriteria lives in the markdown body, not the YAML frontmatter.
	AcceptanceCriteria []AcceptanceCriterion `json:"acceptanceCriteria" yaml:"-"`

	// TimeSpent is the total time tracked in seconds.
	TimeSpent   int         `json:"timeSpent"            yaml:"timeSpent"`
	TimeEntries []TimeEntry `json:"timeEntries,omitempty" yaml:"-"`

	// ImplementationPlan and ImplementationNotes live in the markdown body.
	ImplementationPlan  string `json:"implementationPlan,omitempty"  yaml:"-"`
	ImplementationNotes string `json:"implementationNotes,omitempty" yaml:"-"`
}

// AcceptanceCriterion is a single checkbox item in the Acceptance Criteria
// section of a task file.
type AcceptanceCriterion struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// DefaultStatuses returns the built-in task status list used when no project
// config is available.
func DefaultStatuses() []string {
	return []string{
		"todo",
		"in-progress",
		"in-review",
		"done",
		"blocked",
		"on-hold",
		"urgent",
	}
}

// ValidPriorities returns the allowed priority values.
func ValidPriorities() []string {
	return []string{"low", "medium", "high"}
}
