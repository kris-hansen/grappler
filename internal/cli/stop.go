package cli

import (
	"fmt"

	"github.com/kris-hansen/grappler/internal/config"
	"github.com/kris-hansen/grappler/internal/process"
	"github.com/spf13/cobra"
)

// StopCmd returns the stop command
func StopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <group>",
		Short: "Stop a running worktree group",
		Long:  `Stops the backend and frontend services for a running worktree group and releases ports.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runStop,
	}
}

func runStop(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	// Load state
	state, err := config.LoadState(config.GetStatePath())
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find group state
	groupState := state.GetGroup(groupName)
	if groupState == nil || !groupState.Running {
		return fmt.Errorf("group %q is not running", groupName)
	}

	fmt.Printf("Stopping group %q...\n", groupName)

	procMgr := process.NewManager(config.GetLogsDir())

	// Stop backend
	if groupState.BackendPID > 0 {
		fmt.Printf("Stopping backend (PID: %d)...\n", groupState.BackendPID)
		if err := procMgr.StopProcess(groupState.BackendPID); err != nil {
			fmt.Printf("⚠ Failed to stop backend: %v\n", err)
		} else {
			fmt.Println("✓ Backend stopped")
		}
	}

	// Stop frontend
	if groupState.FrontendPID > 0 {
		fmt.Printf("Stopping frontend (PID: %d)...\n", groupState.FrontendPID)
		if err := procMgr.StopProcess(groupState.FrontendPID); err != nil {
			fmt.Printf("⚠ Failed to stop frontend: %v\n", err)
		} else {
			fmt.Println("✓ Frontend stopped")
		}
	}

	// Remove from state
	state.DeleteGroup(groupName)
	if err := state.Save(config.GetStatePath()); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("\n✓ Group %q stopped\n", groupName)

	return nil
}
