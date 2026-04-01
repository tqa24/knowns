package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/howznguyen/knowns/internal/codegen"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync project from config.json (skills, instructions, model, search index)",
	Long: `Apply project configuration from .knowns/config.json.

This is the recommended command after cloning a repo with Knowns:
  git clone <repo>
  knowns sync

It reads config.json and sets up everything locally:
  • Skills — copies built-in skills to platform directories
  • Instructions — generates agent instruction files (KNOWNS.md, CLAUDE.md, etc.)
  • Model — downloads the configured embedding model (if not installed)
  • Search index — rebuilds the semantic search index
  • Git integration — applies .gitignore rules for the configured tracking mode

Use flags to sync only specific parts.`,
	RunE: runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	store, err := getStoreErr()
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	skillsOnly, _ := cmd.Flags().GetBool("skills")
	instructionsOnly, _ := cmd.Flags().GetBool("instructions")
	modelOnly, _ := cmd.Flags().GetBool("model")
	platform, _ := cmd.Flags().GetString("platform")

	// If no specific flag is set, sync everything
	specificFlag := skillsOnly || instructionsOnly || modelOnly
	syncSkills := !specificFlag || skillsOnly
	syncInstructions := !specificFlag || instructionsOnly
	syncModel := !specificFlag || modelOnly
	syncIndex := !specificFlag

	// Load project config
	cfg, err := store.Config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	projectRoot := filepath.Dir(store.Root)
	configPlatforms := cfg.Settings.Platforms

	fmt.Printf("%s\n\n", RenderInfo(fmt.Sprintf("Syncing project %s from config.json...", StyleBold.Render(cfg.Name))))

	// 1. Skills
	if syncSkills {
		if err := runSyncSkillsForPlatforms(projectRoot, force, configPlatforms); err != nil {
			return fmt.Errorf("sync skills: %w", err)
		}
		fmt.Println()
	}

	// 2. Instructions
	if syncInstructions {
		effectivePlatform := platform
		if err := runSyncInstructions(projectRoot, effectivePlatform, force, configPlatforms); err != nil {
			return fmt.Errorf("sync instructions: %w", err)
		}
		fmt.Println()
	}

	// 3. Git integration
	if !specificFlag {
		if cfg.Settings.GitTrackingMode != "" {
			fmt.Println(RenderField("Git tracking mode", StyleBold.Render(cfg.Settings.GitTrackingMode)))
			if err := writeKnownsGitignore(projectRoot, cfg.Settings.GitTrackingMode); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: git integration failed: %v\n", err)
			} else {
				fmt.Println(RenderSuccess("Git integration configured."))
			}
			fmt.Println()
		}
	}

	// 4. Model download
	if syncModel {
		if err := runSyncModel(store, force); err != nil {
			// Non-fatal — warn and continue
			fmt.Fprintf(os.Stderr, "Warning: model sync failed: %v\n", err)
			fmt.Println()
		}
	}

	// 5. Reindex
	if syncIndex {
		fmt.Println(StyleBold.Render("Rebuilding search index..."))
		if err := runReindex(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: reindex failed: %v\n", err)
		}
		fmt.Println()
	}

	fmt.Println(RenderSuccess("Sync complete."))
	return nil
}

