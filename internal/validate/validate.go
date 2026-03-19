// Package validate provides shared validation logic for tasks, docs, and templates.
// Both CLI and MCP handlers use this package to ensure consistent checks.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/howznguyen/knowns/internal/codegen"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
)

// Issue represents a single validation problem.
type Issue struct {
	Level   string `json:"level"`            // "error", "warning", "info"
	Code    string `json:"code"`             // e.g. "TASK_NO_TITLE"
	Message string `json:"message"`          // human-readable description
	Entity  string `json:"entity,omitempty"` // task ID or doc path
	Fixed   bool   `json:"fixed,omitempty"`  // true if auto-fixed
}

// Result holds the outcome of a validation run.
type Result struct {
	Issues       []Issue `json:"issues"`
	ErrorCount   int     `json:"errorCount"`
	WarningCount int     `json:"warningCount"`
	InfoCount    int     `json:"infoCount"`
	Valid        bool    `json:"valid"`
}

// Options configures the validation run.
type Options struct {
	Scope  string // "all", "tasks", "docs", "templates", "sdd"
	Entity string // validate a single entity (task ID or doc path)
	Strict bool   // treat warnings as errors
	Fix    bool   // auto-fix supported issues
}

// Reference-detection regexes.
var (
	taskRefRE = regexp.MustCompile(`@task-([a-z0-9]+)`)
	docRefRE  = regexp.MustCompile(`@doc/([^\s\)]+)`)
)

// Valid status and priority values.
var (
	validStatuses = map[string]bool{
		"todo": true, "in-progress": true, "in-review": true,
		"done": true, "blocked": true, "on-hold": true, "urgent": true,
	}
	validPriorities = map[string]bool{
		"low": true, "medium": true, "high": true,
	}
)

// Run executes all validation checks according to opts and returns the result.
func Run(store *storage.Store, opts Options) *Result {
	if opts.Scope == "" {
		opts.Scope = "all"
	}

	var issues []Issue

	// Load all tasks and docs for cross-reference validation.
	tasks, _ := store.Tasks.List()
	docs, _ := store.Docs.List()

	taskIDs := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		taskIDs[t.ID] = true
	}
	docPaths := make(map[string]bool, len(docs))
	for _, d := range docs {
		docPaths[d.Path] = true
	}

	// Build parent map for circular detection.
	parentMap := make(map[string]string, len(tasks))
	for _, t := range tasks {
		if t.Parent != "" {
			parentMap[t.ID] = t.Parent
		}
	}

	// --- Tasks ---
	if opts.Scope == "all" || opts.Scope == "tasks" || opts.Scope == "sdd" {
		for _, t := range tasks {
			if opts.Entity != "" && opts.Entity != t.ID {
				continue
			}
			issues = append(issues, validateTask(t, taskIDs, docPaths, parentMap, opts)...)
		}
	}

	// --- Docs ---
	if opts.Scope == "all" || opts.Scope == "docs" || opts.Scope == "sdd" {
		for _, d := range docs {
			if d.IsImported {
				continue
			}
			if opts.Entity != "" && opts.Entity != d.Path {
				continue
			}
			fullDoc, err := store.Docs.Get(d.Path)
			if err != nil {
				issues = append(issues, Issue{
					Level:   "error",
					Code:    "DOC_PARSE_ERROR",
					Message: fmt.Sprintf("Failed to parse doc: %s", err.Error()),
					Entity:  d.Path,
				})
				continue
			}
			issues = append(issues, validateDoc(fullDoc, taskIDs, docPaths)...)
		}
	}

	// --- Templates ---
	if opts.Scope == "all" || opts.Scope == "templates" {
		templates, err := store.Templates.List()
		if err != nil {
			issues = append(issues, Issue{
				Level:   "error",
				Code:    "TEMPLATE_LIST_ERROR",
				Message: fmt.Sprintf("Failed to list templates: %s", err.Error()),
			})
		} else {
			projectRoot := filepath.Dir(store.Root)
			engine := codegen.NewEngine(projectRoot)
			for _, tmpl := range templates {
				if opts.Entity != "" && opts.Entity != tmpl.Name {
					continue
				}
				issues = append(issues, validateTemplate(tmpl, engine, docPaths, opts)...)
			}
		}
	}

	// Strict mode: upgrade warnings → errors.
	if opts.Strict {
		for i := range issues {
			if issues[i].Level == "warning" {
				issues[i].Level = "error"
			}
		}
	}

	// Count by level.
	r := &Result{Issues: issues}
	for _, iss := range issues {
		switch iss.Level {
		case "error":
			r.ErrorCount++
		case "warning":
			r.WarningCount++
		case "info":
			r.InfoCount++
		}
	}
	r.Valid = r.ErrorCount == 0
	if r.Issues == nil {
		r.Issues = []Issue{}
	}
	return r
}

