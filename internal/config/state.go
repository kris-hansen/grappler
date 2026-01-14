package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// State represents the runtime state of grappler
type State struct {
	mu     sync.RWMutex
	Groups map[string]*GroupState `json:"groups"`
}

// GroupState represents the runtime state of a single group
type GroupState struct {
	BackendPort  int `json:"backend_port,omitempty"`
	FrontendPort int `json:"frontend_port,omitempty"`
	BackendPID   int `json:"backend_pid,omitempty"`
	FrontendPID  int `json:"frontend_pid,omitempty"`
	Running      bool `json:"running"`
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		Groups: make(map[string]*GroupState),
	}
}

// LoadState reads the state file from the specified path
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if state.Groups == nil {
		state.Groups = make(map[string]*GroupState)
	}

	return &state, nil
}

// Save writes the state to the specified path
func (s *State) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// GetGroup returns the state for a specific group
func (s *State) GetGroup(name string) *GroupState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Groups[name]
}

// SetGroup sets the state for a specific group
func (s *State) SetGroup(name string, state *GroupState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Groups[name] = state
}

// DeleteGroup removes a group from the state
func (s *State) DeleteGroup(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Groups, name)
}

// GetStatePath returns the path to the grappler state file
func GetStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".grappler", "state.json")
	}
	return filepath.Join(home, ".grappler", "state.json")
}

// GetLogsDir returns the path to the grappler logs directory
func GetLogsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".grappler", "logs")
	}
	return filepath.Join(home, ".grappler", "logs")
}
