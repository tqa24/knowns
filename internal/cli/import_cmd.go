package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Manage imported Knowns packages",
}

// --- import add ---

var importAddCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Add an import source",
	Long: `Add a Knowns package import. The source can be:
  - A local path: ./path/to/package
  - An npm package: @scope/package or package-name`,
	Args: cobra.ExactArgs(1),
	RunE: runImportAdd,
}

func runImportAdd(cmd *cobra.Command, args []string) error {
	source := args[0]

	store, err := getStoreErr()
	if err != nil {
		return err
	}

	// Determine import name from source
	name := importNameFromSource(source)
	importsDir := filepath.Join(store.Root, "imports", name)

	if _, statErr := os.Stat(importsDir); statErr == nil {
		return fmt.Errorf("import %q already exists", name)
	}

	if err := os.MkdirAll(importsDir, 0755); err != nil {
		return fmt.Errorf("create import directory: %w", err)
	}

	// Write a simple manifest file
	manifestPath := filepath.Join(importsDir, "_import.json")
	manifest := fmt.Sprintf(`{"source": %q, "name": %q}`, source, name)
	if err := os.WriteFile(manifestPath, []byte(manifest+"\n"), 0644); err != nil {
		return fmt.Errorf("write import manifest: %w", err)
	}

	fmt.Println(RenderSuccess(fmt.Sprintf("Added import: %s (from %s)", name, source)))
	fmt.Println("Run 'knowns import sync' to fetch the package contents.")
	return nil
}

// --- import remove ---

var importRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an imported package",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportRemove,
}

func runImportRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := getStoreErr()
	if err != nil {
		return err
	}

	importsDir := filepath.Join(store.Root, "imports", name)
	if _, statErr := os.Stat(importsDir); os.IsNotExist(statErr) {
		return fmt.Errorf("import %q not found", name)
	}

	if err := os.RemoveAll(importsDir); err != nil {
		return fmt.Errorf("remove import: %w", err)
	}

	fmt.Println(RenderSuccess(fmt.Sprintf("Removed import: %s", name)))
	return nil
}

// --- import list ---

var importListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all imported packages",
	RunE:  runImportList,
}

func runImportList(cmd *cobra.Command, args []string) error {
	store, err := getStoreErr()
	if err != nil {
		return err
	}

	importsDir := filepath.Join(store.Root, "imports")
	entries, err := os.ReadDir(importsDir)
	if os.IsNotExist(err) {
		fmt.Println("No imports found.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read imports directory: %w", err)
	}

	plain := isPlain(cmd)

	var imports []string
	for _, e := range entries {
		if e.IsDir() {
			imports = append(imports, e.Name())
		}
	}

	if len(imports) == 0 {
		fmt.Println("No imports found.")
		return nil
	}

	if plain {
		for _, name := range imports {
			fmt.Printf("IMPORT: %s\n", name)
		}
	} else {
		fmt.Printf("%s\n", RenderSectionHeader(fmt.Sprintf("Imported packages (%d)", len(imports))))
		for _, name := range imports {
			fmt.Printf("  %s %s\n", StyleDim.Render("•"), name)
		}
	}
	return nil
}

// --- import sync ---

var importSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all imported packages",
	RunE:  runImportSync,
}

func runImportSync(cmd *cobra.Command, args []string) error {
	store, err := getStoreErr()
	if err != nil {
		return err
	}

	importsDir := filepath.Join(store.Root, "imports")
	entries, err := os.ReadDir(importsDir)
	if os.IsNotExist(err) {
		fmt.Println("No imports to sync.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read imports: %w", err)
	}

	synced := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fmt.Printf("Syncing %s...\n", e.Name())
		// Sync implementation would go here (npm install, git clone, etc.)
		// For now we just acknowledge
		synced++
	}

	if synced == 0 {
		fmt.Println("No imports to sync.")
	} else {
		fmt.Printf("Synced %d import(s).\n", synced)
		fmt.Println("(Note: actual network sync not yet implemented in this build.)")
	}
	return nil
}

// importNameFromSource derives a safe directory name from an import source.
func importNameFromSource(source string) string {
	// Strip leading ./ or ../
	name := strings.TrimPrefix(source, "./")
	name = strings.TrimPrefix(name, "../")

	// For npm scoped packages like @scope/package → scope__package
	name = strings.TrimPrefix(name, "@")
	name = strings.ReplaceAll(name, "/", "__")
	name = strings.ReplaceAll(name, "\\", "__")

	// Remove unsafe characters
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	result := b.String()
	if result == "" {
		result = "imported"
	}
	return result
}

// --- import status ---

var importStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show import status",
	RunE:  runImportStatus,
}

func runImportStatus(cmd *cobra.Command, args []string) error {
	store, err := getStoreErr()
	if err != nil {
		return err
	}

	importsDir := filepath.Join(store.Root, "imports")
	entries, err := os.ReadDir(importsDir)
	if os.IsNotExist(err) {
		fmt.Println("No imports configured.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read imports directory: %w", err)
	}

	plain := isPlain(cmd)

	var imports []string
	for _, e := range entries {
		if e.IsDir() {
			imports = append(imports, e.Name())
		}
	}

	if len(imports) == 0 {
		fmt.Println("No imports configured.")
		return nil
	}

	if plain {
		fmt.Printf("IMPORTS: %d\n\n", len(imports))
		for _, name := range imports {
			manifestPath := filepath.Join(importsDir, name, "_import.json")
			status := "unknown"
			source := ""
			if data, readErr := os.ReadFile(manifestPath); readErr == nil {
				status = "installed"
				// Try to extract source
				var manifest map[string]string
				if jsonErr := json.Unmarshal(data, &manifest); jsonErr == nil {
					source = manifest["source"]
				}
			} else {
				status = "missing-manifest"
			}
			fmt.Printf("IMPORT: %s\n", name)
			fmt.Printf("  STATUS: %s\n", status)
			if source != "" {
				fmt.Printf("  SOURCE: %s\n", source)
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("%s\n\n", RenderSectionHeader(fmt.Sprintf("Import status (%d packages)", len(imports))))
		for _, name := range imports {
			manifestPath := filepath.Join(importsDir, name, "_import.json")
			status := "unknown"
			source := ""
			if data, readErr := os.ReadFile(manifestPath); readErr == nil {
				status = "installed"
				var manifest map[string]string
				if jsonErr := json.Unmarshal(data, &manifest); jsonErr == nil {
					source = manifest["source"]
				}
			} else {
				status = "missing manifest"
			}
			statusStyled := StyleDim.Render("[" + status + "]")
			if status == "installed" {
				statusStyled = StyleSuccess.Render("[" + status + "]")
			}
			if source != "" {
				fmt.Printf("  %-20s %s %s\n", name, statusStyled, StyleDim.Render("(from "+source+")"))
			} else {
				fmt.Printf("  %-20s %s\n", name, statusStyled)
			}
		}
	}
	return nil
}

func init() {
	importCmd.AddCommand(importAddCmd)
	importCmd.AddCommand(importRemoveCmd)
	importCmd.AddCommand(importListCmd)
	importCmd.AddCommand(importSyncCmd)
	importCmd.AddCommand(importStatusCmd)

	rootCmd.AddCommand(importCmd)
}
