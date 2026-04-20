package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/howznguyen/knowns/internal/runtimequeue"
	"github.com/howznguyen/knowns/internal/util"
	"github.com/spf13/cobra"
)

const updateDownloadTimeout = 30 * time.Second

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
			return runSync(syncCmd, nil)
		}
		return nil
	}

	fmt.Printf("  %s → %s available (current %s)\n",
		StyleWarning.Render("UPDATE"),
		StyleBold.Render("v"+latest),
		StyleDim.Render("v"+current),
	)

	if checkOnly {
		fmt.Printf("\n  Run: %s\n", StyleInfo.Render(recommendedUpdateCommand()))
		return nil
	}

	// 2. Detect install method and run upgrade
	fmt.Println()
	if err := runUpgrade(); err != nil {
		return err
	}

	// 3. Full sync (skills, instructions, model, search index, MCP configs)
	fmt.Println()
	return runSync(syncCmd, nil)
}

// runUpgrade detects the install method and runs the appropriate upgrade command.
func runUpgrade() error {
	meta, err := util.LoadInstallMetadata()
	if err != nil {
		return fmt.Errorf("read install metadata: %w", err)
	}
	if meta == nil {
		meta = inferScriptInstallMetadata()
	}
	if meta != nil && meta.IsScriptManaged() {
		return runScriptManagedUpgrade(meta)
	}

	installCmd := util.DetectInstallCmd()
	if isHomebrewInstallCommand(installCmd) {
		return runHomebrewUpgrade(installCmd)
	}
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

	fmt.Println(StyleSuccess.Render("✓") + " Upgrade complete.")
	return nil
}

func recommendedUpdateCommand() string {
	meta, err := util.LoadInstallMetadata()
	if err == nil && meta != nil && meta.IsScriptManaged() {
		return "knowns update"
	}
	installCmd := util.DetectInstallCmd()
	if isHomebrewInstallCommand(installCmd) {
		return "brew update && HOMEBREW_NO_AUTO_UPDATE=1 " + installCmd
	}
	return installCmd
}

func isHomebrewInstallCommand(cmd string) bool {
	return strings.HasPrefix(strings.TrimSpace(cmd), "brew ")
}

func runHomebrewUpgrade(installCmd string) error {
	updateBin, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("brew not found in PATH — install it first or upgrade manually")
	}

	updateCmd := exec.Command(updateBin, "update")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	updateCmd.Stdin = os.Stdin
	fmt.Printf("%s Running: %s\n", StyleBold.Render("Refreshing Homebrew..."), StyleInfo.Render("brew update"))
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("brew update failed: %w", err)
	}

	parts := strings.Fields(installCmd)
	if len(parts) == 0 {
		return fmt.Errorf("could not determine upgrade command")
	}
	fmt.Printf("%s Running: %s\n", StyleBold.Render("Upgrading..."), StyleInfo.Render("HOMEBREW_NO_AUTO_UPDATE=1 "+installCmd))
	upgrade := exec.Command(updateBin, parts[1:]...)
	upgrade.Stdout = os.Stdout
	upgrade.Stderr = os.Stderr
	upgrade.Stdin = os.Stdin
	upgrade.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1")
	if err := upgrade.Run(); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Println(StyleSuccess.Render("✓") + " Upgrade complete.")
	return nil
}

func runScriptManagedUpgrade(meta *util.InstallMetadata) error {
	if meta == nil {
		return fmt.Errorf("missing install metadata for script-managed update")
	}
	binaryPath := strings.TrimSpace(meta.BinaryPath)
	if binaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve binary path: %w", err)
		}
		binaryPath = exe
	}
	if !isUserWritable(binaryPath) {
		return fmt.Errorf("script-managed install at %s is not writable by the current user; reinstall to ~/.knowns/bin or set KNOWNS_INSTALL_DIR to a user-writable path", binaryPath)
	}
	version := util.NormalizeVersionTag(util.FetchLatestVersion())
	if version == "" {
		return fmt.Errorf("could not determine latest version")
	}
	url, err := releaseArtifactURL(version)
	if err != nil {
		return err
	}
	fmt.Printf("%s Running: %s\n", StyleBold.Render("Upgrading..."), StyleInfo.Render(url))
	if err := downloadAndReplaceBinary(url, binaryPath); err != nil {
		return err
	}
	meta.Method = "script"
	if meta.ManagedBy == "" {
		meta.ManagedBy = "knowns-script"
	}
	meta.UpdateStrategy = "self-update"
	meta.BinaryPath = binaryPath
	meta.Version = strings.TrimPrefix(version, "v")
	if meta.Channel == "" {
		meta.Channel = "stable"
	}
	if err := util.SaveInstallMetadata(meta); err != nil {
		return fmt.Errorf("persist install metadata: %w", err)
	}
	if err := restartRuntimeIfNeeded(version); err != nil {
		return err
	}
	fmt.Println(StyleSuccess.Render("✓") + " Upgrade complete.")
	return nil
}

