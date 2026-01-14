package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/krish/grappler/internal/config"
	"github.com/krish/grappler/internal/ports"
	"github.com/krish/grappler/internal/process"
	"github.com/spf13/cobra"
)

// StartCmd returns the start command
func StartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <group>",
		Short: "Start a worktree group",
		Long:  `Starts the backend and frontend services for a worktree group with allocated ports.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runStart,
	}
}

func runStart(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	// Load config
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config (run 'grappler init' first): %w", err)
	}

	// Find group
	group, exists := cfg.Groups[groupName]
	if !exists {
		return fmt.Errorf("group %q not found in config", groupName)
	}

	// Load state
	state, err := config.LoadState(config.GetStatePath())
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Check if group is already running
	groupState := state.GetGroup(groupName)
	if groupState != nil && groupState.Running {
		return fmt.Errorf("group %q is already running", groupName)
	}

	fmt.Printf("Starting group %q...\n", groupName)

	// Allocate ports
	allocator := ports.NewAllocator(state)

	var backendPort, frontendPort int
	if group.Backend != nil {
		backendPort, err = allocator.AllocateBackendPort()
		if err != nil {
			return fmt.Errorf("failed to allocate backend port: %w", err)
		}
		fmt.Printf("  Backend port:  %d\n", backendPort)
	}

	if group.Frontend != nil {
		frontendPort, err = allocator.AllocateFrontendPort()
		if err != nil {
			return fmt.Errorf("failed to allocate frontend port: %w", err)
		}
		fmt.Printf("  Frontend port: %d\n", frontendPort)
	}

	// Start processes
	procMgr := process.NewManager(config.GetLogsDir())

	newState := &config.GroupState{
		BackendPort:  backendPort,
		FrontendPort: frontendPort,
		Running:      true,
	}

	// Start backend
	if group.Backend != nil {
		fmt.Println("\nStarting backend...")
		envVars := map[string]string{
			"SERVER_PORT": strconv.Itoa(backendPort),
		}

		pid, err := procMgr.StartService(group.Backend, "backend", groupName, envVars)
		if err != nil {
			return fmt.Errorf("failed to start backend: %w", err)
		}

		newState.BackendPID = pid
		fmt.Printf("✓ Backend started (PID: %d)\n", pid)
	}

	// Start frontend
	if group.Frontend != nil {
		fmt.Println("\nStarting frontend...")
		envVars := map[string]string{
			"CONDUCTOR_PORT": strconv.Itoa(frontendPort),
		}

		pid, err := procMgr.StartService(group.Frontend, "frontend", groupName, envVars)
		if err != nil {
			// If frontend fails, stop backend
			if newState.BackendPID > 0 {
				procMgr.StopProcess(newState.BackendPID)
			}
			return fmt.Errorf("failed to start frontend: %w", err)
		}

		newState.FrontendPID = pid
		fmt.Printf("✓ Frontend started (PID: %d)\n", pid)
	}

	// Save state
	state.SetGroup(groupName, newState)
	if err := state.Save(config.GetStatePath()); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Wait for services to be healthy
	fmt.Println("\nWaiting for services to be healthy...")
	healthChecker := process.NewHealthChecker()

	if newState.BackendPort > 0 {
		if err := healthChecker.WaitForHealth(backendPort, 30*time.Second); err != nil {
			fmt.Printf("⚠ Backend health check failed: %v\n", err)
			fmt.Printf("  Check logs: ~/.grappler/logs/%s-backend.log\n", groupName)
		} else {
			fmt.Printf("✓ Backend healthy (http://localhost:%d)\n", backendPort)
		}
	}

	if newState.FrontendPort > 0 {
		if err := healthChecker.WaitForHealth(frontendPort, 30*time.Second); err != nil {
			fmt.Printf("⚠ Frontend health check failed: %v\n", err)
			fmt.Printf("  Check logs: ~/.grappler/logs/%s-frontend.log\n", groupName)
		} else {
			fmt.Printf("✓ Frontend healthy (http://localhost:%d)\n", frontendPort)
		}
	}

	// Print access info
	fmt.Println("\n" + repeatString("=", 50))
	fmt.Printf("Group %q is running\n", groupName)

	if newState.FrontendPort > 0 {
		fmt.Printf("\nAccess frontend via conductor proxy:\n")
		fmt.Printf("  http://%d.port.localhost:3000\n", frontendPort)
		fmt.Printf("\nDirect access:\n")
	}

	if newState.BackendPort > 0 {
		fmt.Printf("  Backend:  http://localhost:%d\n", backendPort)
	}

	if newState.FrontendPort > 0 {
		fmt.Printf("  Frontend: http://localhost:%d\n", frontendPort)
	}

	fmt.Println("\n" + repeatString("=", 50))

	return nil
}

func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
