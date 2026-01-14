package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kris-hansen/grappler/internal/config"
	"github.com/kris-hansen/grappler/internal/process"
	"github.com/kris-hansen/grappler/internal/worktree"
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
	runningPorts := make(map[string][]servicePort)

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
					if group.Backend != nil {
						runningPorts[group.Backend.Directory] = append(runningPorts[group.Backend.Directory], servicePort{
							Group: name,
							Role:  "backend",
							Port:  groupState.BackendPort,
						})
					}
				}

				if groupState.FrontendPort > 0 {
					frontendPort = fmt.Sprintf("%d", groupState.FrontendPort)
					access = fmt.Sprintf("http://%d.port.localhost:3000", groupState.FrontendPort)
					if group.Frontend != nil {
						runningPorts[group.Frontend.Directory] = append(runningPorts[group.Frontend.Directory], servicePort{
							Group: name,
							Role:  "frontend",
							Port:  groupState.FrontendPort,
						})
					}
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

	fmt.Println(repeatString("=", 80))
	fmt.Println("Worktree Port Map")
	fmt.Println(repeatString("-", 80))

	repoWorktrees, err := scanRepoWorktrees(cfg)
	if err != nil {
		return err
	}

	if len(repoWorktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	printWorktreePortMap(repoWorktrees, runningPorts)

	return nil
}

type servicePort struct {
	Group string
	Role  string
	Port  int
}

func scanRepoWorktrees(cfg *config.Config) (map[string][]worktree.Worktree, error) {
	repoDirs := make(map[string]string)

	for _, group := range cfg.Groups {
		if group.Backend != nil {
			commonDir, err := worktree.GetCommonDir(group.Backend.Directory)
			if err != nil {
				return nil, fmt.Errorf("failed to get backend repo info: %w", err)
			}
			if _, ok := repoDirs[commonDir]; !ok {
				repoDirs[commonDir] = group.Backend.Directory
			}
		}

		if group.Frontend != nil {
			commonDir, err := worktree.GetCommonDir(group.Frontend.Directory)
			if err != nil {
				return nil, fmt.Errorf("failed to get frontend repo info: %w", err)
			}
			if _, ok := repoDirs[commonDir]; !ok {
				repoDirs[commonDir] = group.Frontend.Directory
			}
		}
	}

	repoWorktrees := make(map[string][]worktree.Worktree)
	for commonDir, repoDir := range repoDirs {
		worktrees, err := worktree.ScanWorktrees(repoDir)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worktrees for %s: %w", repoDir, err)
		}
		repoWorktrees[commonDir] = worktrees
	}

	return repoWorktrees, nil
}

func printWorktreePortMap(repoWorktrees map[string][]worktree.Worktree, runningPorts map[string][]servicePort) {
	repos := make([]string, 0, len(repoWorktrees))
	for repo := range repoWorktrees {
		repos = append(repos, repo)
	}
	sort.Strings(repos)

	for _, repo := range repos {
		worktrees := repoWorktrees[repo]
		repoRoot := repo
		if filepath.Base(repo) == ".git" {
			repoRoot = filepath.Dir(repo)
		}

		fmt.Printf("Repository: %s\n", repoRoot)
		fmt.Printf("%-50s %-20s %s\n", "WORKTREE", "BRANCH", "PORTS IN USE")
		fmt.Println(repeatString("-", 80))

		sort.Slice(worktrees, func(i, j int) bool {
			return worktrees[i].Path < worktrees[j].Path
		})

		for _, wt := range worktrees {
			ports := runningPorts[wt.Path]
			portInfo := "-"
			if len(ports) > 0 {
				var parts []string
				for _, port := range ports {
					parts = append(parts, fmt.Sprintf("%s:%d (%s)", port.Role, port.Port, port.Group))
				}
				portInfo = strings.Join(parts, ", ")
			}

			fmt.Printf("%-50s %-20s %s\n", wt.Path, wt.Branch, portInfo)
		}

		fmt.Println()
	}
}