// ---------- Task validation ----------

func validateTask(t *models.Task, taskIDs, docPaths map[string]bool, parentMap map[string]string, opts Options) []Issue {
	var issues []Issue

	// Title required.
	if t.Title == "" {
		issues = append(issues, Issue{
			Level: "error", Code: "TASK_NO_TITLE",
			Message: "Task has no title", Entity: t.ID,
		})
	}

	// Valid status.
	if t.Status == "" {
		issues = append(issues, Issue{
			Level: "warning", Code: "TASK_NO_STATUS",
			Message: "Task has no status", Entity: t.ID,
		})
	} else if !validStatuses[t.Status] {
		issues = append(issues, Issue{
			Level: "warning", Code: "TASK_INVALID_STATUS",
			Message: fmt.Sprintf("Task has non-standard status: %q", t.Status), Entity: t.ID,
		})
	}

	// Valid priority.
	if t.Priority == "" {
		issues = append(issues, Issue{
			Level: "info", Code: "TASK_NO_PRIORITY",
			Message: "Task has no priority", Entity: t.ID,
		})
	} else if !validPriorities[t.Priority] {
		issues = append(issues, Issue{
			Level: "warning", Code: "TASK_INVALID_PRIORITY",
			Message: fmt.Sprintf("Task has invalid priority: %q", t.Priority), Entity: t.ID,
		})
	}

	// Parent ref exists.
	if t.Parent != "" && !taskIDs[t.Parent] {
		issues = append(issues, Issue{
			Level: "error", Code: "BROKEN_TASK_REF",
			Message: fmt.Sprintf("Parent task %q not found", t.Parent), Entity: t.ID,
		})
	}

	// Circular parent chain.
	if t.Parent != "" {
		if detectCircularParent(t.ID, parentMap) {
			issues = append(issues, Issue{
				Level: "error", Code: "TASK_CIRCULAR_PARENT",
				Message: "Circular parent chain detected", Entity: t.ID,
			})
		}
	}

	// Spec ref exists.
	if t.Spec != "" && !docPaths[t.Spec] {
		issues = append(issues, Issue{
			Level: "warning", Code: "BROKEN_DOC_REF",
			Message: fmt.Sprintf("Spec doc %q not found", t.Spec), Entity: t.ID,
		})
	}

	// Fulfills without spec.
	if len(t.Fulfills) > 0 && t.Spec == "" {
		issues = append(issues, Issue{
			Level: "warning", Code: "TASK_FULFILLS_NO_SPEC",
			Message: "Task has fulfills but no linked spec", Entity: t.ID,
		})
	}

	// Duplicate labels.
	if len(t.Labels) > 1 {
		seen := make(map[string]bool, len(t.Labels))
		for _, l := range t.Labels {
			if seen[l] {
				issues = append(issues, Issue{
					Level: "info", Code: "TASK_DUPLICATE_LABELS",
					Message: fmt.Sprintf("Duplicate label: %q", l), Entity: t.ID,
				})
				break
			}
			seen[l] = true
		}
	}

	// Done but unchecked AC (applicable to all scopes, not just SDD).
	if t.Status == "done" && len(t.AcceptanceCriteria) > 0 {
		for i, ac := range t.AcceptanceCriteria {
			if !ac.Completed {
				issues = append(issues, Issue{
					Level: "warning", Code: "TASK_DONE_UNCHECKED_AC",
					Message: fmt.Sprintf("Task is done but AC #%d is not checked: %s", i+1, ac.Text),
					Entity: t.ID,
				})
			}
		}
	}

	// Inline refs in description, plan, notes.
	checkText := t.Description + " " + t.ImplementationPlan + " " + t.ImplementationNotes
	for _, match := range taskRefRE.FindAllStringSubmatch(checkText, -1) {
		refID := match[1]
		if !taskIDs[refID] {
			issues = append(issues, Issue{
				Level: "warning", Code: "BROKEN_TASK_REF",
				Message: fmt.Sprintf("Referenced task @task-%s not found", refID), Entity: t.ID,
			})
		}
	}
	for _, match := range docRefRE.FindAllStringSubmatch(checkText, -1) {
		refPath := strings.TrimRight(match[1], ".,;)")
		if !docPaths[refPath] {
			issues = append(issues, Issue{
				Level: "warning", Code: "BROKEN_DOC_REF",
				Message: fmt.Sprintf("Referenced doc @doc/%s not found", refPath), Entity: t.ID,
			})
		}
	}

	// SDD-specific checks.
	if opts.Scope == "sdd" {
		if t.Spec != "" && len(t.AcceptanceCriteria) == 0 {
			issues = append(issues, Issue{
				Level: "warning", Code: "SDD_NO_AC",
				Message: "Task is linked to a spec but has no acceptance criteria", Entity: t.ID,
			})
		}
	}

	return issues
}

