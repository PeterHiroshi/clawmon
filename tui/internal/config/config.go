// Package config provides configuration file support for the clawmon TUI.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	// ConfigDirName is the config directory under the user's home.
	ConfigDirName = ".clawmon"
	// ConfigFileName is the config file name.
	ConfigFileName = "config.toml"
)

// DefaultConfigContent is the default TOML written on first run.
const DefaultConfigContent = `# clawmon configuration file
# See README.md for full documentation

[daemon]
# port = 9876
# poll_interval = 30

[daemon.workspaces]
# paths = [
#   "~/.openclaw/workspace-main",
# ]

[tui]
# daemon_url = "http://127.0.0.1:9876/api/v1"
# refresh_interval = 5
# theme = "dark"
`

// ConfigFile represents the root configuration structure.
type ConfigFile struct {
	Daemon *DaemonConfig `toml:"daemon"`
	TUI    *TUIConfig    `toml:"tui"`
}

// DaemonConfig is the [daemon] section.
type DaemonConfig struct {
	Port         *int              `toml:"port"`
	PollInterval *int              `toml:"poll_interval"`
	Workspaces   *WorkspacesConfig `toml:"workspaces"`
}

// WorkspacesConfig is the [daemon.workspaces] section.
type WorkspacesConfig struct {
	Paths []string `toml:"paths"`
}

// TUIConfig is the [tui] section.
type TUIConfig struct {
	DaemonURL       *string `toml:"daemon_url"`
	RefreshInterval *int    `toml:"refresh_interval"`
	Theme           *string `toml:"theme"`
}

// ConfigFilePath returns the path to ~/.clawmon/config.toml.
func ConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ConfigDirName, ConfigFileName), nil
}

// LoadConfig reads and parses the configuration file.
// Returns nil (no error) if the file does not exist.
func LoadConfig() (*ConfigFile, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, nil
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return nil, nil
	}

	var config ConfigFile
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return &config, nil
}

// ParseConfig parses TOML content into a ConfigFile.
// Useful for testing without filesystem access.
func ParseConfig(content string) (*ConfigFile, error) {
	var config ConfigFile
	if _, err := toml.Decode(content, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &config, nil
}

// CreateDefaultConfig writes a default config file if none exists.
// Returns the path where the config was created, or empty string if it already exists.
func CreateDefaultConfig() (string, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(path); statErr == nil {
		return "", nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(DefaultConfigContent), 0o644); err != nil {
		return "", fmt.Errorf("writing default config to %s: %w", path, err)
	}

	return path, nil
}

// GetDaemonURL returns the daemon URL from config, or empty string if not set.
func (c *ConfigFile) GetDaemonURL() string {
	if c == nil || c.TUI == nil || c.TUI.DaemonURL == nil {
		return ""
	}
	return *c.TUI.DaemonURL
}

// GetRefreshInterval returns the refresh interval from config, or 0 if not set.
func (c *ConfigFile) GetRefreshInterval() int {
	if c == nil || c.TUI == nil || c.TUI.RefreshInterval == nil {
		return 0
	}
	return *c.TUI.RefreshInterval
}
