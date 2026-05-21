package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/howznguyen/knowns/internal/runtimequeue"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Index docs, tasks, and memories for semantic search",
	Long: `Index docs, tasks, and memories in the project for semantic search.

Code files are not indexed by ingest. Real-time code intelligence uses LSP
queries through the MCP code tool.

Examples:
  knowns ingest              # Index docs, tasks, and memories
  knowns ingest --background # Queue ingest in the runtime daemon`,
	RunE: runIngest,
}

var ingestDryRun bool
var ingestIncludeTests bool
var ingestBackground bool
var ingestFull bool

func registerIngestFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&ingestDryRun, "dry-run", false, "Print what would be indexed without writing to disk")
	cmd.Flags().BoolVar(&ingestIncludeTests, "include-tests", false, "Include test files in indexing")
	cmd.Flags().BoolVar(&ingestBackground, "background", false, "Enqueue to shared runtime daemon and return; follow with `knowns runtime ps --watch`")
	cmd.Flags().BoolVar(&ingestFull, "full", false, "Force full re-ingest, ignoring cached hashes")
}

func init() {
	registerIngestFlags(ingestCmd)
}

// ─── ingest progress model (bubbletea) ───────────────────────────────

type ingestPhase struct {
	name    string
	total   int
	current int
}

type ingestState struct {
	phase     string
	processed int
	total     int
	done      bool
	err       error
}

type ingestDoneMsg struct {
	err       error
	symCount  int
	fileCount int
	edgeCount int
}

type ingestModel struct {
	bar             progress.Model
	state           *ingestState
	quit            bool
	startTime       time.Time
	phaseStartTime  time.Time
	prog            *tea.Program
	lastPhase       string
	lastTotal       int
	completedPhases []struct {
		name  string
		count int
	}
	symCount  int
	fileCount int
	edgeCount int
}

func ingestTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return reindexTickMsg{}
	})
}

func (m *ingestModel) Init() tea.Cmd {
	return ingestTickCmd()
}

func (m *ingestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.quit = true
			return m, tea.Quit
		}
	case reindexTickMsg:
		if m.quit {
			return m, nil
		}
		if m.state.phase != m.lastPhase && m.lastPhase != "" {
			m.completedPhases = append(m.completedPhases, struct {
				name  string
				count int
			}{
				name:  m.lastPhase,
				count: m.lastTotal,
			})
			m.phaseStartTime = time.Now()
		}
		m.lastPhase = m.state.phase
		m.lastTotal = m.state.total

		pct := 0.0
		if m.state.total > 0 {
			pct = float64(m.state.processed) / float64(m.state.total)
		}
		cmd := m.bar.SetPercent(pct)
		return m, tea.Batch(cmd, ingestTickCmd())
	case ingestDoneMsg:
		if m.lastPhase != "" {
			m.completedPhases = append(m.completedPhases, struct {
				name  string
				count int
			}{
				name:  m.lastPhase,
				count: m.lastTotal,
			})
		}
		m.state.done = true
		m.state.err = msg.err
		m.symCount = msg.symCount
		m.fileCount = msg.fileCount
		m.edgeCount = msg.edgeCount
		m.quit = true
		cmd := m.bar.SetPercent(1.0)
		return m, tea.Batch(cmd, tea.Quit)
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *ingestModel) View() tea.View {
	var b strings.Builder

	if m.quit {
		for _, cp := range m.completedPhases {
			b.WriteString(fmt.Sprintf("  %s Indexed %s (%d)\n",
				searchSuccessStyle.Render("✓"), cp.name, cp.count))
		}
		return tea.NewView(b.String())
	}

	for _, cp := range m.completedPhases {
		b.WriteString(fmt.Sprintf("  %s Indexed %s (%d)\n",
			searchSuccessStyle.Render("✓"), cp.name, cp.count))
	}

	processed := m.state.processed
	total := m.state.total
	phase := m.state.phase

	elapsed := time.Since(m.phaseStartTime)
	eta := ""
	if elapsed.Seconds() > 0.5 && processed > 0 && processed < total {
		itemsPerSec := float64(processed) / elapsed.Seconds()
		remaining := float64(total-processed) / itemsPerSec
		if remaining >= 1 {
			eta = fmt.Sprintf("  %s remaining", formatDuration(int(remaining)))
		}
	}

	info := fmt.Sprintf("Indexing %s (%d/%d)%s", phase, processed, total, eta)
	b.WriteString(fmt.Sprintf("  %s  %s\n", m.bar.View(), searchDimStyle.Render(info)))
	return tea.NewView(b.String())
}

// runIngestWithProgress runs the ingest with a bubbletea progress UI.
// When forceFull is false, it uses two-tier delta detection:
//   - Tier 1: file hash (SHA-256) to skip parsing unchanged files
//   - Tier 2: chunk content hash to skip re-storage unchanged symbols
func runIngestWithProgress(projectRoot string, includeTests bool, forceFull bool) (symCount, fileCount, edgeCount int, err error) {
	store := getStore()
	embedder, vecStore, err := search.InitSemantic(store)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("init semantic index: %w", err)
	}
	defer embedder.Close()
	defer vecStore.Close()

	engine := search.NewEngine(store, embedder, vecStore)
	if err := engine.Reindex(nil); err != nil {
		return 0, 0, 0, fmt.Errorf("index knowledge: %w", err)
	}
	return 0, 0, 0, nil
}

func runIngest(cmd *cobra.Command, args []string) error {
	knDir, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	if _, err := os.Stat(knDir); err != nil {
		return fmt.Errorf("not a knowns project (no .knowns/ directory): %w", err)
	}

	store := storage.NewStore(knDir)

	semanticEnabled, err := isSemanticSearchEnabled(store)
	if err != nil {
		return fmt.Errorf("check semantic search: %w", err)
	}
	if !semanticEnabled {
		return fmt.Errorf("semantic search is not enabled. Run 'knowns model set' to configure an embedding model")
	}

	projectRoot := filepath.Dir(knDir)

	if ingestDryRun {
		fmt.Println("Would index docs, tasks, and memories. Code files are not indexed by ingest.")
		return nil
	}

	// Default: in-process so the user sees the rich bubbletea progress bar.
	// `--background` routes through the shared runtime daemon and returns
	// immediately so the user can follow with `knowns runtime ps --watch`.
	if !ingestBackground || ingestIncludeTests || runtimequeue.ShouldBypassDaemon() {
		_, _, _, err := runIngestWithProgress(projectRoot, ingestIncludeTests, ingestFull)
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println(searchSuccessStyle.Render("✓ Indexed docs, tasks, and memories"))
		return nil
	}

	job, err := runtimequeue.Enqueue(store.Root, runtimequeue.JobReindex, projectRoot)
	if err != nil {
		return fmt.Errorf("enqueue ingest job: %w", err)
	}
	fmt.Printf("  %s queued ingest job %s — follow with: %s\n",
		searchDimStyle.Render("·"),
		job.ID,
		searchDimStyle.Render("knowns runtime ps --watch"))
	return nil
}

func countUniqueFiles(symbols []search.CodeSymbol) int {
	files := make(map[string]bool)
	for _, s := range symbols {
		files[s.DocPath] = true
	}
	return len(files)
}

func isSemanticSearchEnabled(store *storage.Store) (bool, error) {
	cfg, err := store.Config.Load()
	if err != nil {
		return false, err
	}
	if cfg == nil || cfg.Settings.SemanticSearch == nil {
		return false, nil
	}
	return cfg.Settings.SemanticSearch.Enabled, nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return storage.FindProjectRoot(dir)
}
