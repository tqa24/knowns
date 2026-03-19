package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/howznguyen/knowns/internal/codegen"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync skills and agent instruction files",
	Long: `Sync skills and agent instruction files to their platform-specific locations.

This command copies embedded built-in skills from internal/instructions/skills/ to the
appropriate platform directories and generates/updates agent instruction files
(KNOWNS.md, CLAUDE.md, GEMINI.md, AGENTS.md, .github/copilot-instructions.md).`,
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
	platform, _ := cmd.Flags().GetString("platform")

	// If neither flag is set, sync both
	syncSkills := !instructionsOnly || skillsOnly
	syncInstructions := !skillsOnly || instructionsOnly

	// Derive the project root directory (parent of .knowns/)
	projectRoot := filepath.Dir(store.Root)

	synced := 0

	// Load enabled platforms from project config (empty = all).
	var configPlatforms []string
	if cfg, err := store.Config.Load(); err == nil {
		configPlatforms = cfg.Settings.Platforms
	}

	if syncSkills {
		if err := runSyncSkillsForPlatforms(projectRoot, force, configPlatforms); err != nil {
			return fmt.Errorf("sync skills: %w", err)
		}
		synced++
	}

	if syncInstructions {
		// --platform flag overrides config; if neither set use config platforms.
		effectivePlatform := platform
		if err := runSyncInstructions(projectRoot, effectivePlatform, force, configPlatforms); err != nil {
			return fmt.Errorf("sync instructions: %w", err)
		}
		synced++
	}

	if synced > 0 {
		fmt.Println(RenderSuccess("Sync complete."))
	}
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

	fmt.Printf("Syncing %s skill(s)...\n", StyleBold.Render(fmt.Sprintf("%d", count)))

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
	syncCmd.Flags().String("platform", "", "Sync specific platform (claude, gemini, copilot, agents)")

	rootCmd.AddCommand(syncCmd)
}
