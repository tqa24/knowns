package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/howznguyen/knowns/internal/util"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Knowns CLI to the latest version and sync project configs",
	Long: `Update the Knowns CLI binary to the latest version, then sync the current
project's MCP configurations to use the local binary directly (instead of npx).

This command:
  1. Detects how Knowns was installed (Homebrew, npm, etc.)
  2. Runs the appropriate upgrade command
  3. Syncs MCP configs (.mcp.json, .kiro/settings/mcp.json) to use the local binary

Use --check to only check for updates without installing.`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")

	// 1. Check for latest version
	fmt.Println(StyleBold.Render("Checking for updates..."))
	latest := util.FetchLatestVersion()
	if latest == "" {
		return fmt.Errorf("could not reach the npm registry — check your network connection")
	}

	current := util.Version
	cmp := util.CompareVersions(latest, current)

	if cmp <= 0 {
		fmt.Printf("  %s Already on the latest version %s\n", RenderSuccess(""), StyleBold.Render("v"+current))
		// Still sync configs even if up to date
		if !checkOnly {
			return syncMCPConfigs()
		}
		return nil
	}

	fmt.Printf("  %s → %s available (current %s)\n",
		StyleWarning.Render("UPDATE"),
		StyleBold.Render("v"+latest),
		StyleDim.Render("v"+current),
	)

	if checkOnly {
		installCmd := util.DetectInstallCmd()
		fmt.Printf("\n  Run: %s\n", StyleInfo.Render(installCmd))
		return nil
	}

	// 2. Detect install method and run upgrade
	fmt.Println()
	if err := runUpgrade(); err != nil {
		return err
	}

	// 3. Sync MCP configs
	fmt.Println()
	return syncMCPConfigs()
}

// runUpgrade detects the install method and runs the appropriate upgrade command.
func runUpgrade() error {
	installCmd := util.DetectInstallCmd()
	fmt.Printf("%s Running: %s\n", StyleBold.Render("Upgrading..."), StyleInfo.Render(installCmd))

	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return fmt.Errorf("could not determine upgrade command")
	}

	bin, err := exec.LookPath(parts[0])
	if err != nil {
		return fmt.Errorf("%s not found in PATH — install it first or upgrade manually", parts[0])
	}

	upgrade := exec.Command(bin, parts[1:]...)
	upgrade.Stdout = os.Stdout
	upgrade.Stderr = os.Stderr
	upgrade.Stdin = os.Stdin

	if err := upgrade.Run(); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Println(RenderSuccess("✓ Upgrade complete."))
	return nil
}

// syncMCPConfigs updates MCP config files in the current project to use the
// local knowns binary instead of npx, for faster and more reliable startup.
func syncMCPConfigs() error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil // non-fatal
	}

	// Find project root by walking up
	projectRoot := ""
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".knowns")); err == nil {
			projectRoot = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if projectRoot == "" {
		return nil // not in a knowns project, skip
	}

	cmd, args := mcpCommand()
	updated := 0

	// Sync .mcp.json
	if n, err := syncMCPJsonFile(projectRoot, cmd, args); err == nil {
		updated += n
	}

	// Sync .kiro/settings/mcp.json
	if n, err := syncKiroMCPConfig(projectRoot, cmd, args); err == nil {
		updated += n
	}

	// Sync opencode.json
	if n, err := syncOpenCodeConfig(projectRoot, cmd, args); err == nil {
		updated += n
	}

	if updated > 0 {
		fmt.Printf("%s Synced %d MCP config(s) to use local binary.\n", RenderSuccess("✓"), updated)
	} else {
		fmt.Printf("%s MCP configs already up to date.\n", StyleDim.Render("·"))
	}

	return nil
}

// syncMCPJsonFile updates .mcp.json to use the direct binary.
// Returns 1 if updated, 0 if unchanged.
func syncMCPJsonFile(projectRoot, cmd string, args []string) (int, error) {
	mcpPath := filepath.Join(projectRoot, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		return 0, nil // file doesn't exist, skip
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, err
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		return 0, nil
	}

	knowns, ok := servers["knowns"].(map[string]any)
	if !ok {
		return 0, nil
	}

	// Check if already using direct binary
	if knowns["command"] == cmd {
		return 0, nil
	}

	knowns["command"] = cmd
	knowns["args"] = args

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(mcpPath, out, 0644); err != nil {
		return 0, err
	}

	fmt.Printf("  %s %s\n", StyleInfo.Render("synced"), ".mcp.json")
	return 1, nil
}

// syncKiroMCPConfig updates .kiro/settings/mcp.json to use the direct binary.
func syncKiroMCPConfig(projectRoot, cmd string, args []string) (int, error) {
	configPath := filepath.Join(projectRoot, ".kiro", "settings", "mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, nil
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, err
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		return 0, nil
	}

	knowns, ok := servers["knowns"].(map[string]any)
	if !ok {
		return 0, nil
	}

	if knowns["command"] == cmd {
		return 0, nil
	}

	knowns["command"] = cmd
	knowns["args"] = args

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(configPath, append(out, '\n'), 0644); err != nil {
		return 0, err
	}

	fmt.Printf("  %s %s\n", StyleInfo.Render("synced"), ".kiro/settings/mcp.json")
	return 1, nil
}

// syncOpenCodeConfig updates opencode.json MCP command to use the direct binary.
func syncOpenCodeConfig(projectRoot, cmd string, args []string) (int, error) {
	configPath := filepath.Join(projectRoot, "opencode.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, nil
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return 0, err
	}

	mcp, ok := config["mcp"].(map[string]any)
	if !ok {
		return 0, nil
	}

	knowns, ok := mcp["knowns"].(map[string]any)
	if !ok {
		return 0, nil
	}

	// OpenCode uses a flat command array
	flat := append([]string{cmd}, args...)
	existing, _ := knowns["command"].([]any)
	if len(existing) > 0 {
		first, _ := existing[0].(string)
		if first == cmd {
			return 0, nil // already using direct binary
		}
	}

	knowns["command"] = flat

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(configPath, append(out, '\n'), 0644); err != nil {
		return 0, err
	}

	fmt.Printf("  %s %s\n", StyleInfo.Render("synced"), "opencode.json")
	return 1, nil
}

func init() {
	updateCmd.Flags().Bool("check", false, "Only check for updates without installing")
	rootCmd.AddCommand(updateCmd)
}
