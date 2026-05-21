package handlers

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/server"
	"strings"
	"time"

	"github.com/howznguyen/knowns/internal/lsp"
	"github.com/howznguyen/knowns/internal/readiness"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
)

func RegisterInitialTool(s *server.MCPServer, getStore func() *storage.Store, getLSPManager ...func() *lsp.Manager) {
	s.AddTool(
		mcp.NewTool("initial",
			mcp.WithDescription(`Provides the Knowns session-ready instructions for AI agents.

- initial: Return dynamic project state, required code-intelligence rules, workflow guidance, and tool summary. Required: none. Optional: none. Returns: plain-text instructions to read before performing project work.
`),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var manager *lsp.Manager
			if len(getLSPManager) > 0 && getLSPManager[0] != nil {
				manager = getLSPManager[0]()
			}
			return mcp.NewToolResultText(buildInitialInstructions(getStore, manager)), nil
		},
	)
}

func buildInitialInstructions(getStore func() *storage.Store, manager *lsp.Manager) string {
	var b strings.Builder
	store := getStore()

	b.WriteString("# Knowns MCP — Session Ready\n\n")
	writeProjectState(&b, store, manager)
	b.WriteString("\n")
	writeCodeIntelligenceRules(&b)
	b.WriteString("\n")
	writeWorkflow(&b)
	b.WriteString("\n")
	writeToolsSummary(&b)

	return b.String()
}

func writeProjectState(b *strings.Builder, store *storage.Store, manager *lsp.Manager) {
	b.WriteString("## Project State\n")
	if store == nil {
		b.WriteString("Project: not connected\n")
		return
	}

	payload := readiness.BuildReadiness(store, readiness.Options{})
	inProgress := countInProgressTasks(store)
	b.WriteString(fmt.Sprintf("Project: %s\n", payload.ProjectName))
	if payload.Knowledge != nil {
		k := payload.Knowledge
		b.WriteString(fmt.Sprintf("Knowledge: docs: %d | tasks: %d (%d in-progress) | templates: %d | memories: %dp, %dg\n",
			k.Docs, k.Tasks, inProgress, k.Templates, k.Memories.Project, k.Memories.Global))
	}

	if timerLine := activeTimerLine(store); timerLine != "" {
		b.WriteString(timerLine)
		b.WriteString("\n")
	}
	if lspLine := lspWarningsLine(manager); lspLine != "" {
		b.WriteString(lspLine)
		b.WriteString("\n")
	}

	symbols, relations := codeIndexCounts(store)
	b.WriteString(fmt.Sprintf("Code index: symbols: %d | relations: %d\n", symbols, relations))
}

func countInProgressTasks(store *storage.Store) int {
	tasks, err := store.Tasks.List()
	if err != nil {
		return 0
	}
	count := 0
	for _, task := range tasks {
		if task.Status == "in-progress" {
			count++
		}
	}
	return count
}

func activeTimerLine(store *storage.Store) string {
	state, err := store.Time.GetState()
	if err != nil || len(state.Active) == 0 {
		return ""
	}
	timer := state.Active[0]
	startedAt, err := time.Parse(time.RFC3339Nano, timer.StartedAt)
	if err != nil {
		return fmt.Sprintf("⏱ Active timer: %s \"%s\"", timer.TaskID, timer.TaskTitle)
	}
	elapsed := time.Since(startedAt) - time.Duration(timer.TotalPausedMs)*time.Millisecond
	if timer.PausedAt != nil {
		if pausedAt, err := time.Parse(time.RFC3339Nano, *timer.PausedAt); err == nil {
			elapsed = pausedAt.Sub(startedAt) - time.Duration(timer.TotalPausedMs)*time.Millisecond
		}
	}
	if elapsed < 0 {
		elapsed = 0
	}
	return fmt.Sprintf("⏱ Active timer: %s \"%s\" (%s)", timer.TaskID, timer.TaskTitle, formatInitialDuration(elapsed))
}

func formatInitialDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	if d < time.Minute {
		return "0m"
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func lspWarningsLine(manager *lsp.Manager) string {
	if manager == nil {
		return ""
	}
	missing := manager.MissingServers()
	if len(missing) == 0 {
		return ""
	}
	parts := make([]string, 0, len(missing))
	for _, server := range missing {
		binary := server.BinaryName
		if binary == "" {
			binary = server.Name
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", server.LanguageID, binary))
	}
	return "⚠ Missing LSP servers: " + strings.Join(parts, ", ")
}

func codeIndexCounts(store *storage.Store) (symbols int, relations int) {
	if !store.CodeRefIndexExists() {
		return 0, 0
	}
	db := store.SemanticDB()
	if db == nil {
		return 0, 0
	}
	defer db.Close()
	_ = db.QueryRow("SELECT COUNT(*) FROM code_symbols").Scan(&symbols)
	_ = db.QueryRow("SELECT COUNT(*) FROM code_edges").Scan(&relations)
	return symbols, relations
}

func writeCodeIntelligenceRules(b *strings.Builder) {
	b.WriteString(`## Code Intelligence Rules
**CRITICAL**: You MUST use Knowns code tools for ALL code-related operations. This is NOT optional.

Knowns code tools provide capabilities that built-in tools CANNOT:
- symbols(path): structured symbol tree — not raw text
- definition(pos): jump to definition via LSP — not grep
- references(pos): all usages across workspace — not grep missing dynamic refs
- find(query): semantic symbol search with body — not reading whole files
- rename/replace/insert/delete: safe structural edits — not blind find-and-replace

**FORBIDDEN** on code files:
- Read for discovery or understanding structure
- Grep/find for locating symbols or patterns
- Built-in Edit for renaming or replacing → use code(rename) or code(replace)

**ONLY** exception — built-in Read:
- Non-code files (config, markdown, docs)
- After code tools located target, AND you need surrounding context
`)
}

func writeWorkflow(b *strings.Builder) {
	b.WriteString(`## Workflow
Discovery:     search(query) → docs/tasks(get) for details
Code context:  code(find) → code(symbols) → code(references)
Task flow:     tasks(get) → follow refs → plan → implement → validate → done
Time:          time(start) when taking task, time(stop) when done
Progress:      tasks(update, appendNotes:"...") — not notes (replaces)

For detailed usage of any tool: help("tool.action") or help("tool.*")
`)
}

func writeToolsSummary(b *strings.Builder) {
	b.WriteString("## Tools (use help for details)\n")
	b.WriteString("code (7) | tasks (7) | docs (6) | search (3) | time (4) | templates (4) | validate (1) | memory (7) | project (4) | help (1)")
}
