package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmptyConfig(t *testing.T) {
	cfg, err := ParseConfig("")
	require.NoError(t, err)
	assert.Nil(t, cfg.Daemon)
	assert.Nil(t, cfg.TUI)
}

func TestParseFullConfig(t *testing.T) {
	tomlContent := `
[daemon]
port = 8080
poll_interval = 15

[daemon.workspaces]
paths = ["~/.openclaw/workspace-main", "/tmp/other"]

[tui]
daemon_url = "http://localhost:8080/api/v1"
refresh_interval = 10
theme = "dark"
`
	cfg, err := ParseConfig(tomlContent)
	require.NoError(t, err)

	require.NotNil(t, cfg.Daemon)
	assert.Equal(t, 8080, *cfg.Daemon.Port)
	assert.Equal(t, 15, *cfg.Daemon.PollInterval)
	require.NotNil(t, cfg.Daemon.Workspaces)
	assert.Len(t, cfg.Daemon.Workspaces.Paths, 2)

	require.NotNil(t, cfg.TUI)
	assert.Equal(t, "http://localhost:8080/api/v1", *cfg.TUI.DaemonURL)
	assert.Equal(t, 10, *cfg.TUI.RefreshInterval)
	assert.Equal(t, "dark", *cfg.TUI.Theme)
}

func TestParsePartialConfig(t *testing.T) {
	tomlContent := `
[daemon]
port = 9999
`
	cfg, err := ParseConfig(tomlContent)
	require.NoError(t, err)

	require.NotNil(t, cfg.Daemon)
	assert.Equal(t, 9999, *cfg.Daemon.Port)
	assert.Nil(t, cfg.Daemon.PollInterval)
	assert.Nil(t, cfg.Daemon.Workspaces)
	assert.Nil(t, cfg.TUI)
}

func TestParseInvalidConfig(t *testing.T) {
	_, err := ParseConfig("not valid [[[toml")
	assert.Error(t, err)
}

func TestGetDaemonURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *ConfigFile
		expected string
	}{
		{"nil config", nil, ""},
		{"nil TUI", &ConfigFile{}, ""},
		{"nil daemon_url", &ConfigFile{TUI: &TUIConfig{}}, ""},
		{"set url", &ConfigFile{TUI: &TUIConfig{DaemonURL: strPtr("http://test:9876/api/v1")}}, "http://test:9876/api/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.GetDaemonURL())
		})
	}
}

func TestGetRefreshInterval(t *testing.T) {
	tests := []struct {
		name     string
		config   *ConfigFile
		expected int
	}{
		{"nil config", nil, 0},
		{"nil TUI", &ConfigFile{}, 0},
		{"nil interval", &ConfigFile{TUI: &TUIConfig{}}, 0},
		{"set interval", &ConfigFile{TUI: &TUIConfig{RefreshInterval: intPtr(10)}}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.GetRefreshInterval())
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	// Use a temp directory as HOME
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// First call should create the file
	path, err := CreateDefaultConfig()
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	expectedPath := filepath.Join(tmpDir, ConfigDirName, ConfigFileName)
	assert.Equal(t, expectedPath, path)

	// File should exist and be parseable
	cfg, err := ParseConfig(mustReadFile(t, path))
	require.NoError(t, err)
	// Default config has commented-out values so no actual port/url values are set
	assert.Empty(t, cfg.GetDaemonURL())
	assert.Equal(t, 0, cfg.GetRefreshInterval())

	// Second call should return empty (already exists)
	path2, err := CreateDefaultConfig()
	require.NoError(t, err)
	assert.Empty(t, path2)
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
