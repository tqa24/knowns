package cli

import (
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/howznguyen/knowns/internal/server"
	"github.com/howznguyen/knowns/internal/util"
)

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Launch the Knowns web UI",
	Long:  "Start the Knowns HTTP server and optionally open it in a browser.",
	RunE:  runBrowser,
}

func runBrowser(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	openFlag, _ := cmd.Flags().GetBool("open")
	noOpen, _ := cmd.Flags().GetBool("no-open")
	restart, _ := cmd.Flags().GetBool("restart")
	dev, _ := cmd.Flags().GetBool("dev")

	store := getStore()

	if port == 0 {
		cfg, cerr := store.Config.Load()
		if cerr == nil && cfg.Settings.ServerPort != 0 {
			port = cfg.Settings.ServerPort
		}
	}
	if port == 0 {
		port = 3001
	}

	// store.Root is the .knowns/ directory; the project root is its parent.
	projectRoot := filepath.Dir(store.Root)

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
	fmt.Printf("  %s  %s\n", StyleDim.Render("⌁"), StyleDim.Render(projectRoot))
	fmt.Println()

	return srv.Start()
}

// stopExistingServer attempts to stop any existing server on the given port.
func stopExistingServer(port int) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		fmt.Println(StyleDim.Render("No existing server found."))
		return
	}
	resp.Body.Close()
	fmt.Println(StyleWarning.Render("Existing server detected.") + " It will be replaced when the new server starts.")
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

	rootCmd.AddCommand(browserCmd)
}