// runSyncModel downloads the embedding model configured in config.json if not already installed.
func runSyncModel(store *storage.Store, force bool) error {
	cfg, err := store.Config.Load()
	if err != nil {
		return nil // no config, skip silently
	}

	if cfg.Settings.SemanticSearch == nil || !cfg.Settings.SemanticSearch.Enabled {
		fmt.Println(StyleDim.Render("Semantic search not enabled. Skipping model download."))
		return nil
	}

	modelID := cfg.Settings.SemanticSearch.Model
	if modelID == "" {
		return nil
	}

	var selected *embeddingModel
	for i := range supportedModels {
		if supportedModels[i].ID == modelID {
			selected = &supportedModels[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("unknown model %q in config", modelID)
	}

	if isModelInstalled(selected) && !force {
		fmt.Printf("%s Model %s already installed.\n", RenderSuccess("✓"), selected.Name)
		return nil
	}

	fmt.Printf("%s\n", RenderInfo(fmt.Sprintf("Downloading embedding model: %s (~%dMB)...", StyleBold.Render(selected.Name), selected.SizeMB)))

	steps, alreadyInstalled, err := buildSemanticDownloadSteps(modelID)
	if err != nil {
		return err
	}
	if alreadyInstalled {
		fmt.Printf("%s Model %s already installed.\n", RenderSuccess("✓"), selected.Name)
		return nil
	}

	if err := runInitSteps(steps); err != nil {
		return fmt.Errorf("model download failed: %w", err)
	}

	fmt.Println(RenderSuccess(fmt.Sprintf("Model %s installed.", selected.Name)))
	fmt.Println()
	return nil
}

// runSyncSkillsForPlatforms copies embedded built-in skills to the platform dirs
// determined by the given platform list (empty = all).
func runSyncSkillsForPlatforms(projectRoot string, force bool, platforms []string) error {
	count, err := codegen.BuiltInSkillCount()
	if err != nil {
		return fmt.Errorf("count built-in skills: %w", err)
	}
	if count == 0 {
		fmt.Println(StyleDim.Render("No embedded built-in skills found. Skipping skill sync."))
		return nil
	}

	fmt.Printf("%s\n", RenderInfo(fmt.Sprintf("Syncing %s skill(s)...", StyleBold.Render(fmt.Sprintf("%d", count)))))

	if err := codegen.SyncSkillsForPlatforms(projectRoot, platforms); err != nil {
		return err
	}

	if force {
		fmt.Printf("  %s\n", StyleWarning.Render("Force sync: overwritten all existing skill files."))
	}

	fmt.Println(RenderSuccess(fmt.Sprintf("Synced %d skill(s).", count)))
	return nil
}

// runSyncInstructions syncs the canonical guidance file and compatibility shims.
// platform filters to a single platform by name (overrides configPlatforms).
// configPlatforms restricts which platforms are active (empty = all).
func runSyncInstructions(projectRoot string, platform string, force bool, configPlatforms []string) error {
	// Define the known platforms and their instruction file paths
	type platformDef struct {
		name     string
		label    string
		filePath string
	}

	platforms := []platformDef{
		{name: "claude", label: "Claude Code", filePath: filepath.Join(projectRoot, "CLAUDE.md")},
		{name: "opencode", label: "OpenCode", filePath: filepath.Join(projectRoot, "OPENCODE.md")},
		{name: "gemini", label: "Gemini CLI", filePath: filepath.Join(projectRoot, "GEMINI.md")},
		{name: "copilot", label: "GitHub Copilot", filePath: filepath.Join(projectRoot, ".github", "copilot-instructions.md")},
		{name: "agents", label: "Generic AI", filePath: filepath.Join(projectRoot, "AGENTS.md")},
	}

	// configID maps sync platform name → config platform ID (allPlatformIDs format).
	configIDOf := map[string]string{
		"claude":   "claude-code",
		"opencode": "opencode",
		"gemini":   "gemini",
		"copilot":  "copilot",
		"agents":   "agents",
	}

	// Filter by --platform flag first (single platform override).
	if platform != "" {
		var filtered []platformDef
		for _, p := range platforms {
			if strings.EqualFold(p.name, platform) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("unknown platform %q (available: claude, opencode, gemini, copilot, agents)", platform)
		}
		platforms = filtered
	} else if len(configPlatforms) > 0 {
		// Apply config platform restriction when no explicit --platform flag.
		configSet := make(map[string]bool, len(configPlatforms))
		for _, id := range configPlatforms {
			configSet[id] = true
		}
		var filtered []platformDef
		for _, p := range platforms {
			if configSet[configIDOf[p.name]] {
				filtered = append(filtered, p)
			}
		}
		platforms = filtered
	}

	if err := writeInstructionFile(projectRoot, canonicalInstructionFile, "Knowns", force); err != nil {
		return err
	}

	fmt.Println(StyleBold.Render("Checking agent instruction files..."))

	for _, p := range platforms {
		exists := false
		if _, err := os.Stat(p.filePath); err == nil {
			exists = true
		}

		if exists && !force {
			fmt.Printf("  %s %s %s\n", StyleDim.Render("["+p.name+"]"), filepath.Base(p.filePath), StyleDim.Render("(use --force to overwrite)"))
			continue
		}

		if err := os.MkdirAll(filepath.Dir(p.filePath), 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", filepath.Dir(p.filePath), err)
		}

		relPath := filepath.Base(p.filePath)
		if p.filePath == filepath.Join(projectRoot, ".github", "copilot-instructions.md") {
			relPath = filepath.Join(".github", "copilot-instructions.md")
		}
		content := generateInstructionContent(relPath, p.label, projectRoot)
		if err := os.WriteFile(p.filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", p.filePath, err)
		}

		if exists && force {
			fmt.Printf("  %s %s %s\n", StyleSuccess.Render("["+p.name+"]"), filepath.Base(p.filePath), StyleWarning.Render("overwritten (force sync)."))
		} else {
			fmt.Printf("  %s %s %s\n", StyleWarning.Render("["+p.name+"]"), filepath.Base(p.filePath), StyleDim.Render("not found. Run 'knowns init' or create manually."))
		}
	}

	return nil
}

func init() {
	syncCmd.Flags().Bool("force", false, "Force resync (overwrite existing files)")
	syncCmd.Flags().Bool("skills", false, "Sync skills only")
	syncCmd.Flags().Bool("instructions", false, "Sync instruction files only")
	syncCmd.Flags().Bool("model", false, "Download embedding model only")
	syncCmd.Flags().String("platform", "", "Sync specific platform (claude, gemini, copilot, agents)")

	rootCmd.AddCommand(syncCmd)
}
