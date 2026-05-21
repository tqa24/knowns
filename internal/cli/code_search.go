package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var codeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search indexed code with optional neighbors",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCodeSearch,
}

func runCodeSearch(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("action removed")
}
