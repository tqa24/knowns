package cli

import (
	"fmt"

	instructionguidelines "github.com/howznguyen/knowns/internal/instructions/guidelines"
	"github.com/spf13/cobra"
)

var guidelinesCmd = &cobra.Command{
	Use:   "guidelines",
	Short: "Display Knowns usage guidelines",
	RunE:  runGuidelines,
}

func runGuidelines(cmd *cobra.Command, args []string) error {
	text, err := instructionguidelines.RenderFull(instructionguidelines.RenderOptions{
		CLI: true,
		MCP: true,
	})
	if err != nil {
		return fmt.Errorf("render guidelines: %w", err)
	}

	if isPlain(cmd) {
		fmt.Print(text)
		return nil
	}

	renderOrPage(cmd, "Knowns Guidelines", text)
	return nil
}

func init() {
	rootCmd.AddCommand(guidelinesCmd)
}
