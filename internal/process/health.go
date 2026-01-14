package process

import (
	"fmt"
	"net/http"
	"time"
)

// HealthChecker checks service health
type HealthChecker struct {
	client *http.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// WaitForHealth waits for a service to become healthy
func (h *HealthChecker) WaitForHealth(port int, maxWait time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d", port)
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		resp, err := h.client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("service did not become healthy within %s", maxWait)
}
