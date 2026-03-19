package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/howznguyen/knowns/internal/search"
)

// ─── download step ───────────────────────────────────────────────────

type downloadStep struct {
	label   string
	url     string
	dst     string
	size    int64 // populated at runtime from HEAD
	written int64
	done    bool
	err     error
	// post-download hook (e.g. extract archive)
	postHook func(dst string) error
}

// ─── multi-step setup model (bubbletea) ──────────────────────────────

type setupModel struct {
	steps    []downloadStep
	current  int
	bar      progress.Model
	spinner  spinner.Model
	started  time.Time
	quitting bool
	err      error

	// background download state
	resp    *http.Response
	outFile *os.File
	doneCh  chan error
}

// setupTickMsg triggers periodic UI refresh during download.
type setupTickMsg struct{}

// setupStepDoneMsg signals current step finished downloading.
type setupStepDoneMsg struct{ err error }

// setupPostHookDoneMsg signals post-hook finished.
type setupPostHookDoneMsg struct{ err error }

func newSetupModel(steps []downloadStep) *setupModel {
	bar := progress.New(
		progress.WithDefaultBlend(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(StyleDim),
	)
	return &setupModel{
		steps:   steps,
		bar:     bar,
		spinner: sp,
		started: time.Now(),
	}
}

func (m *setupModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startCurrentStep(),
	)
}

func (m *setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled")
			m.quitting = true
			m.cleanup()
			return m, tea.Quit
		}

	case setupTickMsg:
		if m.quitting {
			return m, nil
		}
		step := &m.steps[m.current]
		if step.size > 0 {
			pct := float64(step.written) / float64(step.size)
			cmd := m.bar.SetPercent(pct)
			return m, tea.Batch(cmd, m.tickCmd())
		}
		return m, m.tickCmd()

	case setupStepDoneMsg:
		step := &m.steps[m.current]
		if msg.err != nil {
			step.err = msg.err
			m.err = msg.err
			m.quitting = true
			return m, tea.Quit
		}
		// Run post-hook if present
		if step.postHook != nil {
			return m, func() tea.Msg {
				err := step.postHook(step.dst)
				return setupPostHookDoneMsg{err: err}
			}
		}
		return m, m.advanceStep()

	case setupPostHookDoneMsg:
		if msg.err != nil {
			m.steps[m.current].err = msg.err
			m.err = msg.err
			m.quitting = true
			return m, tea.Quit
		}
		return m, m.advanceStep()

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *setupModel) View() tea.View {
	var b strings.Builder

	for i, step := range m.steps {
		if step.done {
			sizeInfo := ""
			if step.written > 0 {
				sizeInfo = StyleDim.Render(fmt.Sprintf(" (%s)", formatBytes(step.written)))
			}
			b.WriteString(fmt.Sprintf("  %s %s%s\n",
				StyleSuccess.Render("✓"),
				step.label,
				sizeInfo,
			))
		} else if i == m.current && !m.quitting {
			// Active step
			pct := 0.0
			if step.size > 0 {
				pct = float64(step.written) / float64(step.size)
			}
			pctStr := fmt.Sprintf("%.0f%%", pct*100)

			elapsed := time.Since(m.started).Seconds()
			speed := ""
			if elapsed > 0.5 && step.written > 0 {
				speed = fmt.Sprintf("  %s/s", formatBytes(int64(float64(step.written)/elapsed)))
			}

			sizeInfo := ""
			if step.size > 0 {
				sizeInfo = fmt.Sprintf("  %s/%s", formatBytes(step.written), formatBytes(step.size))
			}

			b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), step.label))
			b.WriteString(fmt.Sprintf("    %s %s%s%s\n",
				m.bar.View(),
				StyleDim.Render(pctStr),
				StyleDim.Render(sizeInfo),
				StyleDim.Render(speed),
			))
		} else if step.err != nil {
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				StyleWarning.Render("✗"),
				step.label,
				StyleWarning.Render(step.err.Error()),
			))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				StyleDim.Render("○"),
				StyleDim.Render(step.label),
			))
		}
	}

	return tea.NewView(b.String())
}

func (m *setupModel) tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return setupTickMsg{}
	})
}

