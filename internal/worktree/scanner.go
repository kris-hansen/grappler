package worktree

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	Branch string
}

// ScanWorktrees scans a git repository for worktrees
func ScanWorktrees(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(out.String()), nil
}

// parseWorktreeList parses the output of `git worktree list --porcelain`
func parseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current Worktree

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if strings.HasPrefix(line, "detached") {
			current.Branch = "(detached)"
		}
	}

	// Add last worktree if present
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// GetWorktreeName extracts a meaningful name from a worktree path
func GetWorktreeName(path string) string {
	// Extract the last directory name
	base := filepath.Base(path)

	// Handle special cases
	if base == "core" || base == "web" {
		return "main"
	}

	return base
}

// IsInConductorWorkspace checks if a path is in the conductor/workspaces directory
func IsInConductorWorkspace(path string) bool {
	return strings.Contains(path, "conductor/workspaces/")
}

// ExtractConductorName extracts the worktree name from a conductor workspace path
// Example: /Users/krish/conductor/workspaces/core/dakar -> dakar
func ExtractConductorName(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "workspaces" && i+2 < len(parts) {
			return parts[i+2]
		}
	}
	return ""
}
