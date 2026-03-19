package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/howznguyen/knowns/internal/codegen"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/howznguyen/knowns/internal/util"
)

var bannerLines = []string{
	"▄▄▄   ▄▄▄ ▄▄▄    ▄▄▄   ▄▄▄▄▄   ▄▄▄▄  ▄▄▄  ▄▄▄▄ ▄▄▄    ▄▄▄  ▄▄▄▄▄▄▄",
	"███ ▄███▀ ████▄  ███ ▄███████▄ ▀███  ███  ███▀ ████▄  ███ █████▀▀▀",
	"███████   ███▀██▄███ ███   ███  ███  ███  ███  ███▀██▄███  ▀████▄",
	"███▀███▄  ███  ▀████ ███▄▄▄███  ███▄▄███▄▄███  ███  ▀████    ▀████",
	"███  ▀███ ███    ███  ▀█████▀    ▀████▀████▀   ███    ███ ███████▀",
}

var rootCmd = &cobra.Command{
	Use:     "knowns [options] [command]",
	Short:   "The memory layer for AI-native software development",
	Version: util.Version,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()
		for _, line := range bannerLines {
			fmt.Println(StyleInfo.Render(line))
		}
		fmt.Println()
		fmt.Printf("  %s %s\n", StyleBold.Render("Knowns"), StyleDim.Render("v"+util.Version))
		fmt.Println(StyleDim.Render("  The memory layer for AI-native software development."))
		fmt.Println(StyleDim.Render("  Enabling AI to understand your project instantly."))
		fmt.Println()
		fmt.Println(StyleBold.Render("  Quick Start:"))
		fmt.Printf("    %s  %s\n", StyleInfo.Render("knowns init"), StyleDim.Render("Initialize project"))
		fmt.Printf("    %s  %s\n", StyleInfo.Render("knowns task list"), StyleDim.Render("List all tasks"))
		fmt.Printf("    %s  %s\n", StyleInfo.Render("knowns browser"), StyleDim.Render("Open web UI"))
		fmt.Printf("    %s  %s\n", StyleInfo.Render("knowns --help"), StyleDim.Render("Show all commands"))
		fmt.Println()
		fmt.Printf("  %s  %s\n", StyleDim.Render("Homepage: "), "https://knowns.sh")
		fmt.Printf("  %s  %s\n", StyleDim.Render("Documents:"), "https://knowns.sh/docs")
		fmt.Printf("  %s  %s\n", StyleDim.Render("Discord:  "), "https://discord.knowns.dev")
		fmt.Println()
	},
}

// customHelpFunc renders a clean, styled help output matching the TS CLI style.
func customHelpFunc(cmd *cobra.Command, args []string) {
	// Header
	fmt.Printf("%s %s\n", StyleBold.Render(cmd.Short), StyleDim.Render("(v"+util.Version+")"))
	fmt.Println()

	// Usage
	fmt.Printf("%s %s\n", StyleBold.Render("Usage:"), StyleInfo.Render(cmd.UseLine()))
	fmt.Println()

	// Commands - grouped
	if cmd.HasAvailableSubCommands() {
		fmt.Println(StyleBold.Render("Commands:"))

		// Find max command name length for alignment
		maxLen := 0
		for _, c := range cmd.Commands() {
			if !c.IsAvailableCommand() || c.Name() == "help" || c.Name() == "completion" {
				continue
			}
			if len(c.Name()) > maxLen {
				maxLen = len(c.Name())
			}
		}

		for _, c := range cmd.Commands() {
			if !c.IsAvailableCommand() || c.Name() == "help" || c.Name() == "completion" {
				continue
			}
			padding := strings.Repeat(" ", maxLen-len(c.Name())+2)
			fmt.Printf("  %s%s%s\n",
				StyleInfo.Render(c.Name()),
				padding,
				StyleDim.Render(c.Short),
			)
		}
		fmt.Println()
	}

	// Flags
	if cmd.HasAvailableLocalFlags() {
		fmt.Println(StyleBold.Render("Options:"))
		fmt.Println(StyleDim.Render(cmd.LocalFlags().FlagUsages()))
	}

	// Footer
	fmt.Printf("%s\n", StyleDim.Render("Use \"knowns [command] --help\" for more information about a command."))
}

// maybeAutoSync silently re-syncs skills when AutoSyncOnUpdate is enabled and
// the embedded skill version differs from the current CLI version.
// Errors are intentionally swallowed — this is a best-effort background operation.
func maybeAutoSync() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	root := filepath.Join(cwd, ".knowns")
	if _, err := os.Stat(root); err != nil {
		return // not a knowns project
	}

	store := storage.NewStore(root)
	cfg, err := store.Config.Load()
	if err != nil || cfg.Settings.AutoSyncOnUpdate == nil || !*cfg.Settings.AutoSyncOnUpdate {
		return
	}

	// Read synced version from .claude/skills/.version or .agent/skills/.version
	syncedVersion := codegen.ReadSyncedSkillVersion(cwd)
	if syncedVersion == "" || syncedVersion == util.Version {
		return
	}

	// Version mismatch — re-sync silently
	_ = codegen.SyncSkillsForPlatforms(cwd, cfg.Settings.Platforms)
}

// Execute runs the root command.
func Execute() error {
	// Auto-sync skills if version changed (best-effort, silent).
	maybeAutoSync()

	// Start update check in background while command runs
	msgCh := make(chan string, 1)
	go func() {
		msgCh <- util.CheckForUpdate()
	}()

	err := rootCmd.Execute()

	// After command completes, wait for update check (max 3s) and print if available
	select {
	case msg := <-msgCh:
		if msg != "" {
			fmt.Fprint(os.Stderr, msg)
		}
	case <-time.After(3 * time.Second):
	}

	return err
}

func init() {
	rootCmd.SetHelpFunc(customHelpFunc)
	rootCmd.PersistentFlags().Bool("plain", false, "Plain text output (for AI agents)")
	rootCmd.PersistentFlags().Bool("json", false, "JSON output")
	rootCmd.PersistentFlags().Bool("no-pager", false, "Disable TUI pager (print styled output directly)")
	rootCmd.PersistentFlags().Int("page", 0, "Page number for paginated output (e.g. --page 2)")
	rootCmd.PersistentFlags().Int("page-size", 0, "Lines per page (default 50)")
}