func inferScriptInstallMetadata() *util.InstallMetadata {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	defaultDir := filepath.Join(home, ".knowns", "bin")
	exeDir := filepath.Dir(exe)
	if !samePath(exeDir, defaultDir) {
		return nil
	}
	return &util.InstallMetadata{
		Method:         "script",
		ManagedBy:      "knowns-script",
		UpdateStrategy: "self-update",
		Channel:        "stable",
		BinaryPath:     exe,
		Version:        util.Version,
	}
}

func samePath(a, b string) bool {
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(cleanA, cleanB)
	}
	return cleanA == cleanB
}

func isUserWritable(path string) bool {
	dir := filepath.Dir(path)
	probe := filepath.Join(dir, ".knowns-write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0644); err != nil {
		return false
	}
	_ = os.Remove(probe)
	return true
}

func releaseArtifactURL(version string) (string, error) {
	platform, err := releasePlatform()
	if err != nil {
		return "", err
	}
	archive := fmt.Sprintf("knowns-%s.tar.gz", platform)
	return fmt.Sprintf("https://github.com/knowns-dev/knowns/releases/download/%s/%s", version, archive), nil
}

func releasePlatform() (string, error) {
	var osName string
	switch runtime.GOOS {
	case "darwin":
		osName = "darwin"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "win"
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
	return osName + "-" + arch, nil
}

func downloadAndReplaceBinary(url, binaryPath string) error {
	tmpDir, err := os.MkdirTemp("", "knowns-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, filepath.Base(url))
	if err := downloadFile(url, archivePath); err != nil {
		return fmt.Errorf("download release artifact: %w", err)
	}
	if err := extractTarGz(archivePath, tmpDir); err != nil {
		return fmt.Errorf("extract release artifact: %w", err)
	}
	binaryName := "knowns"
	if runtime.GOOS == "windows" {
		binaryName = "knowns.exe"
	}
	extractedPath, err := findFile(tmpDir, func(path string, info os.FileInfo) bool {
		name := strings.ToLower(info.Name())
		return !info.IsDir() && (name == strings.ToLower(binaryName) || strings.HasPrefix(name, "knowns-"))
	})
	if err != nil {
		return err
	}
	if err := replaceBinary(extractedPath, binaryPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: updateDownloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func extractTarGz(archivePath, destDir string) error {
	tarBin, err := exec.LookPath("tar")
	if err != nil {
		return fmt.Errorf("tar not found in PATH")
	}
	cmd := exec.Command(tarBin, "-xzf", archivePath, "-C", destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findFile(root string, match func(string, os.FileInfo) bool) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return err
		}
		if match(path, info) {
			found = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("updated binary not found in release artifact")
	}
	return found, nil
}

func replaceBinary(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(dest + ".old")
		if _, err := os.Stat(dest); err == nil {
			if err := os.Rename(dest, dest+".old"); err != nil {
				_ = os.Remove(tmp)
				return err
			}
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	return nil
}

func restartRuntimeIfNeeded(targetVersion string) error {
	status, err := runtimequeue.LoadStatus()
	if err != nil {
		return nil
	}
	if !status.Running {
		return nil
	}
	targetVersion = strings.TrimPrefix(targetVersion, "v")
	if status.Version == "" || strings.TrimPrefix(status.Version, "v") == targetVersion {
		return nil
	}
	if err := runtimequeue.RequestShutdown(5 * time.Second); err == nil {
		return nil
	}
	pidFile := runtimequeue.PIDFile()
	if pid, readErr := os.ReadFile(pidFile); readErr == nil {
		if parsed := strings.TrimSpace(string(pid)); parsed != "" {
			if process, findErr := os.FindProcess(mustAtoi(parsed)); findErr == nil {
				_ = process.Kill()
			}
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !runtimequeue.IsRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for shared runtime to stop after update")
}

func mustAtoi(value string) int {
	n, _ := strconv.Atoi(value)
	return n
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
		fmt.Printf("%s Synced %d MCP config(s) to use local binary.\n", StyleSuccess.Render("✓"), updated)
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
