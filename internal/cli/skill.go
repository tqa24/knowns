package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Knowns AI skills",
}

// --- skill list ---

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills",
	RunE:  runSkillList,
}

func runSkillList(cmd *cobra.Command, args []string) error {
	store, err := getStoreErr()
	if err != nil {
		return err
	}

	// Skills live in .knowns/imports/*/skills/ or .knowns/skills/
	// For now, list imported packages that may contain skills
	templates, err := store.Templates.List()
	if err != nil {
		return fmt.Errorf("list templates: %w", err)
	}

	plain := isPlain(cmd)

	if len(templates) == 0 {
		fmt.Println(StyleDim.Render("No skills found."))
		return nil
	}

	if plain {
		fmt.Printf("SKILLS: %d\n\n", len(templates))
		for _, t := range templates {
			fmt.Printf("SKILL: %s\n", t.Name)
			if t.Description != "" {
				fmt.Printf("  DESCRIPTION: %s\n", t.Description)
			}
			if t.IsImported {
				fmt.Printf("  SOURCE: %s\n", t.ImportName)
			}
		}
	} else {
		fmt.Printf("%s\n\n", RenderSectionHeader(fmt.Sprintf("Available skills (%d)", len(templates))))
		for _, t := range templates {
			if t.IsImported {
				fmt.Printf("  %s %s\n",
					StyleID.Render(fmt.Sprintf("[%s/%s]", t.ImportName, t.Name)),
					t.Description)
			} else {
				fmt.Printf("  %s %s\n",
					StyleID.Render(fmt.Sprintf("[local/%s]", t.Name)),
					t.Description)
			}
		}
	}
	return nil
}

// --- skill sync ---

var skillSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync skills from imported packages",
	RunE:  runSkillSync,
}

func runSkillSync(cmd *cobra.Command, args []string) error {
	fmt.Println("Syncing skills...")
	fmt.Println("(Skills are synced via 'knowns import sync'. Skill sync not yet implemented separately.)")
	return nil
}

// --- skill view ---

var skillViewCmd = &cobra.Command{
	Use:   "view <name>",
	Short: "View skill details",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillView,
}

func runSkillView(cmd *cobra.Command, args []string) error {
	name := args[0]
	store, err := getStoreErr()
	if err != nil {
		return err
	}

	plain := isPlain(cmd)
	jsonOut := isJSON(cmd)

	// Skills are backed by templates
	templates, err := store.Templates.List()
	if err != nil {
		return fmt.Errorf("list templates: %w", err)
	}

	// Find matching template/skill
	for _, t := range templates {
		fullName := t.Name
		if t.IsImported {
			fullName = t.ImportName + "/" + t.Name
		}
		if t.Name == name || fullName == name {
			if jsonOut {
				printJSON(t)
				return nil
			}
			if plain {
				fmt.Printf("SKILL: %s\n", fullName)
				if t.Description != "" {
					fmt.Printf("DESCRIPTION: %s\n", t.Description)
				}
				if t.IsImported {
					fmt.Printf("SOURCE: %s\n", t.ImportName)
				} else {
					fmt.Printf("SOURCE: local\n")
				}
				if t.Doc != "" {
					fmt.Printf("DOC: %s\n", t.Doc)
				}
				if len(t.Actions) > 0 {
					fmt.Printf("ACTIONS: %d\n", len(t.Actions))
					for _, a := range t.Actions {
						fmt.Printf("  ACTION: %s %s\n", a.Type, a.Path)
					}
				}
				if len(t.Prompts) > 0 {
					fmt.Printf("PROMPTS:\n")
					for _, p := range t.Prompts {
						fmt.Printf("  PROMPT: %s (%s)\n", p.Name, p.Type)
					}
				}
			} else {
				fmt.Println(StyleTitle.Render(fullName))
				if t.Description != "" {
					fmt.Println(StyleDim.Render(t.Description))
				}
				if t.IsImported {
					fmt.Println(RenderKeyValue("Source", t.ImportName))
				} else {
					fmt.Println(RenderKeyValue("Source", "local"))
				}
				if t.Doc != "" {
					fmt.Println(RenderKeyValue("Doc", t.Doc))
				}
				if len(t.Actions) > 0 {
					fmt.Printf("\n%s\n", RenderSectionHeader(fmt.Sprintf("Actions (%d)", len(t.Actions))))
					for _, a := range t.Actions {
						fmt.Printf("  %s %s %s\n", StyleDim.Render("-"), StyleInfo.Render("["+a.Type+"]"), a.Path)
					}
				}
				if len(t.Prompts) > 0 {
					fmt.Printf("\n%s\n", RenderSectionHeader("Prompts"))
					for _, p := range t.Prompts {
						fmt.Printf("  %s %s %s: %s\n", StyleDim.Render("-"), StyleBold.Render(p.Name), StyleDim.Render("("+p.Type+")"), p.Message)
					}
				}
			}
			return nil
		}
	}

	return fmt.Errorf("skill %q not found", name)
}

func init() {
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillSyncCmd)
	skillCmd.AddCommand(skillViewCmd)

	rootCmd.AddCommand(skillCmd)
}
