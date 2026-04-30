package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	// Determine project root: --project flag > KNOWNS_PROJECT env > cwd walk-up
	projectRoot, _ := cmd.Flags().GetString("project")
	if projectRoot == "" {
		projectRoot = os.Getenv("KNOWNS_PROJECT")
	}

	s := mcp.NewMCPServer(projectRoot)
	return s.Start()
}

// --- mcp setup ---

// mcpPlatform describes a supported MCP platform and how to configure it.
type mcpPlatform struct {
	id    string // canonical lowercase name used as CLI argument
	label string // display name
	scope string // "project" or "global"
	// setup returns the config file path and any error.
	// projectRoot is the current working directory (needed for project-level and some global configs).
	setup func(projectRoot string) (path string, err error)
}

// mcpPlatforms is the registry of all supported platforms.
// Order determines display order.
var mcpPlatforms = []mcpPlatform{
	// Project-level
	{id: "claude", label: "Claude Code", scope: "project", setup: func(root string) (string, error) {
		return filepath.Join(root, ".mcp.json"), createMCPJsonFileQuiet(root, true)
	}},
	{id: "kiro", label: "Kiro", scope: "project", setup: func(root string) (string, error) {
		return filepath.Join(root, ".kiro", "settings", "mcp.json"), createKiroMCPConfigQuiet(root)
	}},
	{id: "opencode", label: "OpenCode", scope: "project", setup: func(root string) (string, error) {
		return filepath.Join(root, "opencode.json"), createOpenCodeConfigQuiet(root)
	}},
	{id: "cursor", label: "Cursor", scope: "project", setup: func(root string) (string, error) {
		return filepath.Join(root, ".cursor", "mcp.json"), createCursorMCPConfigQuiet(root)
	}},
	{id: "codex", label: "Codex", scope: "project", setup: func(root string) (string, error) {
		return filepath.Join(root, ".codex", "config.toml"), createCodexMCPConfigQuiet(root)
	}},
	{id: "cline", label: "Cline", scope: "project", setup: setupClineMCP},
	{id: "continue", label: "Continue", scope: "project", setup: setupContinueMCP},
	// Global
	{id: "claude-desktop", label: "Claude Desktop", scope: "global", setup: func(_ string) (string, error) {
		return setupClaudeDesktopMCP()
	}},
	{id: "antigravity", label: "Antigravity", scope: "global", setup: func(root string) (string, error) {
		return setupAntigravityMCP(root)
	}},
	{id: "gemini", label: "Gemini CLI", scope: "global", setup: func(_ string) (string, error) {
		return setupGeminiCLIMCP()
	}},
}

// mcpPlatformIDs returns sorted list of all platform IDs.
func mcpPlatformIDs() []string {
	ids := make([]string, len(mcpPlatforms))
	for i, p := range mcpPlatforms {
		ids[i] = p.id
	}
	return ids
}

// mcpPlatformByID looks up a platform by its canonical ID.
func mcpPlatformByID(id string) *mcpPlatform {
	id = strings.ToLower(id)
	for i := range mcpPlatforms {
		if mcpPlatforms[i].id == id {
			return &mcpPlatforms[i]
		}
	}
	return nil
}

var mcpSetupCmd = &cobra.Command{
	Use:   "setup [platforms...]",
	Short: "Set up MCP configuration for AI assistants",
	Long: `Configure the Knowns MCP server for AI assistants.

Usage:
  knowns mcp setup                  Set up all platforms
  knowns mcp setup claude kiro      Set up specific platforms only
  knowns mcp setup --project        Set up all project-level platforms
  knowns mcp setup --global         Set up all global platforms

Available platforms:
  Project-level:
    claude           Claude Code (.mcp.json)
    kiro             Kiro (.kiro/settings/mcp.json)
    opencode         OpenCode (opencode.json)
    cursor           Cursor (.cursor/mcp.json)
    codex            Codex (.codex/config.toml)
    cline            Cline (.cline/mcp.json)
    continue         Continue (.continue/config.json)

  Global:
    claude-desktop   Claude Desktop (~/Library/Application Support/Claude/...)
    antigravity      Antigravity (~/.gemini/antigravity/mcp_config.json)
    gemini           Gemini CLI (~/.gemini/settings.json)`,
	ValidArgsFunction: mcpSetupValidArgs,
	RunE:              runMCPSetup,
}

