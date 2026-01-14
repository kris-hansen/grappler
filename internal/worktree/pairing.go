package worktree

import (
	"github.com/kris-hansen/grappler/internal/config"
)

// PairWorktrees pairs backend and frontend worktrees to create groups
func PairWorktrees(backendWorktrees, frontendWorktrees []Worktree) map[string]*config.Group {
	groups := make(map[string]*config.Group)

	// Create a map for quick lookup
	frontendMap := make(map[string]Worktree)
	for _, wt := range frontendWorktrees {
		frontendMap[wt.Path] = wt
	}

	// Track which frontend worktrees have been paired
	pairedFrontends := make(map[string]bool)

	// First pass: pair worktrees in conductor/workspaces
	for _, backend := range backendWorktrees {
		if !IsInConductorWorkspace(backend.Path) {
			continue
		}

		backendName := ExtractConductorName(backend.Path)
		if backendName == "" {
			continue
		}

		// Try to find matching frontend in conductor/workspaces
		for _, frontend := range frontendWorktrees {
			if !IsInConductorWorkspace(frontend.Path) {
				continue
			}

			frontendName := ExtractConductorName(frontend.Path)
			if frontendName != "" {
				// Create group with both backend and frontend
				groupName := backendName + "-" + frontendName
				groups[groupName] = &config.Group{
					Name: groupName,
					Backend: &config.Service{
						Directory: backend.Path,
						Branch:    backend.Branch,
						Command:   "go run cmd/api-server/main.go",
					},
					Frontend: &config.Service{
						Directory: frontend.Path,
						Branch:    frontend.Branch,
						Command:   "pnpm conductor:customer",
					},
				}
				pairedFrontends[frontend.Path] = true
				break // Found a match
			}
		}
	}

	// Second pass: pair main repositories
	var mainBackend *Worktree
	var mainFrontend *Worktree

	for _, backend := range backendWorktrees {
		name := GetWorktreeName(backend.Path)
		if name == "main" && !IsInConductorWorkspace(backend.Path) {
			mainBackend = &backend
			break
		}
	}

	for _, frontend := range frontendWorktrees {
		name := GetWorktreeName(frontend.Path)
		if name == "main" && !IsInConductorWorkspace(frontend.Path) {
			mainFrontend = &frontend
			break
		}
	}

	if mainBackend != nil && mainFrontend != nil {
		groups["main"] = &config.Group{
			Name: "main",
			Backend: &config.Service{
				Directory: mainBackend.Path,
				Branch:    mainBackend.Branch,
				Command:   "go run cmd/api-server/main.go",
			},
			Frontend: &config.Service{
				Directory: mainFrontend.Path,
				Branch:    mainFrontend.Branch,
				Command:   "pnpm conductor:customer",
			},
		}
		pairedFrontends[mainFrontend.Path] = true
	}

	// Third pass: create backend-only groups for unpaired worktrees
	for _, backend := range backendWorktrees {
		// Skip if already in a group
		alreadyPaired := false
		for _, group := range groups {
			if group.Backend != nil && group.Backend.Directory == backend.Path {
				alreadyPaired = true
				break
			}
		}

		if alreadyPaired {
			continue
		}

		// Create backend-only group
		name := GetWorktreeName(backend.Path)
		if name == "main" {
			continue // Already handled
		}

		groups[name] = &config.Group{
			Name: name,
			Backend: &config.Service{
				Directory: backend.Path,
				Branch:    backend.Branch,
				Command:   "go run cmd/api-server/main.go",
			},
		}
	}

	return groups
}
