package cli

import (
	"fmt"

	"github.com/kris-hansen/grappler/internal/config"
	"github.com/kris-hansen/grappler/internal/worktree"
	"github.com/spf13/cobra"
)

// InitCmd returns the init command
func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <backend-repo> <frontend-repo>",
		Short: "Initialize grappler configuration by scanning worktrees",
		Long:  `Scans the specified backend and frontend repositories for git worktrees and generates a configuration file.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	backendRepo := args[0]
	frontendRepo := args[1]

	fmt.Println("Scanning worktrees...")
	fmt.Printf("  Backend:  %s\n", backendRepo)
	fmt.Printf("  Frontend: %s\n", frontendRepo)

	// Scan backend worktrees
	backendWorktrees, err := worktree.ScanWorktrees(backendRepo)
	if err != nil {
		return fmt.Errorf("failed to scan backend worktrees: %w", err)
	}

	// Scan frontend worktrees
	frontendWorktrees, err := worktree.ScanWorktrees(frontendRepo)
	if err != nil {
		return fmt.Errorf("failed to scan frontend worktrees: %w", err)
	}

	fmt.Printf("\nFound %d backend worktrees and %d frontend worktrees\n", len(backendWorktrees), len(frontendWorktrees))

	// Pair worktrees into groups
	groups := worktree.PairWorktrees(backendWorktrees, frontendWorktrees)

	// Create config
	cfg := &config.Config{
		Version: "1",
		Groups:  groups,
		Proxy: &config.ProxyConfig{
			Enabled:              true,
			UseExistingConductor: true,
		},
	}

	// Save config
	configPath := config.GetConfigPath()
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Initialize empty state
	state := config.NewState()
	statePath := config.GetStatePath()
	if err := state.Save(statePath); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	fmt.Printf("✓ State file created at %s\n", statePath)
	fmt.Printf("\nDiscovered groups:\n")

	for name, group := range groups {
		fmt.Printf("  %s:\n", name)
		if group.Backend != nil {
			fmt.Printf("    Backend:  %s (%s)\n", group.Backend.Directory, group.Backend.Branch)
		}
		if group.Frontend != nil {
			fmt.Printf("    Frontend: %s (%s)\n", group.Frontend.Directory, group.Frontend.Branch)
		}
	}

	fmt.Printf("\nRun 'grappler start <group>' to start a group\n")

	return nil
}
