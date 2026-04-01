package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/howznguyen/knowns/internal/agents/opencode"
)

// openCodeInstallURL is the manual install reference.
const openCodeInstallURL = "https://opencode.ai"

// openCodeInstallOption describes an install method for a platform.
type openCodeInstallOption struct {
	Label   string
	Command string
	Args    []string
}

// getOpenCodeInstallOptions returns available install options for the current OS.
func getOpenCodeInstallOptions() []openCodeInstallOption {
	var opts []openCodeInstallOption

	switch runtime.GOOS {
	case "darwin":
		opts = append(opts, openCodeInstallOption{
			Label:   "curl (recommended)",
			Command: "bash",
			Args:    []string{"-c", "curl -fsSL https://opencode.ai/install | bash"},
		})
		// Only offer brew if available
		if _, err := exec.LookPath("brew"); err == nil {
			opts = append(opts, openCodeInstallOption{
				Label:   "Homebrew",
				Command: "brew",
				Args:    []string{"install", "anomalyco/tap/opencode"},
			})
		}
	case "linux":
		opts = append(opts, openCodeInstallOption{
			Label:   "curl (recommended)",
			Command: "bash",
			Args:    []string{"-c", "curl -fsSL https://opencode.ai/install | bash"},
		})
	case "windows":
		opts = append(opts, openCodeInstallOption{
			Label:   "npm",
			Command: "npm",
			Args:    []string{"install", "-g", "opencode-ai"},
		})
	}

	// npm fallback for all platforms
	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("npm"); err == nil {
			opts = append(opts, openCodeInstallOption{
				Label:   "npm (fallback)",
				Command: "npm",
				Args:    []string{"install", "-g", "opencode-ai"},
			})
		}
	}

	return opts
}

// maybeInstallOpenCode checks if OpenCode needs to be installed and guides the user.
// Returns nil even on install failure — init should continue regardless.
func maybeInstallOpenCode(force bool) error {
	status := opencode.DetectOpenCode()

	if status.Installed && status.Compatible && !force {
		fmt.Printf("  %s OpenCode v%s detected\n", RenderSuccess("✓"), status.Version)
		return nil
	}

	if status.Installed && !status.Compatible {
		fmt.Printf("  %s OpenCode v%s is outdated (requires >= %s)\n",
			warnStyle.Render("⚠"), status.Version, status.MinVersion)
	}

	// Ask user
	var wantInstall bool
	prompt := "Install OpenCode for Chat UI?"
	if status.Installed && !status.Compatible {
		prompt = "Update OpenCode for Chat UI?"
	}

	err := huh.NewConfirm().
		Title(prompt).
		Description("OpenCode powers the AI chat features.").
		Value(&wantInstall).
		Run()
	if err != nil || !wantInstall {
		fmt.Printf("  %s Skipped OpenCode install\n", dimStyle.Render("·"))
		fmt.Printf("  %s Install manually: %s\n", dimStyle.Render(" "), dimStyle.Render(openCodeInstallURL))
		return nil
	}

	// Get install options
	opts := getOpenCodeInstallOptions()
	if len(opts) == 0 {
		fmt.Printf("  %s No install method available for %s/%s\n",
			warnStyle.Render("⚠"), runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  %s Install manually: %s\n", dimStyle.Render(" "), dimStyle.Render(openCodeInstallURL))
		return nil
	}

	// If multiple options, let user choose
	selectedIdx := 0
	if len(opts) > 1 {
		options := make([]huh.Option[int], len(opts))
		for i, o := range opts {
			cmdPreview := o.Command
			if len(o.Args) > 0 {
				cmdPreview += " " + o.Args[len(o.Args)-1]
			}
			options[i] = huh.NewOption(o.Label, i)
			_ = cmdPreview
		}
		err := huh.NewSelect[int]().
			Title("Install method").
			Options(options...).
			Value(&selectedIdx).
			Run()
		if err != nil {
			return nil
		}
	}

	chosen := opts[selectedIdx]
	displayCmd := chosen.Command
	for _, a := range chosen.Args {
		displayCmd += " " + a
	}
	fmt.Printf("  %s Running: %s\n", StyleInfo.Render("→"), displayCmd)

	// Execute
	cmd := exec.Command(chosen.Command, chosen.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("  %s Install failed: %v\n", warnStyle.Render("⚠"), err)
		fmt.Printf("  %s Install manually: %s\n", dimStyle.Render(" "), dimStyle.Render(openCodeInstallURL))
		return nil // don't block init
	}

	// Verify
	verified := opencode.DetectOpenCode()
	if !verified.Installed {
		fmt.Printf("  %s OpenCode not found after install. You may need to restart your shell.\n",
			warnStyle.Render("⚠"))
		fmt.Printf("  %s Install manually: %s\n", dimStyle.Render(" "), dimStyle.Render(openCodeInstallURL))
		return nil
	}

	if !verified.Compatible {
		fmt.Printf("  %s OpenCode v%s installed but below minimum %s\n",
			warnStyle.Render("⚠"), verified.Version, verified.MinVersion)
		return nil
	}

	fmt.Printf("  %s OpenCode v%s installed successfully\n", RenderSuccess("✓"), verified.Version)
	return nil
}