func (m *setupModel) advanceStep() tea.Cmd {
	m.steps[m.current].done = true
	m.current++
	if m.current >= len(m.steps) {
		m.quitting = true
		return tea.Quit
	}
	m.started = time.Now()
	return m.startCurrentStep()
}

func (m *setupModel) startCurrentStep() tea.Cmd {
	step := &m.steps[m.current]
	url := step.url
	dst := step.dst

	return func() tea.Msg {
		client := &http.Client{Timeout: 30 * time.Minute}

		// HEAD to get content length
		headResp, err := client.Head(url)
		if err == nil {
			step.size = headResp.ContentLength
			headResp.Body.Close()
		}

		// Ensure parent dir
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return setupStepDoneMsg{err: err}
		}

		resp, err := client.Get(url)
		if err != nil {
			return setupStepDoneMsg{err: err}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return setupStepDoneMsg{err: fmt.Errorf("HTTP %d", resp.StatusCode)}
		}

		if step.size <= 0 {
			step.size = resp.ContentLength
		}

		outFile, err := os.Create(dst)
		if err != nil {
			resp.Body.Close()
			return setupStepDoneMsg{err: err}
		}

		// Stream download with progress tracking
		buf := make([]byte, 32*1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := outFile.Write(buf[:n]); writeErr != nil {
					outFile.Close()
					resp.Body.Close()
					return setupStepDoneMsg{err: writeErr}
				}
				step.written += int64(n)
			}
			if readErr != nil {
				if readErr == io.EOF {
					break
				}
				outFile.Close()
				resp.Body.Close()
				return setupStepDoneMsg{err: readErr}
			}
		}

		outFile.Close()
		resp.Body.Close()
		return setupStepDoneMsg{err: nil}
	}
}

func (m *setupModel) cleanup() {
	if m.resp != nil {
		m.resp.Body.Close()
	}
	if m.outFile != nil {
		m.outFile.Close()
	}
}

// ─── public API ──────────────────────────────────────────────────────

// buildSemanticDownloadSteps returns initSteps for downloading ONNX Runtime
// and embedding model files. Returns (steps, alreadyInstalled, error).
// Used by the init command to integrate downloads into the progressive step UI.
func buildSemanticDownloadSteps(modelID string) ([]initStep, bool, error) {
	var selected *embeddingModel
	for i := range supportedModels {
		if supportedModels[i].ID == modelID {
			selected = &supportedModels[i]
			break
		}
	}
	if selected == nil {
		return nil, false, fmt.Errorf("unknown model %q", modelID)
	}

	if isModelInstalled(selected) {
		onnxOK, _ := search.IsONNXAvailable()
		if onnxOK {
			return nil, true, nil
		}
	}

	var steps []initStep

	// ONNX Runtime (if not available)
	if avail, _ := search.IsONNXAvailable(); !avail {
		url, libName, err := search.ONNXRuntimeDownloadURL()
		if err != nil {
			return nil, false, fmt.Errorf("unsupported platform: %w", err)
		}

		home, _ := os.UserHomeDir()
		destDir := filepath.Join(home, ".knowns", "lib")
		_ = os.MkdirAll(destDir, 0755)
		tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("onnxruntime-%d%s", time.Now().UnixNano(), onnxArchiveSuffix(url)))
		destPath := filepath.Join(destDir, libName)

		steps = append(steps, initStep{
			label: fmt.Sprintf("ONNX Runtime (%s/%s)", runtime.GOOS, runtime.GOARCH),
			url:   url,
			dst:   tmpPath,
			postHook: func(dst string) error {
				defer os.Remove(dst)
				return extractONNXLib(dst, libName, destPath)
			},
		})
	}

	// Model files
	if !isModelInstalled(selected) {
		modelDir := getModelDir(selected.HuggingFace)
		for _, file := range selected.Files {
			url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", selected.HuggingFace, file)
			dst := filepath.Join(modelDir, file)
			steps = append(steps, initStep{
				label: fmt.Sprintf("%s — %s", selected.Name, file),
				url:   url,
				dst:   dst,
			})
		}
	}

	return steps, false, nil
}

