package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"

	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Index code files using AST-based code intelligence",
	Long: `Index all code files in the project using tree-sitter AST parsing.

This command walks the project directory, parses AST for Go, TypeScript,
JavaScript, and Python files, and stores code symbols as indexed code data.

Files matching .gitignore and test files (*_test.go, *.spec.ts, etc.)
are skipped by default. Use --include-tests to include them.

Examples:
  knowns code ingest              # Index all code files
  knowns code ingest --dry-run    # Preview what would be indexed
  knowns code ingest --include-tests  # Include test files`,
	RunE: runIngest,
}

var ingestDryRun bool
var ingestIncludeTests bool

func registerIngestFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&ingestDryRun, "dry-run", false, "Print what would be indexed without writing to disk")
	cmd.Flags().BoolVar(&ingestIncludeTests, "include-tests", false, "Include test files in indexing")
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
	phase    string
	processed int
	total    int
	done     bool
	err      error
}

type ingestDoneMsg struct {
	err        error
	symCount   int
	fileCount  int
	edgeCount  int
}

type ingestModel struct {
	bar            progress.Model
	state          *ingestState
	quit           bool
	startTime      time.Time
	phaseStartTime time.Time
	prog           *tea.Program
	lastPhase      string
	lastTotal      int
	completedPhases []struct{ name string; count int }
	symCount       int
	fileCount      int
	edgeCount      int
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
			m.completedPhases = append(m.completedPhases, struct{ name string; count int }{
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
			m.completedPhases = append(m.completedPhases, struct{ name string; count int }{
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
	b.WriteString(fmt.Sprintf("  %s%s\n", m.bar.View(), searchDimStyle.Render(info)))
	return tea.NewView(b.String())
}

// runIngestWithProgress runs the ingest with a bubbletea progress UI.
func runIngestWithProgress(projectRoot string, includeTests bool) (symCount, fileCount, edgeCount int, err error) {
	state := &ingestState{phase: "parsing files", total: 1}

	bar := NewBrandProgressBar()
	m := &ingestModel{
		bar:            bar,
		state:          state,
		startTime:      time.Now(),
		phaseStartTime: time.Now(),
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin))
	m.prog = p

	go func() {
		syms, edges, err := search.IndexAllFiles(projectRoot, includeTests)
		if err != nil {
			p.Send(ingestDoneMsg{err: err})
			return
		}

		fileCount = countUniqueFiles(syms)

		// Phase 2: embed chunks
		p.Send(reindexTickMsg{}) // trigger phase label update
		state.phase = "embedding"
		state.total = len(syms)
		state.processed = 0

		store := getStore()
		embedder, vecStore, err := search.InitSemantic(store)
		if err != nil {
			p.Send(ingestDoneMsg{err: fmt.Errorf("init semantic search: %w", err)})
			return
		}
		defer embedder.Close()
		defer vecStore.Close()

		vecStore.RemoveByPrefix("code::")

		var chunks []search.Chunk
		for i, sym := range syms {
			chunk := sym.ToChunk()
			vec, err := embedder.EmbedDocument(chunk.Content)
			if err != nil {
				continue
			}
			chunk.Embedding = vec
			chunks = append(chunks, chunk)
			state.processed = i + 1
		}

		vecStore.AddChunks(chunks)
		if err := vecStore.Save(); err != nil {
			p.Send(ingestDoneMsg{err: fmt.Errorf("save index: %w", err)})
			return
		}

		// Save code edges
		db := store.SemanticDB()
		if db != nil && len(edges) > 0 {
			resolvedEdges := search.ResolveCodeEdges(syms, edges)
			if dbErr := search.SaveCodeEdges(db, resolvedEdges); dbErr == nil {
				edgeCount = len(resolvedEdges)
			}
		}

		p.Send(ingestDoneMsg{
			err:       nil,
			symCount:  len(chunks),
			fileCount: fileCount,
			edgeCount: edgeCount,
		})
	}()

	if _, runErr := p.Run(); runErr != nil {
		return 0, 0, 0, runErr
	}
	if m.state.err != nil {
		return 0, 0, 0, m.state.err
	}
	return m.symCount, m.fileCount, m.edgeCount, nil
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
		syms, _, err := search.IndexAllFiles(projectRoot, ingestIncludeTests)
		if err != nil {
			return fmt.Errorf("index files: %w", err)
		}
		fmt.Printf("Would index %d symbols:\n\n", len(syms))
		for _, sym := range syms {
			fmt.Printf("  [%s] %s — %s\n", sym.Kind, sym.Name, sym.DocPath)
		}
		return nil
	}

	symCount, fileCount, edgeCount, err := runIngestWithProgress(projectRoot, ingestIncludeTests)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(searchSuccessStyle.Render(fmt.Sprintf(
		"✓ Indexed %d symbols (%d files, %d edges)", symCount, fileCount, edgeCount)))
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
