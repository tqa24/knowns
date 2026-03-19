package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tasks and documentation",
	Args:  cobra.ArbitraryArgs,
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	reindex, _ := cmd.Flags().GetBool("reindex")
	setup, _ := cmd.Flags().GetBool("setup")
	statusCheck, _ := cmd.Flags().GetBool("status-check")
	installRuntime, _ := cmd.Flags().GetBool("install-runtime")

	if installRuntime {
		return runInstallRuntime()
	}

	if statusCheck {
		return runStatusCheck()
	}

	if setup {
		return runSetup()
	}

	if reindex {
		return runReindex()
	}

	if len(args) == 0 {
		return fmt.Errorf("search query required (use --help for usage)")
	}

	query := strings.Join(args, " ")
	store := getStore()

	typeFilter, _ := cmd.Flags().GetString("type")
	statusFilter, _ := cmd.Flags().GetString("status")
	priorityFilter, _ := cmd.Flags().GetString("priority")
	labelFilter, _ := cmd.Flags().GetString("label")
	tagFilter, _ := cmd.Flags().GetString("tag")
	assigneeFilter, _ := cmd.Flags().GetString("assignee")
	keywordOnly, _ := cmd.Flags().GetBool("keyword")

	plain := isPlain(cmd)
	jsonOut := isJSON(cmd)

	// Determine search mode.
	mode := "hybrid"
	if keywordOnly {
		mode = "keyword"
	}

	// Try to create embedder for semantic search.
	embedder, vecStore, _ := initSemanticSearchReal()

	opts := search.SearchOptions{
		Query:    query,
		Type:     typeFilter,
		Mode:     mode,
		Status:   statusFilter,
		Priority: priorityFilter,
		Assignee: assigneeFilter,
		Label:    labelFilter,
		Tag:      tagFilter,
		Limit:    20,
	}

	engine := search.NewEngine(store, embedder, vecStore)
	results, err := engine.Search(opts)
	if err != nil {
		return err
	}

	if jsonOut {
		printJSON(results)
		return nil
	}

	if len(results) == 0 {
		if plain {
			fmt.Println("No results found.")
		} else {
			fmt.Printf("No results for %q\n", query)
		}
		return nil
	}

	// Detect actual mode used.
	actualMode := "keyword"
	if engine.SemanticAvailable() && !keywordOnly {
		actualMode = "hybrid"
	}

	// Normalize scores for percentage display.
	maxScore := 0.0
	for _, r := range results {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}

	// Split into tasks and docs.
	var taskResults, docResults []models.SearchResult
	for _, r := range results {
		if r.Type == "task" {
			taskResults = append(taskResults, r)
		} else {
			docResults = append(docResults, r)
		}
	}

	if plain {
		content := sprintPlainResults(taskResults, docResults, maxScore)
		printPaged(cmd, content)
	} else {
		content := renderPrettyResults(query, actualMode, results, taskResults, docResults, maxScore)
		renderOrPage(cmd, fmt.Sprintf("Search: %s", query), content)
	}
	return nil
}

