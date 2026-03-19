package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage project configuration",
}

// --- config get ---

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	store := getStore()

	val, err := store.Config.Get(key)
	if err != nil {
		// Try with "settings." prefix as shorthand
		if !strings.HasPrefix(key, "settings.") {
			val, err = store.Config.Get("settings." + key)
		}
		if err != nil {
			return fmt.Errorf("config get %q: %w", key, err)
		}
	}

	plain := isPlain(cmd)
	jsonOut := isJSON(cmd)

	if jsonOut {
		printJSON(val)
		return nil
	}

	if plain {
		fmt.Println(formatConfigValue(val))
	} else {
		fmt.Printf("%s %s %s\n", StyleBold.Render(key), StyleDim.Render("="), formatConfigValue(val))
	}
	return nil
}

// --- config set ---

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	rawVal := args[1]
	store := getStore()

	// Try to parse as number, bool, then fall back to string
	var val any
	if i, err := strconv.ParseInt(rawVal, 10, 64); err == nil {
		val = i
	} else if f, err := strconv.ParseFloat(rawVal, 64); err == nil {
		val = f
	} else if b, err := strconv.ParseBool(rawVal); err == nil {
		val = b
	} else {
		val = rawVal
	}

	// Auto-resolve shorthand: if key doesn't contain a dot and isn't a top-level key,
	// try with "settings." prefix
	actualKey := key
	if !strings.Contains(key, ".") && key != "name" && key != "id" {
		actualKey = "settings." + key
	}

	if err := store.Config.Set(actualKey, val); err != nil {
		return fmt.Errorf("config set %q: %w", key, err)
	}

	fmt.Println(RenderSuccess(fmt.Sprintf("Set %s = %v", key, rawVal)))
	return nil
}

// --- config list ---

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config settings",
	RunE:  runConfigList,
}

func runConfigList(cmd *cobra.Command, args []string) error {
	store := getStore()

	project, err := store.Config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	plain := isPlain(cmd)
	jsonOut := isJSON(cmd)

	if jsonOut {
		printJSON(project)
		return nil
	}

	if plain {
		fmt.Printf("PROJECT: %s\n", project.Name)
		fmt.Printf("ID: %s\n", project.ID)
		fmt.Printf("settings.defaultAssignee: %s\n", project.Settings.DefaultAssignee)
		fmt.Printf("settings.defaultPriority: %s\n", project.Settings.DefaultPriority)
		fmt.Printf("settings.statuses: %s\n", strings.Join(project.Settings.Statuses, ", "))
		if project.Settings.ServerPort != 0 {
			fmt.Printf("settings.serverPort: %d\n", project.Settings.ServerPort)
		}
		if project.Settings.GitTrackingMode != "" {
			fmt.Printf("settings.gitTrackingMode: %s\n", project.Settings.GitTrackingMode)
		}
		if project.Settings.OpenCodeServerConfig != nil {
			oc := project.Settings.OpenCodeServerConfig
			fmt.Printf("settings.opencodeServer.host: %s\n", oc.Host)
			if oc.Port != 0 {
				fmt.Printf("settings.opencodeServer.port: %d\n", oc.Port)
			}
			if oc.Password != "" {
				fmt.Printf("settings.opencodeServer.password: ****\n")
			}
		}
	} else {
		fmt.Printf("%s %s\n\n",
			StyleBold.Render(project.Name),
			StyleDim.Render(fmt.Sprintf("(ID: %s)", project.ID)))
		fmt.Println(RenderSectionHeader("Settings"))
		fmt.Printf("  %s %s\n", StyleDim.Render("defaultAssignee:"), project.Settings.DefaultAssignee)
		fmt.Printf("  %s %s\n", StyleDim.Render("defaultPriority:"), project.Settings.DefaultPriority)
		fmt.Printf("  %s %s\n", StyleDim.Render("statuses:       "), strings.Join(project.Settings.Statuses, ", "))
		if project.Settings.ServerPort != 0 {
			fmt.Printf("  %s %d\n", StyleDim.Render("serverPort:     "), project.Settings.ServerPort)
		}
		if project.Settings.GitTrackingMode != "" {
			fmt.Printf("  %s %s\n", StyleDim.Render("gitTrackingMode:"), project.Settings.GitTrackingMode)
		}
		if project.Settings.TimeFormat != "" {
			fmt.Printf("  %s %s\n", StyleDim.Render("timeFormat:     "), project.Settings.TimeFormat)
		}
		if project.Settings.OpenCodeServerConfig != nil {
			oc := project.Settings.OpenCodeServerConfig
			fmt.Println(RenderSectionHeader("OpenCode Server"))
			if oc.Host != "" {
				fmt.Printf("  %s %s\n", StyleDim.Render("host:         "), oc.Host)
			} else {
				fmt.Printf("  %s %s\n", StyleDim.Render("host:         "), "127.0.0.1 (default)")
			}
			if oc.Port != 0 {
				fmt.Printf("  %s %d\n", StyleDim.Render("port:         "), oc.Port)
			} else {
				fmt.Printf("  %s %d\n", StyleDim.Render("port:         "), 4096)
			}
			if oc.Password != "" {
				fmt.Printf("  %s ****\n", StyleDim.Render("password:     "))
			}
		}
	}
	return nil
}

// formatConfigValue converts a config value to its string representation.
func formatConfigValue(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int, int64:
		return fmt.Sprintf("%v", x)
	case []any:
		strs := make([]string, 0, len(x))
		for _, item := range x {
			strs = append(strs, formatConfigValue(item))
		}
		return strings.Join(strs, ", ")
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// --- config reset ---

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset config to defaults",
	RunE:  runConfigReset,
}

func runConfigReset(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	store := getStore()

	if !yes {
		answer := ""
		fmt.Print("Reset config to defaults? This cannot be undone. (y/n): ")
		fmt.Scanln(&answer)
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Load current config to preserve the project name
	project, err := store.Config.Load()
	name := "knowns"
	if err == nil && project.Name != "" {
		name = project.Name
	}

	// Re-initialize default config
	if err := store.Config.Set("settings", nil); err != nil {
		// Fall back: reinit the whole config
		_ = err
	}

	// Write a fresh default config preserving the project name
	defaultSettings := models.DefaultProjectSettings()
	project = &models.Project{
		Name:     name,
		ID:       project.ID,
		Settings: defaultSettings,
	}
	if err := store.Config.Save(project); err != nil {
		return fmt.Errorf("reset config: %w", err)
	}

	fmt.Println(RenderSuccess("Config reset to defaults."))
	return nil
}

func init() {
	configResetCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configResetCmd)

	rootCmd.AddCommand(configCmd)
}
