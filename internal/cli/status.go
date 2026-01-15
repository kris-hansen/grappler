package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	if pruneMissingDirectories(cfg) {
		if err := cfg.Save(config.GetConfigPath()); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
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

	externalPorts, err := scanListeningPorts(repoWorktrees)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to scan listening ports: %v\n", err)
	} else {
		mergePorts(runningPorts, externalPorts)
	}

	printWorktreePortMap(repoWorktrees, runningPorts)

	return nil
}

type servicePort struct {
	Group   string
	Role    string
	Port    int
	Process string
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

func pruneMissingDirectories(cfg *config.Config) bool {
	updated := false

	for name, group := range cfg.Groups {
		if group.Backend != nil {
			if _, err := os.Stat(group.Backend.Directory); err != nil {
				if os.IsNotExist(err) {
					group.Backend = nil
					updated = true
				} else {
					fmt.Printf("Warning: failed to stat backend directory for %s: %v\n", name, err)
				}
			}
		}

		if group.Frontend != nil {
			if _, err := os.Stat(group.Frontend.Directory); err != nil {
				if os.IsNotExist(err) {
					group.Frontend = nil
					updated = true
				} else {
					fmt.Printf("Warning: failed to stat frontend directory for %s: %v\n", name, err)
				}
			}
		}

		if group.Backend == nil && group.Frontend == nil {
			delete(cfg.Groups, name)
			updated = true
		}
	}

	return updated
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
					label := port.Group
					if label == "" {
						label = port.Process
					}
					if label == "" {
						parts = append(parts, fmt.Sprintf("%s:%d", port.Role, port.Port))
					} else {
						parts = append(parts, fmt.Sprintf("%s:%d (%s)", port.Role, port.Port, label))
					}
				}
				portInfo = strings.Join(parts, ", ")
			}

			fmt.Printf("%-50s %-20s %s\n", wt.Path, wt.Branch, portInfo)
		}

		fmt.Println()
	}
}

type listeningPort struct {
	PID     int
	Port    int
	Command string
	Cwd     string
}

func scanListeningPorts(repoWorktrees map[string][]worktree.Worktree) (map[string][]servicePort, error) {
	worktreePaths := collectWorktreePaths(repoWorktrees)
	if len(worktreePaths) == 0 {
		return map[string][]servicePort{}, nil
	}

	ports, err := collectListeningPorts()
	if err != nil {
		return nil, err
	}

	portsByWorktree := make(map[string][]servicePort)
	for _, port := range ports {
		if port.Cwd == "" {
			continue
		}
		worktreePath := matchWorktree(worktreePaths, port.Cwd)
		if worktreePath == "" {
			continue
		}

		portsByWorktree[worktreePath] = appendUniquePort(portsByWorktree[worktreePath], servicePort{
			Role:    "listen",
			Port:    port.Port,
			Process: port.Command,
		})
	}

	return portsByWorktree, nil
}

func collectWorktreePaths(repoWorktrees map[string][]worktree.Worktree) []string {
	paths := []string{}
	for _, worktrees := range repoWorktrees {
		for _, wt := range worktrees {
			paths = append(paths, wt.Path)
		}
	}
	return paths
}

func collectListeningPorts() ([]listeningPort, error) {
	cmd := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN", "-F", "pcn")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var currentPID int
	var currentCommand string
	seen := make(map[string]bool)
	ports := []listeningPort{}

	for _, line := range lines {
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, err := strconv.Atoi(strings.TrimPrefix(line, "p"))
			if err != nil {
				currentPID = 0
				currentCommand = ""
				continue
			}
			currentPID = pid
		case 'c':
			currentCommand = strings.TrimPrefix(line, "c")
		case 'n':
			if currentPID == 0 {
				continue
			}
			port := parsePort(strings.TrimPrefix(line, "n"))
			if port == 0 {
				continue
			}
			key := fmt.Sprintf("%d:%d", currentPID, port)
			if seen[key] {
				continue
			}
			seen[key] = true
			ports = append(ports, listeningPort{
				PID:     currentPID,
				Port:    port,
				Command: currentCommand,
			})
		}
	}

	if len(ports) == 0 {
		return ports, nil
	}

	cwdByPID := make(map[int]string)
	for _, port := range ports {
		if _, ok := cwdByPID[port.PID]; ok {
			continue
		}
		cwd, err := lookupProcessCwd(port.PID)
		if err != nil {
			continue
		}
		cwdByPID[port.PID] = cwd
	}

	for i := range ports {
		ports[i].Cwd = cwdByPID[ports[i].PID]
	}

	return ports, nil
}

func lookupProcessCwd(pid int) (string, error) {
	cmd := exec.Command("lsof", "-a", "-p", fmt.Sprintf("%d", pid), "-d", "cwd", "-Fn")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n"), nil
		}
	}
	return "", nil
}

func parsePort(name string) int {
	name = strings.Split(name, "->")[0]
	lastColon := strings.LastIndex(name, ":")
	if lastColon == -1 || lastColon == len(name)-1 {
		return 0
	}
	portStr := name[lastColon+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return port
}

func matchWorktree(paths []string, cwd string) string {
	cwd = filepath.Clean(cwd)
	best := ""
	for _, path := range paths {
		if isSubpath(path, cwd) {
			if len(path) > len(best) {
				best = path
			}
		}
	}
	return best
}

func isSubpath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func appendUniquePort(ports []servicePort, port servicePort) []servicePort {
	for _, existing := range ports {
		if existing.Port == port.Port {
			return ports
		}
	}
	return append(ports, port)
}

func mergePorts(destination, source map[string][]servicePort) {
	for path, ports := range source {
		for _, port := range ports {
			destination[path] = appendUniquePort(destination[path], port)
		}
	}
}