// mcpSetupValidArgs provides shell completion for platform names.
func mcpSetupValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Exclude already-specified platforms
	used := make(map[string]bool, len(args))
	for _, a := range args {
		used[strings.ToLower(a)] = true
	}

	var completions []string
	for _, p := range mcpPlatforms {
		if !used[p.id] {
			completions = append(completions, fmt.Sprintf("%s\t%s (%s)", p.id, p.label, p.scope))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// mcpSetupResult tracks what was created/updated during setup.
type mcpSetupResult struct {
	platform string
	path     string
	err      error
}

func runMCPSetup(cmd *cobra.Command, args []string) error {
	projectOnly, _ := cmd.Flags().GetBool("project")
	globalOnly, _ := cmd.Flags().GetBool("global")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Determine which platforms to set up
	var targets []mcpPlatform

	if len(args) > 0 {
		// Explicit platform selection: knowns mcp setup claude kiro antigravity
		for _, arg := range args {
			p := mcpPlatformByID(arg)
			if p == nil {
				return fmt.Errorf("unknown platform %q\n\nAvailable platforms: %s",
					arg, strings.Join(mcpPlatformIDs(), ", "))
			}
			targets = append(targets, *p)
		}
	} else {
		// No args: use --project / --global flags, or all
		for _, p := range mcpPlatforms {
			switch {
			case projectOnly && p.scope == "project":
				targets = append(targets, p)
			case globalOnly && p.scope == "global":
				targets = append(targets, p)
			case !projectOnly && !globalOnly:
				targets = append(targets, p)
			}
		}
	}

	if len(targets) == 0 {
		fmt.Println(RenderHint("No platforms matched. Use 'knowns mcp setup --help' to see available platforms."))
		return nil
	}

	// Run setup for each target
	var results []mcpSetupResult
	for _, t := range targets {
		path, setupErr := t.setup(cwd)
		results = append(results, mcpSetupResult{
			platform: t.label,
			path:     path,
			err:      setupErr,
		})
	}

	// Print results
	var successCount, failCount int
	for _, r := range results {
		if r.err != nil {
			failCount++
			fmt.Println(RenderError(fmt.Sprintf("%s: %v", r.platform, r.err)))
		} else {
			successCount++
			fmt.Println(RenderSuccess(fmt.Sprintf("%s → %s", r.platform, r.path)))
		}
	}

	fmt.Println()
	if failCount > 0 {
		fmt.Println(RenderField("Result", fmt.Sprintf("%d configured, %d failed", successCount, failCount)))
	} else {
		fmt.Println(RenderField("Result", fmt.Sprintf("%d platform(s) configured", successCount)))
	}
	fmt.Println(RenderHint("Restart your AI assistant to load the new MCP server."))

	return nil
}

// --- Platform setup helpers ---

// setupClineMCP creates/updates .cline/mcp.json.
func setupClineMCP(projectRoot string) (string, error) {
	settingsDir := filepath.Join(projectRoot, ".cline")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", fmt.Errorf("create .cline: %w", err)
	}

	configPath := filepath.Join(settingsDir, "mcp.json")
	config := map[string]any{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return configPath, fmt.Errorf("parse .cline/mcp.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return configPath, err
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok || servers == nil {
		servers = make(map[string]any)
	}

	cmd, args := mcpCommand()
	servers["knowns"] = map[string]any{
		"command": cmd,
		"args":    args,
	}

	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return configPath, err
	}

	return configPath, os.WriteFile(configPath, append(data, '\n'), 0644)
}

// setupContinueMCP creates/updates .continue/config.json.
func setupContinueMCP(projectRoot string) (string, error) {
	settingsDir := filepath.Join(projectRoot, ".continue")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", fmt.Errorf("create .continue: %w", err)
	}

	configPath := filepath.Join(settingsDir, "config.json")
	config := map[string]any{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return configPath, fmt.Errorf("parse .continue/config.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return configPath, err
	}

	// Continue uses: experimental.modelContextProtocolServers
	experimental, ok := config["experimental"].(map[string]any)
	if !ok || experimental == nil {
		experimental = make(map[string]any)
	}

	cmd, args := mcpCommand()
	knownsServer := map[string]any{
		"name": "knowns",
		"transport": map[string]any{
			"type":    "stdio",
			"command": cmd,
			"args":    args,
		},
	}

	// Replace existing knowns entry or append
	serverList, _ := experimental["modelContextProtocolServers"].([]any)
	found := false
	for i, s := range serverList {
		if srv, ok := s.(map[string]any); ok {
			if name, _ := srv["name"].(string); name == "knowns" {
				serverList[i] = knownsServer
				found = true
				break
			}
		}
	}
	if !found {
		serverList = append(serverList, knownsServer)
	}

	experimental["modelContextProtocolServers"] = serverList
	config["experimental"] = experimental

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return configPath, err
	}

	return configPath, os.WriteFile(configPath, append(data, '\n'), 0644)
}

// setupClaudeDesktopMCP creates/updates the Claude Desktop global config.
func setupClaudeDesktopMCP() (string, error) {
	settingsPath := getClaudeDesktopConfigPath()

	var settings map[string]any
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
			return settingsPath, fmt.Errorf("parse existing config: %w", jsonErr)
		}
	} else {
		settings = make(map[string]any)
	}

	mcpServers, ok := settings["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	cmd, args := mcpCommand()
	mcpServers["knowns"] = map[string]any{
		"command": cmd,
		"args":    args,
	}

	settings["mcpServers"] = mcpServers

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return settingsPath, fmt.Errorf("create settings directory: %w", err)
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return settingsPath, fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, append(out, '\n'), 0644); err != nil {
		return settingsPath, fmt.Errorf("write config: %w", err)
	}

	return settingsPath, nil
}

