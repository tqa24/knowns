package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/howznguyen/knowns/internal/codegen"
	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/runtimeinstall"
	"github.com/howznguyen/knowns/internal/server"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/spf13/cobra"
)

// embeddingModelInfo describes a supported embedding model for semantic search.
type embeddingModelInfo struct {
	ID            string
	Title         string
	Description   string
	HuggingFaceID string
	Dimensions    int
	MaxTokens     int
}

var supportedEmbeddingModels = []embeddingModelInfo{
	{
		ID:            "gte-small",
		Title:         "gte-small (recommended)",
		Description:   "384 dims, 67MB — best balance",
		HuggingFaceID: "Xenova/gte-small",
		Dimensions:    384,
		MaxTokens:     512,
	},
	{
		ID:            "all-MiniLM-L6-v2",
		Title:         "all-MiniLM-L6-v2",
		Description:   "384 dims, 45MB — fastest",
		HuggingFaceID: "Xenova/all-MiniLM-L6-v2",
		Dimensions:    384,
		MaxTokens:     256,
	},
	{
		ID:            "gte-base",
		Title:         "gte-base",
		Description:   "768 dims, 220MB — highest quality",
		HuggingFaceID: "Xenova/gte-base",
		Dimensions:    768,
		MaxTokens:     512,
	},
	{
		ID:            "bge-small-en-v1.5",
		Title:         "bge-small-en-v1.5",
		Description:   "384 dims, 67MB — strong retrieval",
		HuggingFaceID: "Xenova/bge-small-en-v1.5",
		Dimensions:    384,
		MaxTokens:     512,
	},
	{
		ID:            "bge-base-en-v1.5",
		Title:         "bge-base-en-v1.5",
		Description:   "768 dims, 220MB — top retrieval quality",
		HuggingFaceID: "Xenova/bge-base-en-v1.5",
		Dimensions:    768,
		MaxTokens:     512,
	},
	{
		ID:            "nomic-embed-text-v1.5",
		Title:         "nomic-embed-text-v1.5",
		Description:   "768 dims, 274MB — long context (8192 tokens)",
		HuggingFaceID: "nomic-ai/nomic-embed-text-v1.5",
		Dimensions:    768,
		MaxTokens:     8192,
	},
	{
		ID:            "multilingual-e5-small",
		Title:         "multilingual-e5-small",
		Description:   "384 dims, 471MB — multilingual support",
		HuggingFaceID: "Xenova/multilingual-e5-small",
		Dimensions:    384,
		MaxTokens:     512,
	},
}

// instructionFile defines an agent instruction file to generate during init.
type instructionFile struct {
	Path       string
	Platform   string // display name passed to generateInstructionContent
	PlatformID string // matches allPlatformIDs entry
}

const canonicalInstructionFile = "KNOWNS.md"

