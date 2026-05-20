package adapters

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/howznguyen/knowns/internal/lsp"
)

type RoslynAdapter struct{ lsp.BaseAdapter }

func NewRoslynAdapter() *RoslynAdapter { return &RoslynAdapter{} }

func (a *RoslynAdapter) ID() string           { return "csharp" }
func (a *RoslynAdapter) Name() string         { return "C#" }
func (a *RoslynAdapter) Extensions() []string { return []string{".cs"} }
func (a *RoslynAdapter) Binaries() []lsp.BinaryCandidate {
	return []lsp.BinaryCandidate{{Name: "csharp-ls", Args: []string{"--stdio"}, CheckArgs: []string{"--version"}}}
}
func (a *RoslynAdapter) Prerequisites() []lsp.Prerequisite {
	return []lsp.Prerequisite{{Name: ".NET SDK 8+", CheckCmd: "dotnet --version", InstallHint: "Install .NET SDK 8+ from https://dotnet.microsoft.com/download"}}
}
func (a *RoslynAdapter) CheckPrerequisites(ctx context.Context) error {
	output, err := commandOutput(ctx, "dotnet", "--version")
	if err != nil {
		return err
	}
	return requireMinVersion(output, ".NET SDK", 8, 0)
}
func (a *RoslynAdapter) InstallGuide() lsp.InstallGuide {
	return lsp.InstallGuide{Command: "dotnet tool install --global csharp-ls", KnownsCmd: "knowns lsp install csharp", URL: "https://github.com/razzmatazz/csharp-language-server", Notes: "Requires .NET SDK 8+"}
}
func (a *RoslynAdapter) CanInstall() bool                     { return true }
func (a *RoslynAdapter) RuntimeDeps() []lsp.RuntimeDependency { return nil }
func (a *RoslynAdapter) Install(ctx context.Context, targetDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "dotnet", "tool", "install", "--global", "csharp-ls")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("dotnet tool install failed: %w: %s", err, output)
	}
	path, err := exec.LookPath("csharp-ls")
	if err != nil {
		return "", fmt.Errorf("csharp-ls installed but not found in PATH: %w", err)
	}
	return path, nil
}
func (a *RoslynAdapter) InstalledPath() (string, bool) { return installedPath(a.ID(), a.RuntimeDeps()) }
func (a *RoslynAdapter) DefaultArgs() []string         { return []string{"--stdio"} }
func (a *RoslynAdapter) InitializeParams(root string, settings map[string]any) map[string]any {
	return initializeParams(root, settings)
}
func (a *RoslynAdapter) InitializationOptions(settings map[string]any) map[string]any {
	return initializationOptions(settings)
}
func (a *RoslynAdapter) IsIgnoredDir(name string) bool {
	return isIgnoredDir(name, map[string]struct{}{"bin": {}, "obj": {}, ".vs": {}, "packages": {}})
}
