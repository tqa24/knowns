package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/howznguyen/knowns/internal/runtimequeue"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/howznguyen/knowns/internal/storage"
	"github.com/spf13/cobra"
)

var runtimeCmd = &cobra.Command{
	Use:    "__runtime",
	Short:  "Internal shared runtime",
	Hidden: true,
}

var runtimeRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the internal shared runtime",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runtimequeue.RunDaemon(cmd.Context(), search.ExecuteRuntimeJob, startRuntimeWatcher)
	},
}

var runtimeStatusCmd = &cobra.Command{
	Use:    "status",
	Short:  "Show shared runtime status",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := runtimequeue.LoadStatus()
		if err != nil {
			return err
		}
		if isJSON(cmd) || isPlain(cmd) {
			printJSON(status)
			return nil
		}
		fmt.Printf("Runtime running: %v\n", status.Running)
		fmt.Printf("PID: %d\n", status.PID)
		fmt.Printf("Clients: %d\n", len(status.Clients))
		fmt.Printf("Projects: %d\n", len(status.Project))
		return nil
	},
}

func startRuntimeWatcher(ctx context.Context, storeRoot string) error {
	store := storage.NewStore(storeRoot)
	return StartCodeWatcher(ctx, store, filepath.Dir(storeRoot), watchDebounceMs)
}

func init() {
	runtimeCmd.AddCommand(runtimeRunCmd)
	runtimeCmd.AddCommand(runtimeStatusCmd)
	rootCmd.AddCommand(runtimeCmd)
}
