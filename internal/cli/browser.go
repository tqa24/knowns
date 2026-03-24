package cli

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/howznguyen/knowns/internal/registry"
	"github.com/howznguyen/knowns/internal/server"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/howznguyen/knowns/internal/util"
)

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Launch the Knowns web UI",
	Long:  "Start the Knowns HTTP server and optionally open it in a browser.\nCan be launched outside a repo to use the workspace picker.",
	RunE:  runBrowser,
}

// resolveProject determines which project to open using a fallback chain:
//  1. --project <path> flag
//  2. --scan <dirs> flag (pre-populate registry)
//  3. cwd-based .knowns/ discovery
//  4. Registry last-active project
//  5. Picker mode (nil store)
func resolveProject(cmd *cobra.Command) (store *storage.Store, projectRoot string) {
	projectFlag, _ := cmd.Flags().GetString("project")
	scanFlag, _ := cmd.Flags().GetString("scan")

	// 1. Explicit --project flag
	if projectFlag != "" {
		absPath, err := filepath.Abs(projectFlag)
		if err == nil {
			knDir := filepath.Join(absPath, ".knowns")
			store = storage.NewStore(knDir)
			projectRoot = absPath
			return
		}
	}

	// 2. --scan flag: pre-populate registry before resolution
	if scanFlag != "" {
		dirs := strings.Split(scanFlag, ",")
		for i := range dirs {
			dirs[i] = strings.TrimSpace(dirs[i])
		}
		reg := registry.NewRegistry()
		if err := reg.Load(); err == nil {
			added, _ := reg.Scan(dirs)
			if len(added) > 0 {
				fmt.Printf("  %s  Discovered %d project(s)\n", StyleInfo.Render("⊕"), len(added))
			}
		}
	}

	// 3. Try cwd-based discovery
	s, err := getStoreErr()
	if err == nil {
		store = s
		projectRoot = filepath.Dir(s.Root)
		return
	}

	// 4. Fallback to registry last-active
	reg := registry.NewRegistry()
	if err := reg.Load(); err == nil {
		if active := reg.GetActive(); active != nil {
			knDir := filepath.Join(active.Path, ".knowns")
			store = storage.NewStore(knDir)
			projectRoot = active.Path
			fmt.Printf("  %s  Using last-active project: %s\n", StyleInfo.Render("↩"), StyleBold.Render(active.Name))
			return
		}
	}

	// 5. Picker mode — no project found
	return nil, ""
}

func runBrowser(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	openFlag, _ := cmd.Flags().GetBool("open")
	noOpen, _ := cmd.Flags().GetBool("no-open")
	restart, _ := cmd.Flags().GetBool("restart")
	dev, _ := cmd.Flags().GetBool("dev")

	store, projectRoot := resolveProject(cmd)

	// Resolve port from config (only if we have a store).
	if port == 0 && store != nil {
		cfg, cerr := store.Config.Load()
		if cerr == nil && cfg.Settings.ServerPort != 0 {
			port = cfg.Settings.ServerPort
		}
	}
	if port == 0 {
		port = 3001
	}

	// Auto-register this project in the global registry (only if we have a project).
	if store != nil {
		reg := registry.NewRegistry()
		if err := reg.Load(); err == nil {
			if p, err := reg.Add(projectRoot); err == nil {
				_ = reg.SetActive(p.ID)
			}
		}
	}

	// Handle restart: attempt to stop existing server first
	if restart {
		fmt.Printf("%s Attempting to stop existing server on port %d...\n", StyleWarning.Render("↻"), port)
		stopExistingServer(port)
	}

	// Determine whether to open browser: --open enables, --no-open disables
	shouldOpen := openFlag && !noOpen
	if shouldOpen {
		go func() {
			url := fmt.Sprintf("http://localhost:%d", port)
			openBrowser(url)
		}()
	}

	srv := server.NewServer(store, projectRoot, port, server.Options{Dev: dev})

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Println()
	fmt.Printf("  %s  %s %s\n", StyleSuccess.Render("●"), StyleBold.Render("Knowns"), StyleDim.Render("v"+util.Version))
	fmt.Println()
	fmt.Printf("  %s  %s\n", StyleInfo.Render("→"), StyleBold.Render(url))
	if projectRoot != "" {
		fmt.Printf("  %s  %s\n", StyleDim.Render("⌁"), StyleDim.Render(projectRoot))
	} else {
		fmt.Printf("  %s  %s\n", StyleWarning.Render("◇"), StyleDim.Render("No project — workspace picker mode"))
	}
	fmt.Println()

	return srv.Start()
}

// stopExistingServer sends a shutdown request to any existing server on the
// given port and waits for the port to be released. Returns true if the port
// was freed, false if no server was found or the stop timed out.
func stopExistingServer(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/api/shutdown", port),
		"application/json", nil,
	)
	if err != nil {
		fmt.Println(StyleDim.Render("No existing server found."))
		return false
	}
	resp.Body.Close()

	fmt.Println(StyleWarning.Render("Existing server detected.") + " Waiting for shutdown...")

	// Poll until the port is released (max ~3s).
	for i := 0; i < 10; i++ {
		conn, dialErr := net.DialTimeout("tcp",
			fmt.Sprintf("localhost:%d", port), 200*time.Millisecond)
		if dialErr != nil {
			fmt.Println(StyleSuccess.Render("Previous server stopped."))
			return true // Port released
		}
		conn.Close()
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println(StyleWarning.Render("Timed out waiting for previous server to stop."))
	return false
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func init() {
	browserCmd.Flags().Int("port", 0, "HTTP server port (default: 3001 or config value)")
	browserCmd.Flags().Bool("open", false, "Open browser after starting")
	browserCmd.Flags().Bool("no-open", false, "Don't automatically open browser")
	browserCmd.Flags().Bool("restart", false, "Restart server if already running")
	browserCmd.Flags().Bool("dev", false, "Enable development mode (verbose logging)")
	browserCmd.Flags().String("project", "", "Project path to open directly")
	browserCmd.Flags().String("scan", "", "Comma-separated directories to scan for projects")

	rootCmd.AddCommand(browserCmd)
}