func sprintPlainResults(taskResults, docResults []models.SearchResult, maxScore float64) string {
	var b strings.Builder
	if len(taskResults) > 0 {
		fmt.Fprintln(&b, "Tasks:")
		for _, r := range taskResults {
			pct := scoreToPercent(r.Score, maxScore)
			fmt.Fprintf(&b, "  #%s [%s] [%s] (%d%%)\n", r.ID, r.Status, r.Priority, pct)
			if r.Snippet != "" {
				snip := truncate(r.Snippet, 100)
				fmt.Fprintf(&b, "    %s\n", snip)
			}
			fmt.Fprintf(&b, "    Matched by: %s\n", formatMatchedBy(r.MatchedBy))
		}
		fmt.Fprintln(&b)
	}
	if len(docResults) > 0 {
		fmt.Fprintln(&b, "Docs:")
		for _, r := range docResults {
			pct := scoreToPercent(r.Score, maxScore)
			fmt.Fprintf(&b, "  %s (%d%%)\n", r.Path, pct)
			if r.Snippet != "" {
				snip := truncate(r.Snippet, 100)
				fmt.Fprintf(&b, "    %s\n", snip)
			}
			fmt.Fprintf(&b, "    Matched by: %s\n", formatMatchedBy(r.MatchedBy))
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func renderPrettyResults(query, mode string, results []models.SearchResult, taskResults, docResults []models.SearchResult, maxScore float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s\n\n",
		StyleBold.Render(fmt.Sprintf("Found %d result(s)", len(results))),
		StyleDim.Render("for"),
		StyleInfo.Render(fmt.Sprintf("%q", query))+" "+StyleDim.Render(fmt.Sprintf("(%s mode)", mode)))

	if len(taskResults) > 0 {
		fmt.Fprintln(&b, RenderSectionHeader("Tasks"))
		fmt.Fprintln(&b)
		for _, r := range taskResults {
			pct := scoreToPercent(r.Score, maxScore)
			fmt.Fprintf(&b, "  %s %s %s  %s %s %s\n",
				RenderBadge("TASK", colorBlue),
				StyleID.Render("#"+r.ID),
				StyleBold.Render("— "+r.Title),
				RenderStatusBadge(r.Status),
				RenderPriorityBadge(r.Priority),
				StyleDim.Render(fmt.Sprintf("(%d%%)", pct)))
			if r.Snippet != "" {
				snip := truncate(r.Snippet, 100)
				fmt.Fprintf(&b, "    %s\n", StyleDim.Render(snip))
			}
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render("Matched by: "+formatMatchedBy(r.MatchedBy)))
			fmt.Fprintln(&b)
		}
	}

	if len(docResults) > 0 {
		fmt.Fprintln(&b, RenderSectionHeader("Docs"))
		fmt.Fprintln(&b)
		for _, r := range docResults {
			pct := scoreToPercent(r.Score, maxScore)
			tags := ""
			if len(r.Tags) > 0 {
				tags = " " + RenderTags(r.Tags)
			}
			fmt.Fprintf(&b, "  %s %s %s%s %s\n",
				RenderBadge("DOC", colorMagenta),
				StyleID.Render(r.Path),
				StyleBold.Render("— "+r.Title),
				tags,
				StyleDim.Render(fmt.Sprintf("(%d%%)", pct)))
			if r.Snippet != "" {
				snip := truncate(r.Snippet, 100)
				fmt.Fprintf(&b, "    %s\n", StyleDim.Render(snip))
			}
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render("Matched by: "+formatMatchedBy(r.MatchedBy)))
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func formatMatchedBy(methods []string) string {
	if len(methods) == 0 {
		return "keyword"
	}
	return strings.Join(methods, " + ")
}

// ─── semantic search initialization ──────────────────────────────────

// initSemanticSearchReal attempts to create an embedder and vector store.
// Returns (nil, nil) if semantic search is not available (silent fallback).
func initSemanticSearchReal() (*search.Embedder, search.VectorStore, error) {
	store := getStore()
	embedder, vecStore, err := search.InitSemantic(store)
	return embedder, vecStore, err
}

// ─── status check ────────────────────────────────────────────────────

func runStatusCheck() error {
	store := getStore()
	cfg, _ := store.Config.Load()

	onnxAvail, onnxPath := search.IsONNXAvailable()

	fmt.Println()
	fmt.Println(StyleBold.Render("Semantic Search Status"))
	fmt.Println(strings.Repeat("─", 40))

	// ONNX Runtime.
	if onnxAvail {
		fmt.Println(searchSuccessStyle.Render("  ONNX Runtime: installed"))
		fmt.Println(searchDimStyle.Render(fmt.Sprintf("    Path: %s", onnxPath)))
	} else {
		fmt.Println(searchWarnStyle.Render("  ONNX Runtime: not found"))
		fmt.Println(searchDimStyle.Render("    Install: knowns search --install-runtime"))
	}

	// Model.
	if cfg != nil && cfg.Settings.SemanticSearch != nil {
		ss := cfg.Settings.SemanticSearch
		fmt.Printf("  Model: %s\n", ss.Model)
		fmt.Printf("  Enabled: %v\n", ss.Enabled)
		fmt.Printf("  Dimensions: %d\n", ss.Dimensions)
		fmt.Printf("  Max Tokens: %d\n", ss.MaxTokens)
	} else {
		fmt.Println("  Model: not configured")
		fmt.Println(searchDimStyle.Render("    Set up: knowns search --setup"))
	}

	// Vector index.
	searchDir := filepath.Join(store.Root, ".search")
	vs := search.NewSQLiteVectorStore(searchDir, "", 0)
	count, model, indexedAt := vs.Stats()
	if count > 0 {
		fmt.Printf("  Index: %d chunks (model: %s)\n", count, model)
		fmt.Printf("  Indexed at: %s\n", indexedAt.Format(time.RFC3339))
	} else {
		fmt.Println("  Index: empty")
		fmt.Println(searchDimStyle.Render("    Build: knowns search --reindex"))
	}

	// Overall status.
	fmt.Println()
	if onnxAvail && cfg != nil && cfg.Settings.SemanticSearch != nil && cfg.Settings.SemanticSearch.Enabled && count > 0 {
		fmt.Println(searchSuccessStyle.Render("  Status: ready (hybrid search active)"))
	} else if onnxAvail && cfg != nil && cfg.Settings.SemanticSearch != nil && cfg.Settings.SemanticSearch.Enabled {
		fmt.Println(searchWarnStyle.Render("  Status: needs reindex (run: knowns search --reindex)"))
	} else {
		fmt.Println(searchDimStyle.Render("  Status: keyword-only mode"))
	}
	fmt.Println()

	return nil
}

// ─── setup ───────────────────────────────────────────────────────────

func runSetup() error {
	store := getStore()
	cfg, err := store.Config.Load()
	if err != nil {
		return err
	}

	// Check ONNX Runtime.
	onnxAvail, _ := search.IsONNXAvailable()
	if !onnxAvail {
		fmt.Println(searchWarnStyle.Render("ONNX Runtime not found."))
		fmt.Println()
		fmt.Println("Install ONNX Runtime first:")
		fmt.Println("  knowns search --install-runtime")
		fmt.Println()
		return nil
	}

	// Check if a model is set.
	if cfg.Settings.SemanticSearch == nil || cfg.Settings.SemanticSearch.Model == "" {
		fmt.Println(searchWarnStyle.Render("No embedding model configured."))
		fmt.Println()
		fmt.Println("Set a model first:")
		fmt.Println("  knowns model set gte-small")
		fmt.Println()
		return nil
	}

	ss := cfg.Settings.SemanticSearch
	if !ss.Enabled {
		ss.Enabled = true
		_ = store.Config.Set("settings.semanticSearch.enabled", true)
		fmt.Println(searchSuccessStyle.Render("Semantic search enabled."))
	} else {
		fmt.Println("Semantic search is already enabled.")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Build the index: knowns search --reindex")
	fmt.Println("  2. Search: knowns search \"your query\"")
	fmt.Println()

	return nil
}

// ─── install runtime ─────────────────────────────────────────────────

func runInstallRuntime() error {
	avail, path := search.IsONNXAvailable()
	if avail {
		fmt.Println(searchSuccessStyle.Render("ONNX Runtime is already installed."))
		fmt.Println(searchDimStyle.Render(fmt.Sprintf("  Path: %s", path)))
		return nil
	}

	url, libName, err := search.ONNXRuntimeDownloadURL()
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	destDir := filepath.Join(home, ".knowns", "lib")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	fmt.Printf("Downloading ONNX Runtime for %s/%s...\n\n", runtime.GOOS, runtime.GOARCH)

	// Download to temp file.
	tmpFile, err := os.CreateTemp("", "onnxruntime-*.tgz")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, dlErr := downloadWithProgress("ONNX Runtime", url, tmpPath)
	if dlErr != nil {
		tmpFile.Close()
		// Fallback.
		_, dlErr = downloadSimple(url, tmpPath)
	} else {
		tmpFile.Close()
	}
	if dlErr != nil {
		return fmt.Errorf("download failed: %w", dlErr)
	}
	fmt.Println()

	// Extract the library from the archive.
	destPath := filepath.Join(destDir, libName)
	if err := extractONNXLib(tmpPath, libName, destPath); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	fmt.Println(searchSuccessStyle.Render("ONNX Runtime installed successfully."))
	fmt.Println(searchDimStyle.Render(fmt.Sprintf("  Path: %s", destPath)))
	fmt.Println()
	fmt.Println("Next: knowns search --setup")
	return nil
}

// extractONNXLib extracts the shared library from an ONNX Runtime tgz archive.
func extractONNXLib(archivePath, libName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the library file in the archive.
		base := filepath.Base(hdr.Name)
		if base == libName && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			// Make executable on unix.
			os.Chmod(destPath, 0755)
			return nil
		}
	}

	return fmt.Errorf("%s not found in archive", libName)
}

// ─── reindex with bubbletea progress ─────────────────────────────────

var (
	searchSuccessStyle = StyleSuccess
	searchWarnStyle    = StyleWarning
	searchDimStyle     = StyleDim
)

// reindexTickMsg polls shared state from the background goroutine.
type reindexTickMsg struct{}

// reindexDoneMsg signals the reindex finished.
type reindexDoneMsg struct {
	err        error
	chunkCount int
}

// reindexState is shared between the bubbletea model and the reindex goroutine.
type reindexState struct {
	phase     string
	processed int
	total     int
	done      bool
	err       error
}

type reindexModel struct {
	bar       progress.Model
	state     *reindexState
	quit      bool
	startTime time.Time
	prog      *tea.Program
}

func reindexTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return reindexTickMsg{}
	})
}