// setupAntigravityMCP creates/updates ~/.gemini/antigravity/mcp_config.json.
func setupAntigravityMCP(projectRoot string) (string, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", fmt.Errorf("create antigravity config dir: %w", err)
	}

	configPath := filepath.Join(settingsDir, "mcp_config.json")
	config := map[string]any{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return configPath, fmt.Errorf("parse antigravity config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return configPath, err
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok || servers == nil {
		servers = make(map[string]any)
	}

	cmd, args := mcpCommand()
	// Antigravity is global — include --project so the server knows which project to use
	absRoot, absErr := filepath.Abs(projectRoot)
	if absErr == nil {
		args = append(args, "--project", absRoot)
	}
	servers["knowns"] = map[string]any{
		"command": cmd,
		"args":    args,
	}

	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return configPath, err
	}

	return configPath, os.WriteFile(configPath, append(data, '\n'), 0644)
}

// setupGeminiCLIMCP creates/updates ~/.gemini/settings.json.
func setupGeminiCLIMCP() (string, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	settingsDir := filepath.Join(home, ".gemini")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", fmt.Errorf("create .gemini dir: %w", err)
	}

	configPath := filepath.Join(settingsDir, "settings.json")
	config := map[string]any{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return configPath, fmt.Errorf("parse gemini settings: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return configPath, err
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok || servers == nil {
		servers = make(map[string]any)
	}

	cmd, args := mcpCommand()
	servers["knowns"] = map[string]any{
		"command": cmd,
		"args":    args,
	}

	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return configPath, err
	}

	return configPath, os.WriteFile(configPath, append(data, '\n'), 0644)
}

// getClaudeDesktopConfigPath returns the Claude Desktop config file path.
func getClaudeDesktopConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
	default:
		return filepath.Join(home, ".config", "claude", "claude_desktop_config.json")
	}
}

func init() {
	mcpCmd.Flags().Bool("stdio", false, "Use stdio transport (for MCP clients)")
	mcpCmd.Flags().String("project", "", "Project root directory (auto-detected from cwd if not set)")

	mcpSetupCmd.Flags().Bool("project", false, "Only set up project-level platforms")
	mcpSetupCmd.Flags().Bool("global", false, "Only set up global/user-level platforms")

	mcpCmd.AddCommand(mcpSetupCmd)

	rootCmd.AddCommand(mcpCmd)
}

