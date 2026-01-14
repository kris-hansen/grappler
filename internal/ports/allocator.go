package ports

import (
	"fmt"
	"net"

	"github.com/kris-hansen/grappler/internal/config"
)

const (
	// BackendPortStart is the starting port for backend services
	BackendPortStart = 8000
	// BackendPortEnd is the ending port for backend services
	BackendPortEnd = 8999
	// FrontendPortStart is the starting port for frontend services
	FrontendPortStart = 5000
	// FrontendPortEnd is the ending port for frontend services
	FrontendPortEnd = 5999
)

// Allocator manages port allocation
type Allocator struct {
	state *config.State
}

// NewAllocator creates a new port allocator
func NewAllocator(state *config.State) *Allocator {
	return &Allocator{state: state}
}

// AllocateBackendPort finds and allocates an available backend port
func (a *Allocator) AllocateBackendPort() (int, error) {
	usedPorts := a.getUsedBackendPorts()

	for port := BackendPortStart; port <= BackendPortEnd; port++ {
		if usedPorts[port] {
			continue
		}

		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available backend ports in range %d-%d", BackendPortStart, BackendPortEnd)
}

// AllocateFrontendPort finds and allocates an available frontend port
func (a *Allocator) AllocateFrontendPort() (int, error) {
	usedPorts := a.getUsedFrontendPorts()

	for port := FrontendPortStart; port <= FrontendPortEnd; port++ {
		if usedPorts[port] {
			continue
		}

		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available frontend ports in range %d-%d", FrontendPortStart, FrontendPortEnd)
}

// getUsedBackendPorts returns a map of currently allocated backend ports
func (a *Allocator) getUsedBackendPorts() map[int]bool {
	used := make(map[int]bool)

	for _, groupState := range a.state.Groups {
		if groupState.BackendPort > 0 {
			used[groupState.BackendPort] = true
		}
	}

	return used
}

// getUsedFrontendPorts returns a map of currently allocated frontend ports
func (a *Allocator) getUsedFrontendPorts() map[int]bool {
	used := make(map[int]bool)

	for _, groupState := range a.state.Groups {
		if groupState.FrontendPort > 0 {
			used[groupState.FrontendPort] = true
		}
	}

	return used
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
