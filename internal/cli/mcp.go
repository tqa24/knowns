package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/howznguyen/knowns/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP (Model Context Protocol) server",
	Long: `Start the Knowns MCP server, which exposes project management tools
to AI agents via the Model Context Protocol.

Use --stdio to communicate over stdin/stdout (default for MCP clients).`,
	RunE:         runMCP,
	SilenceUsage: true,
}

func runMCP(cmd *cobra.Command, args []string) error {
	s := mcp.NewMCPServer()
	return s.Start()
}

// --- mcp setup ---

var mcpSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up MCP in Claude Code config",
	Long:  "Writes or updates the Claude Code MCP settings file to include the Knowns MCP server.",
	RunE:  runMCPSetup,
}

func runMCPSetup(cmd *cobra.Command, args []string) error {
	// Determine the Claude Code settings path
	settingsPath := getMCPSettingsPath()

	// Read existing settings or create new
	var settings map[string]any
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return fmt.Errorf("parse existing MCP settings: %w", jsonErr)
		}
	} else {
		settings = make(map[string]any)
	}

	// Ensure mcpServers map exists
	mcpServers, ok := settings["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	// Find the knowns binary path
	execPath, err := os.Executable()
	if err != nil {
		execPath = "knowns"
	}

	// Add/update the knowns server entry
	mcpServers["knowns"] = map[string]any{
		"command": execPath,
		"args":    []string{"mcp", "--stdio"},
	}

	settings["mcpServers"] = mcpServers

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	// Write updated settings
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("write MCP settings: %w", err)
	}

	fmt.Printf("MCP setup complete.\n")
	fmt.Printf("Settings file: %s\n", settingsPath)
	fmt.Println("Knowns MCP server has been configured for Claude Code.")
	return nil
}

// getMCPSettingsPath returns the default Claude Code MCP settings file path.
func getMCPSettingsPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
	default:
		return filepath.Join(home, ".claude", "claude_desktop_config.json")
	}
}

func init() {
	mcpCmd.Flags().Bool("stdio", false, "Use stdio transport (for MCP clients)")

	mcpCmd.AddCommand(mcpSetupCmd)

	rootCmd.AddCommand(mcpCmd)
}