func (m *reindexModel) Init() tea.Cmd {
	return reindexTickCmd()
}

func (m *reindexModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		pct := 0.0
		if m.state.total > 0 {
			pct = float64(m.state.processed) / float64(m.state.total)
		}
		cmd := m.bar.SetPercent(pct)
		return m, tea.Batch(cmd, reindexTickCmd())
	case reindexDoneMsg:
		m.state.done = true
		m.state.err = msg.err
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

func (m *reindexModel) View() tea.View {
	if m.quit {
		return tea.NewView("")
	}
	pct := 0.0
	processed := m.state.processed
	total := m.state.total
	phase := m.state.phase
	if total > 0 {
		pct = float64(processed) / float64(total)
	}

	elapsed := time.Since(m.startTime)
	eta := ""
	if elapsed.Seconds() > 0.5 && processed > 0 && processed < total {
		itemsPerSec := float64(processed) / elapsed.Seconds()
		remaining := float64(total-processed) / itemsPerSec
		if remaining >= 1 {
			eta = fmt.Sprintf("  %s remaining", formatDuration(int(remaining)))
		}
	}

	info := fmt.Sprintf(" %3.0f%%  Indexing %s (%d/%d)%s",
		pct*100, phase, processed, total, eta)
	return tea.NewView(fmt.Sprintf("  %s%s\n", m.bar.View(), searchDimStyle.Render(info)))
}

func runReindex() error {
	store := getStore()

	tasks, _ := store.Tasks.List()
	docs, _ := store.Docs.List()
	taskCount := len(tasks)
	docCount := len(docs)
	total := taskCount + docCount

	if total == 0 {
		fmt.Println("No tasks or docs to index.")
		return nil
	}

	// Auto-install ONNX Runtime if needed.
	if err := ensureONNXRuntime(); err != nil {
		fmt.Println(searchWarnStyle.Render(fmt.Sprintf("  Warning: ONNX Runtime install failed: %s", err)))
		fmt.Println(searchDimStyle.Render("  Falling back to keyword-only search."))
		fmt.Println()
	}

	// Auto-download default model if ONNX is available but no model configured.
	if avail, _ := search.IsONNXAvailable(); avail {
		cfg, _ := store.Config.Load()
		if cfg != nil && (cfg.Settings.SemanticSearch == nil || cfg.Settings.SemanticSearch.Model == "") {
			defaultModel := "gte-small"
			fmt.Println(searchDimStyle.Render(fmt.Sprintf("  No model configured — downloading default model (%s)...", defaultModel)))
			fmt.Println()
			if err := runSemanticSetup(defaultModel); err != nil {
				fmt.Println(searchWarnStyle.Render(fmt.Sprintf("  Warning: model download failed: %s", err)))
				fmt.Println()
			} else {
				// Auto-configure the model.
				for i := range supportedModels {
					if supportedModels[i].ID == defaultModel {
						m := &supportedModels[i]
						_ = store.Config.Set("settings.semanticSearch.model", m.ID)
						_ = store.Config.Set("settings.semanticSearch.huggingFaceId", m.HuggingFace)
						_ = store.Config.Set("settings.semanticSearch.dimensions", m.Dimensions)
						_ = store.Config.Set("settings.semanticSearch.maxTokens", m.MaxTokens)
						_ = store.Config.Set("settings.semanticSearch.enabled", true)
						fmt.Println(searchSuccessStyle.Render(fmt.Sprintf("✓ Configured %s as default model", defaultModel)))
						fmt.Println()
						break
					}
				}
			}
		}
	}

	// Try semantic reindex.
	embedder, vecStore, semanticErr := initSemanticSearchReal()
	if embedder != nil && vecStore != nil {
		defer embedder.Close()

		fmt.Printf("Rebuilding semantic search index (%d tasks, %d docs)...\n\n", taskCount, docCount)

		startTime := time.Now()
		state := &reindexState{phase: "tasks", total: total}

		bar := progress.New(
			progress.WithDefaultBlend(),
			progress.WithWidth(40),
			progress.WithoutPercentage(),
		)
		m := &reindexModel{
			bar:       bar,
			state:     state,
			startTime: startTime,
		}
		p := tea.NewProgram(m, tea.WithInput(os.Stdin))
		m.prog = p

		// Start reindex in background goroutine (outside Init, like dlProgressModel).
		go func() {
			engine := search.NewEngine(store, embedder, vecStore)
			err := engine.Reindex(func(phase string, current, totalItems int) {
				state.phase = phase
				state.processed = current
				state.total = totalItems
			})
			chunkCount := vecStore.Count()
			p.Send(reindexDoneMsg{err: err, chunkCount: chunkCount})
		}()

		if _, err := p.Run(); err != nil {
			// Fallback: no TTY — run with simple text progress.
			engine := search.NewEngine(store, embedder, vecStore)
			if err := engine.Reindex(func(phase string, current, totalItems int) {
				fmt.Fprintf(os.Stderr, "\r  Indexing %s (%d/%d)...", phase, current, totalItems)
			}); err != nil {
				fmt.Fprintln(os.Stderr)
				return fmt.Errorf("reindex failed: %w", err)
			}
			fmt.Fprintln(os.Stderr)
		} else if state.err != nil {
			return fmt.Errorf("reindex failed: %w", state.err)
		}

		fmt.Println()
		count := vecStore.Count()
		elapsed := int(time.Since(startTime).Seconds())
		fmt.Println(searchSuccessStyle.Render(
			fmt.Sprintf("✓ Search index rebuilt (%d tasks, %d docs, %d chunks) in %s",
				taskCount, docCount, count, formatDuration(elapsed))))
		return nil
	}

	// Fallback: keyword-only — no index to rebuild.
	if semanticErr != nil && semanticErr != search.ErrSemanticNotConfigured {
		fmt.Println(searchWarnStyle.Render(fmt.Sprintf("Semantic search initialization failed: %s", semanticErr)))
		fmt.Println()
	} else {
		fmt.Println(searchDimStyle.Render("Semantic search is not configured."))
		fmt.Println()
		fmt.Println(searchDimStyle.Render("Set up with:"))
		fmt.Println(searchDimStyle.Render("  knowns model download gte-small"))
		fmt.Println(searchDimStyle.Render("  knowns search --reindex"))
		fmt.Println()
	}
	fmt.Println(searchDimStyle.Render("Keyword search does not require indexing (scans tasks/docs on each query)."))
	fmt.Printf("Found %d tasks and %d docs available for keyword search.\n", taskCount, docCount)
	return nil
}

// ─── scoring helpers (CLI-local, used for non-engine paths) ──────────

func scoreTaskCLI(query string, t *models.Task) (float64, string) {
	tokens := tokenizeCLI(query)
	if len(tokens) == 0 {
		return 0, ""
	}

	score := 0.0
	snippet := ""

	for _, tok := range tokens {
		if containsTokenCI(t.Title, tok) {
			score += 3.0
		}
		if containsTokenCI(t.Description, tok) {
			score += 1.5
			if snippet == "" {
				snippet = excerptAround(t.Description, tok, 80)
			}
		}
		for _, ac := range t.AcceptanceCriteria {
			if containsTokenCI(ac.Text, tok) {
				score += 0.5
			}
		}
		if containsTokenCI(t.ImplementationNotes, tok) {
			score += 0.5
		}
		if containsTokenCI(t.ImplementationPlan, tok) {
			score += 0.5
		}
		for _, l := range t.Labels {
			if containsTokenCI(l, tok) {
				score += 0.5
			}
		}
	}

	return score / float64(len(tokens)), snippet
}

func tokenizeCLI(query string) []string {
	words := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return words
}

func containsTokenCI(text, token string) bool {
	return strings.Contains(strings.ToLower(text), token)
}

func excerptAround(text, token string, maxLen int) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, token)
	if idx < 0 {
		if len(text) > maxLen {
			return text[:maxLen]
		}
		return text
	}
	start := idx - 30
	if start < 0 {
		start = 0
	}
	end := idx + len(token) + 50
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}

func scoreToPercent(score, maxScore float64) int {
	if maxScore <= 0 {
		return 0
	}
	pct := int((score / maxScore) * 100)
	if pct > 100 {
		pct = 100
	}
	if pct < 1 && score > 0 {
		pct = 1
	}
	return pct
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func sortSearchResults(results []models.SearchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

func init() {
	searchCmd.Flags().String("type", "", "Search type: all|task|doc (default: all)")
	searchCmd.Flags().String("status", "", "Filter tasks by status")
	searchCmd.Flags().String("priority", "", "Filter tasks by priority")
	searchCmd.Flags().String("label", "", "Filter tasks by label")
	searchCmd.Flags().String("tag", "", "Filter docs by tag")
	searchCmd.Flags().String("assignee", "", "Filter tasks by assignee")
	searchCmd.Flags().Bool("keyword", false, "Force keyword-only search")
	searchCmd.Flags().Bool("reindex", false, "Rebuild the search index")
	searchCmd.Flags().Bool("setup", false, "Set up semantic search")
	searchCmd.Flags().Bool("status-check", false, "Show semantic search status")
	searchCmd.Flags().Bool("install-runtime", false, "Download and install ONNX Runtime")

	rootCmd.AddCommand(searchCmd)
}
