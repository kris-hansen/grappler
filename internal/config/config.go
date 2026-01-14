package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the grappler configuration file
type Config struct {
	Version string            `yaml:"version"`
	Groups  map[string]*Group `yaml:"groups"`
	Proxy   *ProxyConfig      `yaml:"proxy,omitempty"`
}

// Group represents a worktree group (backend + frontend pair)
type Group struct {
	Name     string   `yaml:"name"`
	Backend  *Service `yaml:"backend"`
	Frontend *Service `yaml:"frontend,omitempty"`
}

// Service represents a single service (backend or frontend)
type Service struct {
	Directory string            `yaml:"directory"`
	Branch    string            `yaml:"branch,omitempty"`
	Command   string            `yaml:"command"`
	Env       map[string]string `yaml:"env,omitempty"`
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	Enabled             bool `yaml:"enabled"`
	UseExistingConductor bool `yaml:"use_existing_conductor"`
}

// Load reads the config file from the specified path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to the specified path
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the grappler config file
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".grappler", "config.yaml")
	}
	return filepath.Join(home, ".grappler", "config.yaml")
}