// detectCircularParent walks the parent chain and returns true if a cycle is found.
func detectCircularParent(id string, parentMap map[string]string) bool {
	visited := map[string]bool{id: true}
	cur := parentMap[id]
	for cur != "" {
		if visited[cur] {
			return true
		}
		visited[cur] = true
		cur = parentMap[cur]
	}
	return false
}

// ---------- Doc validation ----------

func validateDoc(d *models.Doc, taskIDs, docPaths map[string]bool) []Issue {
	var issues []Issue

	if d.Title == "" {
		issues = append(issues, Issue{
			Level: "warning", Code: "DOC_NO_TITLE",
			Message: "Doc has no title", Entity: d.Path,
		})
	}

	if d.Description == "" {
		issues = append(issues, Issue{
			Level: "info", Code: "DOC_NO_DESCRIPTION",
			Message: "Doc has no description", Entity: d.Path,
		})
	}

	if d.Content == "" {
		issues = append(issues, Issue{
			Level: "info", Code: "DOC_NO_CONTENT",
			Message: "Doc has no content", Entity: d.Path,
		})
	}

	// Inline refs in doc content.
	for _, match := range taskRefRE.FindAllStringSubmatch(d.Content, -1) {
		refID := match[1]
		if !taskIDs[refID] {
			issues = append(issues, Issue{
				Level: "info", Code: "BROKEN_TASK_REF",
				Message: fmt.Sprintf("Referenced task @task-%s not found", refID), Entity: d.Path,
			})
		}
	}
	for _, match := range docRefRE.FindAllStringSubmatch(d.Content, -1) {
		refPath := strings.TrimRight(match[1], ".,;)")
		if !docPaths[refPath] {
			issues = append(issues, Issue{
				Level: "info", Code: "BROKEN_DOC_REF",
				Message: fmt.Sprintf("Referenced doc @doc/%s not found", refPath), Entity: d.Path,
			})
		}
	}

	return issues
}

// ---------- Template validation ----------

func validateTemplate(tmpl *models.Template, engine *codegen.Engine, docPaths map[string]bool, opts Options) []Issue {
	var issues []Issue

	if tmpl.Name == "" {
		issues = append(issues, Issue{
			Level: "error", Code: "TEMPLATE_NO_NAME",
			Message: "Template has no name", Entity: tmpl.Path,
		})
	}

	if len(tmpl.Actions) == 0 {
		issues = append(issues, Issue{
			Level: "warning", Code: "TEMPLATE_NO_ACTIONS",
			Message: "Template has no actions defined", Entity: tmpl.Name,
		})
	}

	// Check each action's template file and path syntax.
	for i, action := range tmpl.Actions {
		if action.Template != "" {
			tplFile := filepath.Join(tmpl.Path, action.Template)
			if _, err := os.Stat(tplFile); os.IsNotExist(err) {
				issues = append(issues, Issue{
					Level: "error", Code: "TEMPLATE_FILE_MISSING",
					Message: fmt.Sprintf("action[%d] template file %q not found", i+1, action.Template),
					Entity: tmpl.Name,
				})
			} else {
				content, err := os.ReadFile(tplFile)
				if err == nil {
					if _, err := engine.ValidateTemplate(string(content)); err != nil {
						issues = append(issues, Issue{
							Level: "error", Code: "TEMPLATE_PARSE_ERROR",
							Message: fmt.Sprintf("action[%d] %q parse error: %s", i+1, action.Template, err),
							Entity: tmpl.Name,
						})
					}
				}
			}
		}
		if action.Path != "" {
			if _, err := engine.ValidateTemplate(action.Path); err != nil {
				issues = append(issues, Issue{
					Level: "error", Code: "TEMPLATE_PATH_ERROR",
					Message: fmt.Sprintf("action[%d] path %q parse error: %s", i+1, action.Path, err),
					Entity: tmpl.Name,
				})
			}
		}
	}

	// Linked doc ref.
	if tmpl.Doc != "" && !docPaths[tmpl.Doc] {
		severity := "warning"
		if opts.Fix {
			severity = "info"
		}
		issues = append(issues, Issue{
			Level: severity, Code: "BROKEN_DOC_REF",
			Message: fmt.Sprintf("Linked doc %q not found", tmpl.Doc), Entity: tmpl.Name,
		})
	}

	return issues
}

// LooksLikeTaskID returns true if s looks like a task ID rather than a doc path.
func LooksLikeTaskID(s string) bool {
	if len(s) > 20 {
		return false
	}
	if strings.Contains(s, "/") {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-') {
			return false
		}
	}
	return true
}
