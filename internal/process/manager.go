package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/krish/grappler/internal/config"
)

// Manager handles process lifecycle
type Manager struct {
	logsDir string
}

// NewManager creates a new process manager
func NewManager(logsDir string) *Manager {
	return &Manager{
		logsDir: logsDir,
	}
}

// StartService starts a service and returns its PID
func (m *Manager) StartService(service *config.Service, serviceName, groupName string, envVars map[string]string) (int, error) {
	if service == nil {
		return 0, nil
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(m.logsDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Open log file
	logPath := filepath.Join(m.logsDir, fmt.Sprintf("%s-%s.log", groupName, serviceName))
	logFile, err := os.Create(logPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create log file: %w", err)
	}

	// Parse command
	cmdParts := parseCommand(service.Command)
	if len(cmdParts) == 0 {
		return 0, fmt.Errorf("empty command")
	}

	// Create command
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	cmd.Dir = service.Directory
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Set environment variables
	cmd.Env = os.Environ()

	// Add service-specific env vars from config
	if service.Env != nil {
		for key, value := range service.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Add runtime env vars (ports)
	for key, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("failed to start process: %w", err)
	}

	// Return PID
	pid := cmd.Process.Pid

	// Launch goroutine to wait for process and close log file
	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	return pid, nil
}

// StopProcess stops a process by sending SIGTERM
func (m *Manager) StopProcess(pid int) error {
	if pid == 0 {
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// IsProcessRunning checks if a process is running
func (m *Manager) IsProcessRunning(pid int) bool {
	if pid == 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// parseCommand splits a command string into parts
func parseCommand(cmd string) []string {
	// Simple split on spaces - could be enhanced for quoted strings
	parts := []string{}
	current := ""
	inQuote := false

	for _, char := range cmd {
		if char == '"' {
			inQuote = !inQuote
		} else if char == ' ' && !inQuote {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
