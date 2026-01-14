package cli

import (
	"fmt"

	"github.com/krish/grappler/internal/config"
	"github.com/krish/grappler/internal/process"
	"github.com/spf13/cobra"
)

// StatusCmd returns the status command
func StatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of all groups",
		Long:  `Displays the status of all configured groups including their ports and running state.`,
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config (run 'grappler init' first): %w", err)
	}

	// Load state
	state, err := config.LoadState(config.GetStatePath())
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	procMgr := process.NewManager(config.GetLogsDir())

	fmt.Println("Grappler Status")
	fmt.Println(repeatString("=", 80))

	if len(cfg.Groups) == 0 {
		fmt.Println("No groups configured")
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-15s %-15s %-10s %s\n", "GROUP", "BACKEND PORT", "FRONTEND PORT", "STATUS", "ACCESS")
	fmt.Println(repeatString("-", 80))

	for name, group := range cfg.Groups {
		groupState := state.GetGroup(name)

		backendPort := "-"
		frontendPort := "-"
		status := "stopped"
		access := "-"

		if groupState != nil && groupState.Running {
			// Verify processes are actually running
			backendRunning := procMgr.IsProcessRunning(groupState.BackendPID)
			frontendRunning := procMgr.IsProcessRunning(groupState.FrontendPID)

			if !backendRunning && !frontendRunning {
				// Both stopped - clean up state
				status = "stopped"
				state.DeleteGroup(name)
			} else {
				status = "running"

				if groupState.BackendPort > 0 {
					backendPort = fmt.Sprintf("%d", groupState.BackendPort)
				}

				if groupState.FrontendPort > 0 {
					frontendPort = fmt.Sprintf("%d", groupState.FrontendPort)
					access = fmt.Sprintf("http://%d.port.localhost:3000", groupState.FrontendPort)
				} else if groupState.BackendPort > 0 {
					access = fmt.Sprintf("http://localhost:%d", groupState.BackendPort)
				}
			}
		}

		// Print group info
		fmt.Printf("%-20s %-15s %-15s %-10s %s\n", name, backendPort, frontendPort, status, access)

		// Show branch info
		if group.Backend != nil {
			fmt.Printf("  Backend:  %s\n", group.Backend.Branch)
		}
		if group.Frontend != nil {
			fmt.Printf("  Frontend: %s\n", group.Frontend.Branch)
		}
		fmt.Println()
	}

	// Save state if we cleaned up any stopped groups
	if err := state.Save(config.GetStatePath()); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}