// runSemanticSetup downloads ONNX Runtime (if needed) and the embedding model
// using a unified bubbletea multi-step progress UI.
// Pass force=true to re-download even if already installed.
func runSemanticSetup(modelID string, force ...bool) error {
	forceDownload := len(force) > 0 && force[0]

	// Find the model
	var selected *embeddingModel
	for i := range supportedModels {
		if supportedModels[i].ID == modelID {
			selected = &supportedModels[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("unknown model %q", modelID)
	}

	if !forceDownload && isModelInstalled(selected) {
		onnxOK, _ := search.IsONNXAvailable()
		if onnxOK {
			fmt.Println(StyleSuccess.Render(fmt.Sprintf("✓ Semantic search ready (model: %s, ONNX Runtime: installed)", modelID)))
			return nil
		}
	}

	// Build download steps
	var steps []downloadStep

	// Step 1: ONNX Runtime (if not available)
	if avail, _ := search.IsONNXAvailable(); !avail {
		url, libName, err := search.ONNXRuntimeDownloadURL()
		if err != nil {
			return fmt.Errorf("unsupported platform: %w", err)
		}

		home, _ := os.UserHomeDir()
		destDir := filepath.Join(home, ".knowns", "lib")
		_ = os.MkdirAll(destDir, 0755)
		tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("onnxruntime-%d%s", time.Now().UnixNano(), onnxArchiveSuffix(url)))
		destPath := filepath.Join(destDir, libName)

		steps = append(steps, downloadStep{
			label: fmt.Sprintf("ONNX Runtime (%s/%s)", runtime.GOOS, runtime.GOARCH),
			url:   url,
			dst:   tmpPath,
			postHook: func(dst string) error {
				defer os.Remove(dst)
				return extractONNXLib(dst, libName, destPath)
			},
		})
	}

	// Step 2+: Model files
	if forceDownload || !isModelInstalled(selected) {
		modelDir := getModelDir(selected.HuggingFace)
		for _, file := range selected.Files {
			url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", selected.HuggingFace, file)
			dst := filepath.Join(modelDir, file)
			steps = append(steps, downloadStep{
				label: fmt.Sprintf("%s — %s", selected.Name, file),
				url:   url,
				dst:   dst,
			})
		}
	}

	if len(steps) == 0 {
		fmt.Println(StyleSuccess.Render("✓ Semantic search already set up"))
		return nil
	}

	fmt.Println()
	fmt.Printf("  Setting up semantic search (%d downloads)...\n\n", len(steps))

	// Drain any pending terminal escape responses from prior bubbletea/huh
	// programs to prevent ^[[?2026;2$y leak in output.
	drainStdin()

	m := newSetupModel(steps)
	p := tea.NewProgram(m, tea.WithInput(os.Stdin))
	if _, err := p.Run(); err != nil {
		return err
	}

	if m.err != nil {
		return m.err
	}

	fmt.Println()
	fmt.Println(StyleSuccess.Render("✓ Semantic search ready"))
	return nil
}

// ensureONNXRuntime downloads and installs ONNX Runtime if not already present.
// Returns nil immediately if already available. Shows a progress UI during download.
func ensureONNXRuntime() error {
	if avail, _ := search.IsONNXAvailable(); avail {
		return nil
	}

	url, libName, err := search.ONNXRuntimeDownloadURL()
	if err != nil {
		return fmt.Errorf("unsupported platform: %w", err)
	}

	home, _ := os.UserHomeDir()
	destDir := filepath.Join(home, ".knowns", "lib")
	_ = os.MkdirAll(destDir, 0755)
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("onnxruntime-%d%s", time.Now().UnixNano(), onnxArchiveSuffix(url)))
	destPath := filepath.Join(destDir, libName)

	steps := []downloadStep{
		{
			label: fmt.Sprintf("ONNX Runtime (%s/%s)", runtime.GOOS, runtime.GOARCH),
			url:   url,
			dst:   tmpPath,
			postHook: func(dst string) error {
				defer os.Remove(dst)
				return extractONNXLib(dst, libName, destPath)
			},
		},
	}

	fmt.Println()
	fmt.Println("  Installing ONNX Runtime...")
	fmt.Println()

	drainStdin()

	m := newSetupModel(steps)
	p := tea.NewProgram(m, tea.WithInput(os.Stdin))
	if _, err := p.Run(); err != nil {
		return err
	}

	if m.err != nil {
		return m.err
	}

	fmt.Println()
	fmt.Println(StyleSuccess.Render("✓ ONNX Runtime installed"))
	return nil
}