var defaultInstructionFiles = []instructionFile{
	{Path: "CLAUDE.md", Platform: "Claude Code", PlatformID: "claude-code"},
	{Path: "OPENCODE.md", Platform: "OpenCode", PlatformID: "opencode"},
	{Path: "GEMINI.md", Platform: "Gemini CLI", PlatformID: "gemini"},
	{Path: "AGENTS.md", Platform: "Generic AI", PlatformID: "agents"},
	{Path: filepath.Join(".github", "copilot-instructions.md"), Platform: "GitHub Copilot", PlatformID: "copilot"},
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Knowns project",
	Long: `Initialize a new Knowns project in the current directory.
Creates a .knowns/ directory with the required structure and a default config.json.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

// allPlatformIDs is the full ordered list of supported platform identifiers.
var allPlatformIDs = []string{"claude-code", "opencode", "codex", "kiro", "gemini", "copilot", "agents"}

// wizardPlatformIDs is the subset shown in the wizard multi-select.
var wizardPlatformIDs = []string{"claude-code", "opencode", "codex", "kiro", "gemini", "copilot", "agents"}

// platformLabel returns the human-readable label for a platform ID.
func platformLabel(id string) string {
	if label := platformLabelFromRuntime(id); label != "" {
		return label
	}
	switch id {
	case "gemini":
		return "Google Gemini  (GEMINI.md)"
	case "copilot":
		return "GitHub Copilot  (.github/copilot-instructions.md)"
	case "agents":
		return "Generic Agents  (AGENTS.md, .agent/skills/)"
	default:
		return id
	}
}

func platformLabelFromRuntime(id string) string {
	switch id {
	case "claude-code", "codex", "opencode", "kiro":
		return compactRuntimePickerLabel(id, runtimeinstall.DefaultOptions())
	default:
		return ""
	}
}

func compactRuntimePickerLabel(id string, opts runtimeinstall.Options) string {
	status := runtimeinstall.RuntimeAvailabilitySummary(id, opts)
	specLabel := map[string]string{
		"claude-code": "Claude Code (CLAUDE.md, SKILL, hooks, ...)",
		"codex":       "Codex (.codex/config.toml, SKILL, hooks, ...)",
		"opencode":    "OpenCode (OPENCODE.md, SKILL, plugin, MCP, ...)",
		"kiro":        "Kiro IDE (.kiro/steering, SKILL, hooks, ...)",
	}[id]
	if specLabel == "" {
		return id
	}
	return fmt.Sprintf("%s %s", runtimeStatusDot(status), specLabel)
}

func compactRuntimeCoverageSummary(opts runtimeinstall.Options) string {
	return ""
}

func runtimeStatusDot(status string) string {
	switch status {
	case "installed":
		return StyleSuccess.Render("●")
	case "available":
		return StyleWarning.Render("●")
	default:
		return StyleError.Render("●")
	}
}

// hasPlatform returns true if id is in platforms, or platforms is empty (= all enabled).
func hasPlatform(platforms []string, id string) bool {
	if len(platforms) == 0 {
		return true
	}
	for _, p := range platforms {
		if p == id {
			return true
		}
	}
	return false
}

// initConfig holds all wizard answers.
type initConfig struct {
	Name            string
	GitTrackingMode string
	EnableSemantic  bool
	SemanticModel   string
	Platforms       []string
	EnableChatUI    bool
}

// Aliases for centralized styles (see styles.go)
var (
	titleStyle   = StyleTitle
	successStyle = StyleSuccess
	warnStyle    = StyleWarning
	dimStyle     = StyleDim
)

func runInit(cmd *cobra.Command, args []string) error {
	gitTracked, _ := cmd.Flags().GetBool("git-tracked")
	gitIgnored, _ := cmd.Flags().GetBool("git-ignored")
	force, _ := cmd.Flags().GetBool("force")
	_, _ = cmd.Flags().GetBool("wizard")
	noWizard, _ := cmd.Flags().GetBool("no-wizard")
	openFlag, _ := cmd.Flags().GetBool("open")
	noOpen, _ := cmd.Flags().GetBool("no-open")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	root := filepath.Join(cwd, ".knowns")
	hasGitRepo := isGitRepo(cwd)
	gitAvailable := isGitAvailable()

	// Check if already initialized
	if _, err := os.Stat(root); err == nil {
		if !force {
			fmt.Println(warnStyle.Render("Project already initialized (.knowns/ directory exists)."))
			fmt.Println(dimStyle.Render("  Use --force to reinitialize."))
			return nil
		}
		fmt.Println(warnStyle.Render("Reinitializing existing project (--force)"))
		fmt.Println()
	}

	// Check git availability / repository status.
	if !hasGitRepo {
		if gitAvailable {
			fmt.Println(dimStyle.Render("No git repository found — Knowns will run git init after setup."))
			fmt.Println()
		} else {
			fmt.Println(warnStyle.Render("Warning: No git repository found and git is not available in PATH."))
			fmt.Println(dimStyle.Render("  Install git to enable repository setup and git-aware tracking."))
			fmt.Println()
		}
	}

	var cfg initConfig

	// Determine if interactive mode
	interactive := !noWizard

	if interactive && len(args) == 0 {
		// Load any existing config to pre-populate wizard defaults.
		var existingPlatforms []string
		var existingEnableChatUI *bool
		var existingName string
		var existingGitTrackingMode string
		var existingSemanticEnabled *bool
		var existingSemanticModel string
		if existingCfg, err := storage.NewStore(root).Config.Load(); err == nil {
			existingPlatforms = existingCfg.Settings.Platforms
			existingEnableChatUI = existingCfg.Settings.EnableChatUI
			existingName = existingCfg.Name
			existingGitTrackingMode = existingCfg.Settings.GitTrackingMode
			if existingCfg.Settings.SemanticSearch != nil {
				enabled := existingCfg.Settings.SemanticSearch.Enabled
				existingSemanticEnabled = &enabled
				existingSemanticModel = existingCfg.Settings.SemanticSearch.Model
			}
		}

		// Run full wizard with huh
		wizardCfg, err := runWizard(cwd, gitTracked, gitIgnored, gitAvailable, existingPlatforms, existingEnableChatUI, existingName, existingGitTrackingMode, existingSemanticEnabled, existingSemanticModel)
		if err != nil {
			if err == huh.ErrUserAborted {
				fmt.Println(warnStyle.Render("Setup cancelled."))
				return nil
			}
			return err
		}
		cfg = *wizardCfg
	} else {
		// Non-interactive or name provided
		name := filepath.Base(cwd)
		if len(args) > 0 {
			name = args[0]
		}
		gitMode := "git-tracked"
		if gitTracked {
			gitMode = "git-tracked"
		} else if gitIgnored {
			gitMode = "git-ignored"
		}
		cfg = initConfig{
			Name:            name,
			GitTrackingMode: gitMode,
			EnableSemantic:  isTTY(),
			SemanticModel:   "multilingual-e5-small",
			Platforms:       []string{"claude-code", "agents"},
			EnableChatUI:    true,
		}
	}

	// Build init steps
	steps := []initStep{
		{
			label: "Creating project structure",
			run: func() error {
				store := storage.NewStore(root)
				return store.Init(cfg.Name)
			},
		},
		{
			label: "Applying settings",
			run: func() error {
				store := storage.NewStore(root)
				project, err := store.Config.Load()
				if err != nil {
					return err
				}
				if cfg.GitTrackingMode != "" {
					project.Settings.GitTrackingMode = cfg.GitTrackingMode
				}
				if cfg.EnableSemantic && cfg.SemanticModel != "" {
					m := findEmbeddingModel(cfg.SemanticModel)
					if m != nil {
						project.Settings.SemanticSearch = &models.SemanticSearchSettings{
							Enabled:       true,
							Model:         m.ID,
							HuggingFaceID: m.HuggingFaceID,
							Dimensions:    m.Dimensions,
							MaxTokens:     m.MaxTokens,
						}
					}
				}
				if len(cfg.Platforms) > 0 {
					project.Settings.Platforms = cfg.Platforms
				}
				enableChatUI := cfg.EnableChatUI
				project.Settings.EnableChatUI = &enableChatUI
				return store.Config.Save(project)
			},
		},
		{
			label: "Configuring git integration",
			run: func() error {
				return writeKnownsGitignore(cwd, cfg.GitTrackingMode)
			},
		},
	}

	if !hasGitRepo && gitAvailable {
		steps = append([]initStep{{
			label: "Initializing git repository",
			run: func() error {
				return gitInit(cwd)
			},
		}}, steps...)
	}

	// Conditional semantic download steps
	if cfg.EnableSemantic {
		dlSteps, alreadyInstalled, err := buildSemanticDownloadSteps(cfg.SemanticModel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: semantic search setup failed: %v\n", err)
			fmt.Println(dimStyle.Render(fmt.Sprintf("  You can set up later: knowns model download %s", cfg.SemanticModel)))
		} else if alreadyInstalled {
			steps = append(steps, initStep{
				label: "Semantic search (already installed)",
				run:   func() error { return nil },
			})
		} else {
			steps = append(steps, dlSteps...)
		}
	}

	if cfg.EnableSemantic {
		steps = append(steps, initStep{
			label: "Preparing project and global semantic stores",
			run: func() error {
				store := storage.NewStore(root)
				_, _, err := ensureProjectAndGlobalSemanticReady(store, cfg.SemanticModel)
				return err
			},
		})
	}

	// OpenCode install check (before config steps)
	if hasPlatform(cfg.Platforms, "opencode") {
		fmt.Println()
		if err := maybeInstallOpenCode(force); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: OpenCode setup issue: %v\n", err)
		}
		fmt.Println()
	}

	steps = append(steps,
		initStep{
			label: "Syncing skills",
			run: func() error {
				return codegen.SyncSkillsForPlatforms(cwd, cfg.Platforms)
			},
		},
	)
	if hasPlatform(cfg.Platforms, "claude-code") {
		steps = append(steps, initStep{
			label: "Creating MCP config",
			run: func() error {
				return createMCPJsonFileQuiet(cwd, force)
			},
		})
	}
	if hasPlatform(cfg.Platforms, "opencode") {
		steps = append(steps, initStep{
			label: "Creating OpenCode config",
			run: func() error {
				return createOpenCodeConfigQuiet(cwd)
			},
		})
	}
	if hasPlatform(cfg.Platforms, "kiro") {
		steps = append(steps, initStep{
			label: "Creating Kiro steering",
			run: func() error {
				return createKiroSteeringQuiet(cwd, force)
			},
		})
		steps = append(steps, initStep{
			label: "Creating Kiro MCP config",
			run: func() error {
				return createKiroMCPConfigQuiet(cwd)
			},
		})
	}
	for _, runtimeName := range []string{"claude-code", "codex", "kiro", "opencode"} {
		if !hasPlatform(cfg.Platforms, runtimeName) {
			continue
		}
		selectedRuntime := runtimeName
		opts := runtimeinstall.DefaultOptions()
		// Skip runtimes that are unavailable and cannot be auto-installed.
		if !runtimeinstall.CanAutoInstall(selectedRuntime) {
			st, err := runtimeinstall.StatusFor(selectedRuntime, opts)
			if err != nil || !st.Available {
				continue
			}
		}
		steps = append(steps, initStep{
			label: fmt.Sprintf("Installing %s runtime hooks", runtimeinstall.RuntimePickerLabel(selectedRuntime, opts)),
			run: func() error {
				return runtimeinstall.Install(selectedRuntime, opts)
			},
		})
	}
	steps = append(steps,
		initStep{
			label: "Creating instruction files",
			run: func() error {
				return createInstructionFilesForPlatforms(cwd, force, cfg.Platforms)
			},
		},
	)
	if cfg.EnableSemantic {
		steps = append(steps, initStep{
			label: "Building project and global semantic indices",
			run: func() error {
				store := storage.NewStore(root)
				return reindexSemanticStores(store)
			},
		})
	}

	fmt.Println()
	if err := runInitSteps(steps); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	fmt.Println()
	fmt.Println(titleStyle.Render("Get started:"))
	fmt.Println(dimStyle.Render("  knowns task create \"My first task\""))
	fmt.Println(dimStyle.Render("  Use /kn-init to start an AI session"))
	if cfg.EnableChatUI {
		fmt.Println(dimStyle.Render("  knowns browser --open   # Launch Chat UI"))
	}
	fmt.Println()
	return maybeOpenBrowser(cwd, openFlag, noOpen)
}

func runWizard(cwd string, gitTracked, gitIgnored bool, gitAvailable bool, existingPlatforms []string, existingEnableChatUI *bool, existingName string, existingGitTrackingMode string, existingSemanticEnabled *bool, existingSemanticModel string) (*initConfig, error) {
	defaultName := filepath.Base(cwd)
	if existingName != "" {
		defaultName = existingName
	}
	hasGit := isGitRepo(cwd)

	fmt.Println()
	fmt.Println(titleStyle.Render("🚀 Knowns Project Setup"))
	fmt.Println(dimStyle.Render("   Quick configuration"))
	fmt.Println()

	var cfg initConfig
	cfg.Name = defaultName

	// --- Group 1: Project name ---
	nameField := huh.NewGroup(
		huh.NewInput().
			Title("Project name").
			Value(&cfg.Name).
			Placeholder(defaultName).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("project name is required")
				}
				return nil
			}),
	)

	// --- Group 1b: Git tracking mode (only if in a git repo and not set via flag) ---
	var gitGroup *huh.Group
	if (hasGit || gitAvailable) && !gitTracked && !gitIgnored {
		cfg.GitTrackingMode = "git-tracked"
		if existingGitTrackingMode != "" {
			cfg.GitTrackingMode = existingGitTrackingMode
		}
		gitGroup = huh.NewGroup(
			huh.NewSelect[string]().
				Title("Git tracking mode").
				Description("Choose what Knowns data is committed to git.").
				Options(
					huh.NewOption("Git Tracked  · tasks, docs, templates", "git-tracked"),
					huh.NewOption("Git Ignored  · docs, templates only", "git-ignored"),
					huh.NewOption("None  · manage tracking manually", "none"),
				).
				Value(&cfg.GitTrackingMode),
		)
	} else if gitTracked {
		cfg.GitTrackingMode = "git-tracked"
	} else if gitIgnored {
		cfg.GitTrackingMode = "git-ignored"
	}

	// --- Group 2: AI platform selection ---
	defaultPlatforms := []string{"claude-code", "opencode", "agents"}
	if len(existingPlatforms) > 0 {
		defaultPlatforms = append([]string(nil), existingPlatforms...)
		if len(defaultPlatforms) == 0 {
			defaultPlatforms = []string{"claude-code", "opencode", "agents"}
		}
	}
	cfg.Platforms = append([]string(nil), defaultPlatforms...)
	selectedSet := make(map[string]bool, len(defaultPlatforms))
	for _, p := range defaultPlatforms {
		selectedSet[p] = true
	}
	platformOptions := make([]huh.Option[string], len(wizardPlatformIDs))
	for i, id := range wizardPlatformIDs {
		platformOptions[i] = huh.NewOption(platformLabel(id), id).Selected(selectedSet[id])
	}
	runtimeSummary := compactRuntimeCoverageSummary(runtimeinstall.DefaultOptions())
	group2 := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("AI platforms to integrate").
			Description("Choose which platforms to generate config and instruction files for.\n" +
				"At least one platform must be selected.\n\n" + runtimeSummary).
			Options(platformOptions...).
			Validate(func(selected []string) error {
				if len(selected) == 0 {
					return fmt.Errorf("select at least one platform")
				}
				return nil
			}).
			Value(&cfg.Platforms),
	)

	// --- Group 3: Chat UI ---
	if existingEnableChatUI != nil {
		cfg.EnableChatUI = *existingEnableChatUI
	} else {
		cfg.EnableChatUI = hasPlatform(existingPlatforms, "opencode")
	}
	if hasPlatform(cfg.Platforms, "opencode") {
		cfg.EnableChatUI = true
	}
	chatUIDesc := "Enables Chat UI with AI code sessions, task-linked conversations,\n" +
		"and live coding assistance powered by OpenCode.\n"
	if hasPlatform(cfg.Platforms, "opencode") {
		chatUIDesc += runtimeinstall.RuntimePickerDescription("opencode", runtimeinstall.DefaultOptions())
	} else {
		chatUIDesc += "Select OpenCode above if you want Knowns to configure its runtime plugin as part of init."
	}
	group3 := huh.NewGroup(
		huh.NewConfirm().
			Title("Enable Chat UI?").
			Description(chatUIDesc).
			Value(&cfg.EnableChatUI),
	).WithHideFunc(func() bool {
		if hasPlatform(cfg.Platforms, "opencode") {
			cfg.EnableChatUI = true
			return true
		}
		return false
	})

	// --- Group 4: Semantic search ---
	cfg.EnableSemantic = true
	cfg.SemanticModel = "multilingual-e5-small"
	if existingSemanticEnabled != nil {
		cfg.EnableSemantic = *existingSemanticEnabled
	}
	if existingSemanticModel != "" {
		cfg.SemanticModel = existingSemanticModel
	}
	group5 := huh.NewGroup(
		huh.NewConfirm().
			Title("Enable semantic search?").
			Description("Requires embedding model download").
			Value(&cfg.EnableSemantic),
	)

	// --- Group 6: Model selection (only shown if semantic enabled) ---
	modelOptions := make([]huh.Option[string], len(supportedEmbeddingModels))
	for i, m := range supportedEmbeddingModels {
		modelOptions[i] = huh.NewOption(fmt.Sprintf("%s — %s", m.Title, m.Description), m.ID)
	}
	group6 := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select embedding model").
			Options(modelOptions...).
			Value(&cfg.SemanticModel),
	).WithHideFunc(func() bool {
		return !cfg.EnableSemantic
	})

	// Run form
	groups := []*huh.Group{nameField}
	if gitGroup != nil {
		groups = append(groups, gitGroup)
	}
	groups = append(groups, group2, group3, group5, group6)

	form := huh.NewForm(groups...).
		WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return nil, err
	}

	if cfg.EnableChatUI && !hasPlatform(cfg.Platforms, "opencode") {
		cfg.Platforms = append(cfg.Platforms, "opencode")
	}

	return &cfg, nil
}

func findEmbeddingModel(id string) *embeddingModelInfo {
	for _, m := range supportedEmbeddingModels {
		if m.ID == id {
			return &m
		}
	}
	return nil
}

// execLookPath is used to locate binaries in PATH. Overridable in tests.
var execLookPath = exec.LookPath

// defaultExecLookPath is the original value of execLookPath for test cleanup.
var defaultExecLookPath = exec.LookPath

// execCommand is used to run external commands in init flows. Overridable in tests.
var execCommand = exec.Command

// defaultExecCommand is the original value of execCommand for test cleanup.
var defaultExecCommand = exec.Command

func isGitAvailable() bool {
	_, err := execLookPath("git")
	return err == nil
}

func gitInit(dir string) error {
	cmd := execCommand("git", "init")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("git init failed: %w", err)
		}
		return fmt.Errorf("git init failed: %s", trimmed)
	}
	return nil
}

// mcpCommand returns the command and args for starting the Knowns MCP server
// in generated project configs. Uses the local knowns binary if available,
// otherwise falls back to npx so configs work on machines without a global install.
func mcpCommand() (command string, args []string) {
	if _, err := execLookPath("knowns"); err == nil {
		return "knowns", []string{"mcp", "--stdio"}
	}
	return "npx", []string{"-y", "knowns", "mcp", "--stdio"}
}

// mcpCommandFlat returns the MCP command as a single slice (for OpenCode config format).
func mcpCommandFlat() []string {
	cmd, args := mcpCommand()
	return append([]string{cmd}, args...)
}

// createMCPJsonFileQuiet creates .mcp.json without printing (for step runner).
func createMCPJsonFileQuiet(projectRoot string, force bool) error {
	mcpPath := filepath.Join(projectRoot, ".mcp.json")
	if _, err := os.Stat(mcpPath); err == nil && !force {
		return nil
	}

	cmd, args := mcpCommand()
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"knowns": map[string]interface{}{
				"command": cmd,
				"args":    args,
			},
		},
	}

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(mcpPath, data, 0644)
}

func createOpenCodeConfigQuiet(projectRoot string) error {
	configPath := filepath.Join(projectRoot, "opencode.json")

	config := map[string]any{
		"$schema": "https://opencode.ai/config.json",
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parse opencode.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	config["$schema"] = "https://opencode.ai/config.json"

	mcp, ok := config["mcp"].(map[string]any)
	if !ok || mcp == nil {
		mcp = make(map[string]any)
	}

	mcp["knowns"] = map[string]any{
		"type":    "local",
		"command": mcpCommandFlat(),
		"enabled": true,
	}

	config["mcp"] = mcp

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, append(data, '\n'), 0644)
}

// createKiroSteeringQuiet creates .kiro/steering/knowns.md that references
// KNOWNS.md via Kiro's #[[file:...]] directive so the agent always loads the
// canonical guidelines automatically.
func createKiroSteeringQuiet(projectRoot string, force bool) error {
	steeringDir := filepath.Join(projectRoot, ".kiro", "steering")
	if err := os.MkdirAll(steeringDir, 0755); err != nil {
		return fmt.Errorf("create .kiro/steering: %w", err)
	}

	steeringPath := filepath.Join(steeringDir, "knowns.md")
	if _, err := os.Stat(steeringPath); err == nil && !force {
		return nil
	}

	content := `---
description: Knowns project guidelines — always included so the agent follows repo conventions.
---

# Knowns Guidelines

This steering file ensures the agent reads the canonical project guidance on every interaction.

#[[file:KNOWNS.md]]
`
	return os.WriteFile(steeringPath, []byte(content), 0644)
}

// createKiroMCPConfigQuiet creates .kiro/settings/mcp.json with the Knowns
// MCP server entry. It merges into an existing file if present.
func createKiroMCPConfigQuiet(projectRoot string) error {
	settingsDir := filepath.Join(projectRoot, ".kiro", "settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("create .kiro/settings: %w", err)
	}

	configPath := filepath.Join(settingsDir, "mcp.json")

	config := map[string]any{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parse .kiro/settings/mcp.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok || servers == nil {
		servers = make(map[string]any)
	}

	cmd, args := mcpCommand()
	servers["knowns"] = map[string]any{
		"command":     cmd,
		"args":        args,
		"disabled":    false,
		"autoApprove": []string{"*"},
	}

	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, append(data, '\n'), 0644)
}

// createInstructionFilesForPlatforms generates only instruction files for the
// given platform IDs. If platforms is empty all files are generated.
func createInstructionFilesForPlatforms(projectRoot string, force bool, platforms []string) error {
	if err := writeInstructionFile(projectRoot, canonicalInstructionFile, "Knowns", force); err != nil {
		return err
	}

	for _, f := range defaultInstructionFiles {
		if !hasPlatform(platforms, f.PlatformID) {
			continue
		}
		if err := writeInstructionFile(projectRoot, f.Path, f.Platform, force); err != nil {
			return err
		}
	}
	return nil
}

// createInstructionFilesQuiet generates agent instruction files without printing.
func createInstructionFilesQuiet(projectRoot string, force bool) error {
	if err := writeInstructionFile(projectRoot, canonicalInstructionFile, "Knowns", force); err != nil {
		return err
	}

	for _, f := range defaultInstructionFiles {
		if err := writeInstructionFile(projectRoot, f.Path, f.Platform, force); err != nil {
			return err
		}
	}
	return nil
}

// createMCPJsonFile creates .mcp.json for Claude Code MCP auto-discovery.
func createMCPJsonFile(projectRoot string, force bool) {
	mcpPath := filepath.Join(projectRoot, ".mcp.json")
	if _, err := os.Stat(mcpPath); err == nil && !force {
		return
	}

	cmd, args := mcpCommand()
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"knowns": map[string]interface{}{
				"command": cmd,
				"args":    args,
			},
		},
	}

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return
	}

	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create .mcp.json: %v\n", err)
	} else {
		fmt.Println(successStyle.Render("✓ Created .mcp.json") + dimStyle.Render(" (Claude Code MCP auto-discovery)"))
	}
}

// createInstructionFiles generates agent instruction files.
func createInstructionFiles(projectRoot string, force bool) {
	canonicalPath := filepath.Join(projectRoot, canonicalInstructionFile)
	canonicalExists := false
	if _, err := os.Stat(canonicalPath); err == nil {
		canonicalExists = true
	}
	if err := writeInstructionFile(projectRoot, canonicalInstructionFile, "Knowns", force); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create %s: %v\n", canonicalInstructionFile, err)
	} else if canonicalExists && !force {
		fmt.Println(dimStyle.Render(fmt.Sprintf("  Skipped: %s (already exists)", canonicalInstructionFile)))
	} else {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ Created: %s", canonicalInstructionFile)))
	}

	for _, f := range defaultInstructionFiles {
		filePath := filepath.Join(projectRoot, f.Path)
		if _, err := os.Stat(filePath); err == nil && !force {
			fmt.Println(dimStyle.Render(fmt.Sprintf("  Skipped: %s (already exists)", f.Path)))
			continue
		}

		if err := writeInstructionFile(projectRoot, f.Path, f.Platform, force); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create %s: %v\n", f.Path, err)
		} else {
			fmt.Println(successStyle.Render(fmt.Sprintf("✓ Created: %s", f.Path)))
		}
	}
}

func writeInstructionFile(projectRoot, relativePath, platform string, force bool) error {
	filePath := filepath.Join(projectRoot, relativePath)
	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
		if !force {
			return nil
		}
	}

	if dir := filepath.Dir(filePath); dir != projectRoot {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	content := generateInstructionContent(relativePath, platform, projectRoot)

	// For compatibility shim files that already exist, preserve user content
	// outside the managed marker block.
	if fileExists && relativePath != canonicalInstructionFile {
		return syncInstructionMarkerBlock(filePath, content)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("could not create %s: %w", relativePath, err)
	}

	return nil
}

func generateInstructionContent(relativePath, platform, projectRoot string) string {
	if relativePath == canonicalInstructionFile {
		return renderCanonicalInstructionContent()
	}

	return renderCompatibilityInstructionContent(relativePath, platform, projectRoot)
}

func renderCanonicalInstructionContent() string {
	var sb strings.Builder
	sb.WriteString("# KNOWNS\n\n")
	sb.WriteString("Canonical repository guidance for agents working in this project.\n\n")
	sb.WriteString("## Table of Contents\n\n")
	sb.WriteString("- [Source of Truth](#source-of-truth)\n")
	sb.WriteString("- [TL;DR](#tldr)\n")
	sb.WriteString("- [Repo Mental Model](#repo-mental-model)\n")
	sb.WriteString("- [How Agents Should Read This File](#how-agents-should-read-this-file)\n")
	sb.WriteString("- [Tool Selection](#tool-selection)\n")
	sb.WriteString("- [Memory Usage](#memory-usage)\n")
	sb.WriteString("- [Critical Rules](#critical-rules)\n")
	sb.WriteString("- [Git Safety](#git-safety)\n")
	sb.WriteString("- [Context Retrieval Strategy](#context-retrieval-strategy)\n")
	sb.WriteString("- [References](#references)\n")
	sb.WriteString("- [Common Mistakes](#common-mistakes)\n")
	sb.WriteString("- [Recommended File Roles](#recommended-file-roles)\n")
	sb.WriteString("- [Compatibility Pattern](#compatibility-pattern)\n")
	sb.WriteString("- [Maintenance Rules](#maintenance-rules)\n\n")
	sb.WriteString("## Source of Truth\n\n")
	sb.WriteString("- `KNOWNS.md` is the canonical repo-level guidance file.\n")
	sb.WriteString("- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `OPENCODE.md`, and `.github/copilot-instructions.md` are compatibility shims for runtimes that auto-detect those filenames.\n")
	sb.WriteString("- If guidance appears in multiple places, follow this precedence order:\n")
	sb.WriteString("  1. System instructions\n")
	sb.WriteString("  2. Developer instructions\n")
	sb.WriteString("  3. `KNOWNS.md`\n")
	sb.WriteString("  4. Compatibility shim files\n")
	sb.WriteString("  5. Other repository docs\n")
	sb.WriteString("- If a shim file and `KNOWNS.md` differ, treat `KNOWNS.md` as correct.\n\n")
	sb.WriteString("## TL;DR\n\n")
	sb.WriteString("- Read `KNOWNS.md` first.\n")
	sb.WriteString("- Call MCP `status` (or `knowns status --json`) at session start to check project readiness, available capabilities, and knowledge counts.\n")
	sb.WriteString("- Use Knowns as the memory layer for humans and the AI-friendly working layer for agents.\n")
	sb.WriteString("- Search before reading; read only the sections and docs relevant to the current task.\n")
	sb.WriteString("- Never manually edit Knowns-managed task or doc markdown.\n")
	sb.WriteString("- Prefer Knowns MCP tools; use the `knowns` CLI only as fallback.\n")
	sb.WriteString("- Let skills handle detailed workflows; use this file for rules, conventions, and context routing.\n")
	sb.WriteString("- Validate before marking work complete.\n")
	sb.WriteString("- Do not revert user changes you did not make.\n\n")
	sb.WriteString("## Repo Mental Model\n\n")
	sb.WriteString("- Knowns is the project's memory layer for humans and the AI-friendly operating layer for agents.\n")
	sb.WriteString("- Knowns manages tasks, docs, templates, specs, references, and workflow state in one place.\n")
	sb.WriteString("- Tasks and docs may reference each other using `@task-<id>`, `@doc/<path>`, and `@template/<name>`.\n")
	sb.WriteString("- `KNOWNS.md` defines repo-level operating rules; skills define step-by-step execution flows.\n")
	sb.WriteString("- Long guidance should be retrieved by section, not blindly injected in full on every request.\n\n")
	sb.WriteString("## How Agents Should Read This File\n\n")
	sb.WriteString("- Always read `## Source of Truth` and `## TL;DR` first.\n")
	sb.WriteString("- For short or obvious tasks, use the summary sections plus the relevant section only.\n")
	sb.WriteString("- For tool usage questions, read `## Tool Selection` and `## Common Mistakes`.\n")
	sb.WriteString("- For safety-sensitive work, read `## Critical Rules` and `## Git Safety`.\n")
	sb.WriteString("- For large files or docs, read `## Context Retrieval Strategy`.\n")
	sb.WriteString("- For ambiguous requests, search the repo and related docs before asking the user.\n")
	sb.WriteString("- Do not assume the entire file is present in context; retrieve the needed sections when required.\n\n")
	sb.WriteString("## Tool Selection\n\n")
	sb.WriteString("- Use MCP `status` at session start to check project readiness and available capabilities before acting.\n")
	sb.WriteString("- Use Knowns MCP tools first for tasks, docs, templates, validation, and time tracking.\n")
	sb.WriteString("- Use file reading and search tools for local code and text inspection.\n")
	sb.WriteString("- Use shell commands for git, tests, builds, generators, and other terminal operations.\n")
	sb.WriteString("- Prefer targeted retrieval over loading large files in full.\n")
	sb.WriteString("- Use `knowns search` for discovery and quick relevance checks.\n")
	sb.WriteString("- Use MCP `retrieve` tool when a workflow needs structured context with citations and context-pack assembly. Fall back to CLI `knowns retrieve` if MCP is unavailable.\n")
	sb.WriteString("- Prefer `--json` for structured CLI reads consumed by agents, scripts, or workflows, including `get`, `list`, `search`, and `retrieve` commands.\n")
	sb.WriteString("- Prefer `--plain` for human-facing inspection, quick content reads, and logs when JSON is unnecessary.\n")
	sb.WriteString("- Do not rely on styled default CLI output for automation or parsing.\n\n")
	sb.WriteString("### Preferred Tool Matrix\n\n")
	sb.WriteString("- `knowns_*`: canonical operations on tasks, docs, templates, validation, and time.\n")
	sb.WriteString("- `read`: inspect a known file.\n")
	sb.WriteString("- `glob`: find files by path pattern.\n")
	sb.WriteString("- `grep`: locate content by regex.\n")
	sb.WriteString("- `bash`: run git, builds, tests, package managers, or other terminal commands.\n")
	sb.WriteString("- `apply_patch`: make small, explicit file edits.\n")
	sb.WriteString("- `task`: delegate large research or multi-step exploration when useful.\n\n")
	sb.WriteString("## Memory Usage\n\n")
	sb.WriteString("- Session start: `memory({ action: \"list\", layer: \"project\" })` to load accumulated project knowledge.\n")
	sb.WriteString("- After task: `memory({ action: \"add\" })` for reusable patterns, decisions, and conventions (alongside docs).\n")
	sb.WriteString("- Cross-project: `memory({ action: \"promote\" })` to move project knowledge to global (`project→global`).\n")
	sb.WriteString("- Memory complements docs: memory is for fast agent recall, docs are for structured human-readable reference.\n")
	sb.WriteString("- Never duplicate the full doc content into memory — store a summary and reference the doc with `@doc/<path>`.\n")
	sb.WriteString("- During any skill: if you discover a reusable pattern, decision, convention, or failure, save it with `memory({ action: \"add\", layer: \"project\" })`. Capture knowledge as it emerges, don't wait for extraction.\n")
	sb.WriteString("- Proactively save durable memory without waiting for the user to say \"save this\" when confidence is high.\n")
	sb.WriteString("- Use `project` for repo-specific rules, architecture decisions, conventions, recurring failure patterns, and implementation constraints.\n")
	sb.WriteString("- Use `global` for stable user preferences or workflow rules that should carry across repositories and future sessions.\n")
	sb.WriteString("- Ask the user only when the information appears durable but the correct scope (`working`, `project`, or `global`) is genuinely ambiguous.\n")
	sb.WriteString("- After any meaningful user instruction, correction, or newly discovered pattern, quickly evaluate whether it should be stored as memory and save it when appropriate.\n")
	sb.WriteString("- If the user states a stable collaboration preference, default to saving it as `global` memory unless they clearly scoped it to this repository only.\n\n")
	sb.WriteString("## Critical Rules\n\n")
	sb.WriteString("- Never manually edit Knowns-managed task or doc markdown.\n")
	sb.WriteString("- Search first, then read only relevant docs and code.\n")
	sb.WriteString("- Follow `@task-<id>`, `@doc/<path>`, and `@template/<name>` references before acting.\n")
	sb.WriteString("- Use `appendNotes` for progress updates; `notes` replaces existing notes and should only be used intentionally.\n")
	sb.WriteString("- Validate before marking work complete.\n")
	sb.WriteString("- Use skills for detailed workflow execution instead of duplicating step-by-step process here.\n")
	sb.WriteString("- Compatibility shim files must stay lightweight and must direct agents back to `KNOWNS.md` for behavioral rules instead of restating divergent guidance.\n\n")
	sb.WriteString("## Git Safety\n\n")
	sb.WriteString("- Assume the worktree may already contain user changes.\n")
	sb.WriteString("- Never revert or overwrite unrelated user changes unless explicitly requested.\n")
	sb.WriteString("- Avoid destructive git commands unless explicitly requested.\n")
	sb.WriteString("- Do not amend commits unless explicitly requested.\n")
	sb.WriteString("- Do not create commits unless the user explicitly asks for a commit.\n")
	sb.WriteString("- Do not push unless the user explicitly asks for it.\n\n")
	sb.WriteString("## Context Retrieval Strategy\n\n")
	sb.WriteString("- Treat `KNOWNS.md` as an indexed manual, not a prompt to fully inject every time.\n")
	sb.WriteString("- Read in this order when context is limited:\n")
	sb.WriteString("  1. `## Source of Truth`\n")
	sb.WriteString("  2. `## TL;DR`\n")
	sb.WriteString("  3. The section most relevant to the task\n")
	sb.WriteString("- For large or complex tasks, retrieve additional sections on demand.\n")
	sb.WriteString("- Prefer section headings with stable names so tools can target them precisely.\n")
	sb.WriteString("- If a downstream runtime supports startup loading, preload only the top-level summary and fetch deeper sections lazily.\n\n")
	sb.WriteString("## References\n\n")
	sb.WriteString("- Task references use `@task-<id>`.\n")
	sb.WriteString("- Doc references use `@doc/<path>`.\n")
	sb.WriteString("- Template references use `@template/<name>`.\n")
	sb.WriteString("- Doc references support line and range suffixes:\n")
	sb.WriteString("  - `@doc/<path>:42` — link to a specific line.\n")
	sb.WriteString("  - `@doc/<path>:10-25` — link to a line range.\n")
	sb.WriteString("  - `@doc/<path>#heading-slug` — link to a heading anchor.\n")
	sb.WriteString("- Follow references recursively before planning, implementation, or validation work.\n\n")
	sb.WriteString("## Common Mistakes\n\n")
	sb.WriteString("### Notes vs Append Notes\n\n")
	sb.WriteString("- Use `appendNotes` for progress updates and audit trail entries.\n")
	sb.WriteString("- Use `notes` only when intentionally replacing the task's notes content.\n\n")
	sb.WriteString("### CLI Pitfalls\n\n")
	sb.WriteString("- In `task create` and `task edit`, `-a` means `--assignee`, not acceptance criteria.\n")
	sb.WriteString("- In `doc edit`, `-a` means `--append`.\n")
	sb.WriteString("- Use raw task IDs where a command expects an ID value rather than a mention.\n")
	sb.WriteString("- Use `--plain` for read, list, and search commands, not for create or edit commands.\n")
	sb.WriteString("- Use `--json` for structured reads like `get`, `list`, `search`, and `retrieve` when the output will be parsed or fed into an agent workflow.\n")
	sb.WriteString("- Use `--plain` when inspecting manually or when only clean text output is needed.\n")
	sb.WriteString("- Use `--smart` when reading docs through the CLI.\n\n")
	sb.WriteString("### Retrieval Pitfalls\n\n")
	sb.WriteString("- Do not read every doc hoping to find the answer; search first.\n")
	sb.WriteString("- Do not replace discovery-oriented `search` with `retrieve` by default; use `retrieve` only when you need assembled context, citations, or expansion metadata.\n")
	sb.WriteString("- Do not repeatedly list the same tasks or docs if the needed context is already loaded.\n")
	sb.WriteString("- Do not quote large file contents when a concise summary is enough.\n\n")
	sb.WriteString("## Recommended File Roles\n\n")
	sb.WriteString("- `KNOWNS.md`: canonical repo-level guide.\n")
	sb.WriteString("- Compatibility shim files: lightweight entrypoints that introduce Knowns and redirect runtimes to `KNOWNS.md`.\n")
	sb.WriteString("- Other docs: deeper domain, feature, or workflow references.\n\n")
	sb.WriteString("## Compatibility Pattern\n\n")
	sb.WriteString("- Keep shim files short.\n")
	sb.WriteString("- In every shim file, explicitly say that `KNOWNS.md` is canonical.\n")
	sb.WriteString("- Preserve the `<!-- KNOWNS GUIDELINES START -->` and `<!-- KNOWNS GUIDELINES END -->` markers in shim files so tooling can detect and sync them reliably.\n\n")
	sb.WriteString("## Maintenance Rules\n\n")
	sb.WriteString("- Update the Knowns generator when the repository's operational rules change.\n")
	sb.WriteString("- Keep top sections stable so automated loaders can depend on them.\n")
	sb.WriteString("- Prefer adding new sections over bloating the TL;DR.\n")
	sb.WriteString("- Keep workflow details in skills when possible; keep `KNOWNS.md` focused on rules, conventions, and routing.\n")

	return sb.String()
}

func renderCompatibilityInstructionContent(relativePath, platform, projectRoot string) string {
	projectName := filepath.Base(projectRoot)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", compatibilityInstructionTitle(relativePath, platform, projectName)))
	sb.WriteString(fmt.Sprintf("Compatibility entrypoint for runtimes that auto-detect `%s`.\n\n", relativePath))
	sb.WriteString("<!-- KNOWNS GUIDELINES START -->\n\n")

	// Platform-specific file import directive so the runtime actually loads KNOWNS.md.
	if relativePath == "CLAUDE.md" || relativePath == "GEMINI.md" {
		sb.WriteString("@KNOWNS.md\n\n")
	}

	sb.WriteString("**CRITICAL: You MUST read and follow `KNOWNS.md` in the repository root before doing any work. It is the canonical source of truth for all agent behavior in this project.**\n\n")
	sb.WriteString("## Canonical Guidance\n\n")
	sb.WriteString("- Knowns is the repository memory layer for humans and the AI-friendly working layer for agents.\n")
	sb.WriteString("- The source of truth for repo-level agent guidance is `KNOWNS.md`.\n")
	sb.WriteString("- Read `KNOWNS.md` first whenever the runtime supports reading repository files.\n")
	sb.WriteString("- Load behavior, memory policy, and workflow rules from `KNOWNS.md`; treat this file only as a compatibility entrypoint.\n")
	sb.WriteString("- If this file and `KNOWNS.md` differ, follow `KNOWNS.md`.\n\n")
	sb.WriteString("## Minimum Rules\n\n")
	sb.WriteString("- Use Knowns as the canonical system for tasks, docs, templates, and workflow state.\n")
	sb.WriteString("- Never manually edit Knowns-managed task or doc markdown.\n")
	sb.WriteString("- Search first, then read only relevant docs and code.\n")
	sb.WriteString("- Use `search` for discovery; use MCP `retrieve` tool when a workflow needs structured context with citations. Fall back to CLI `knowns retrieve` if MCP is unavailable.\n")
	sb.WriteString("- For code context retrieval, prefer MCP tools over CLI: use `code({ action: \"search\" })` first, then `code({ action: \"symbols\" })`, then `code({ action: \"deps\" })`. Treat CLI `knowns code ...` as fallback for manual inspection or debugging.\n")
	sb.WriteString("- Plan before implementation unless the user explicitly overrides that workflow.\n")
	sb.WriteString("- Validate before considering work complete.\n")
	sb.WriteString("- Use memory tools: `memory({ action: \"list\" })` at session start, `memory({ action: \"add\" })` after tasks for reusable knowledge.\n")
	sb.WriteString("- Proactively capture durable memory based on `KNOWNS.md` memory rules; do not wait for an explicit user instruction to save memory when scope and durability are clear.\n\n")
	sb.WriteString("## Quick Reference\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("knowns doc list --plain               # List docs\n")
	sb.WriteString("knowns task list --plain              # List tasks\n")
	sb.WriteString("knowns task <id> --plain              # View task\n")
	sb.WriteString("knowns doc \"<path>\" --plain --smart  # View doc\n")
	sb.WriteString("knowns search \"query\" --plain        # Search docs/tasks\n")
	sb.WriteString("knowns retrieve \"query\" --json      # Retrieve structured context pack (CLI fallback)\n")
	sb.WriteString("knowns guidelines --plain             # Full workflow reference\n")
	sb.WriteString("```\n\n")
	sb.WriteString("<!-- KNOWNS GUIDELINES END -->\n")
	return sb.String()
}

func compatibilityInstructionTitle(relativePath, platform, projectName string) string {
	switch relativePath {
	case "AGENTS.md":
		return "AGENTS"
	case "CLAUDE.md":
		return "CLAUDE"
	case "GEMINI.md":
		return "GEMINI"
	case "OPENCODE.md":
		return "OPENCODE"
	case filepath.Join(".github", "copilot-instructions.md"):
		return projectName + " - GitHub Copilot Instructions"
	default:
		return platform
	}
}

// isGitRepo checks if the given directory (or any parent) is a git repository.
func isGitRepo(dir string) bool {
	d := dir
	for {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return false
}

const (
	knownsGitignoreBegin = "# >>> KNOWNS >>>"
	knownsGitignoreEnd   = "# <<< KNOWNS <<<"
)

// writeKnownsGitignore replaces the marker-delimited Knowns block in
// .gitignore with the correct rules for the given git tracking mode.
// If no marker block exists yet, it appends one.
// This cleanly handles mode switching on re-init.
func writeKnownsGitignore(dir, mode string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	existing := ""
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		existing = string(data)
	}

	// Strip the old marker block (everything between BEGIN and END inclusive).
	var cleaned []string
	inside := false
	for _, line := range strings.Split(existing, "\n") {
		if strings.TrimSpace(line) == knownsGitignoreBegin {
			inside = true
			continue
		}
		if strings.TrimSpace(line) == knownsGitignoreEnd {
			inside = false
			continue
		}
		if !inside {
			cleaned = append(cleaned, line)
		}
	}

	// Trim trailing blank lines.
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	// Build new marker block when this mode manages .gitignore.
	var block []string
	switch mode {
	case "git-tracked":
		// Track all .knowns/ content; remove any managed block.
	case "git-ignored":
		block = append(block,
			knownsGitignoreBegin,
			"# Managed by Knowns CLI — do not edit manually.",
			"# Run 'knowns init' to regenerate this block.",
		)
		block = append(block,
			".knowns/*",
			"!.knowns/config.json",
			"!.knowns/docs/",
			"!.knowns/docs/**",
			"!.knowns/templates/",
			"!.knowns/templates/**",
		)
		block = append(block, knownsGitignoreEnd)
	case "none":
		// Leave .gitignore unmanaged; remove any managed block.
	}

	// Assemble final content.
	var parts []string
	if len(cleaned) > 0 {
		parts = append(parts, strings.Join(cleaned, "\n"))
	}
	if len(block) > 0 {
		parts = append(parts, strings.Join(block, "\n"))
	}

	content := ""
	if len(parts) > 0 {
		content = strings.Join(parts, "\n\n") + "\n"
	}
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

// addToGitignore appends an entry to .gitignore if not already present.
func addToGitignore(dir, entry string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	existing := ""
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		existing = string(data)
	}

	for _, line := range strings.Split(existing, "\n") {
		if strings.TrimSpace(line) == strings.TrimSpace(entry) {
			return nil
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if existing != "" && !strings.HasSuffix(existing, "\n") {
		_, _ = f.WriteString("\n")
	}
	_, err = f.WriteString(entry + "\n")
	return err
}

// maybeOpenBrowser optionally launches the Chat UI after init.
//
//   - --no-open or --no-wizard: skip silently
//   - --open: launch immediately without prompting
//   - default (interactive): show a confirm prompt
//
// maybeOpenBrowser launches the Chat UI only when --open is passed explicitly.
// Default behavior (no flag) is to do nothing — users follow the printed hint instead.
func maybeOpenBrowser(cwd string, openFlag, noOpen bool) error {
	if noOpen || !openFlag {
		return nil
	}

	root := filepath.Join(cwd, ".knowns")
	store := storage.NewStore(root)
	port := 3001
	if cfg, err := store.Config.Load(); err == nil && cfg.Settings.ServerPort != 0 {
		port = cfg.Settings.ServerPort
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	go openBrowser(url)

	srv := server.NewServer(store, cwd, port, server.Options{})
	fmt.Printf("  %s  %s\n", StyleInfo.Render("→"), StyleBold.Render(url))
	fmt.Println()
	return srv.Start()
}

func init() {
	initCmd.Flags().Bool("git-tracked", false, "Track .knowns/ files in git")
	initCmd.Flags().Bool("git-ignored", false, "Add .knowns/ to .gitignore")
	initCmd.Flags().Bool("wizard", false, "Run interactive setup wizard")
	initCmd.Flags().Bool("no-wizard", false, "Skip interactive prompts, use defaults")
	initCmd.Flags().BoolP("force", "f", false, "Force reinitialize even if already initialized")
	initCmd.Flags().Bool("open", false, "Launch Chat UI immediately after init")
	initCmd.Flags().Bool("no-open", false, "Skip the Chat UI launch prompt after init")

	rootCmd.AddCommand(initCmd)
}
