package integration

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PeterHiroshi/clawmon/tui/internal/app"
	"github.com/PeterHiroshi/clawmon/tui/internal/client"
)

func TestAppWithRealClient(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)

	// Simulate Init
	cmd := m.Init()
	require.NotNil(t, cmd)

	// Execute Init commands manually by fetching data directly
	health, err := c.Health()
	require.NoError(t, err)

	updated, _ := m.Update(app.HealthMsg{Response: health})
	m = updated.(app.Model)
	assert.True(t, m.DaemonOnline)

	ws, err := c.ListWorkspaces()
	require.NoError(t, err)

	updated, cmd = m.Update(app.WorkspacesMsg{Workspaces: ws})
	m = updated.(app.Model)
	require.Len(t, m.Workspaces, 2)

	// Fetch workspace data
	wsID := m.Workspaces[0].ID
	git, err := c.GetGitStatus(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.GitMsg{Status: git})
	m = updated.(app.Model)
	assert.Equal(t, "develop", m.GitStatus.Branch)

	tasks, err := c.GetTasks(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.TasksMsg{Tasks: tasks})
	m = updated.(app.Model)
	assert.Len(t, m.Tasks, 2)

	procs, err := c.GetProcesses(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.ProcessesMsg{Processes: procs})
	m = updated.(app.Model)
	assert.Len(t, m.Processes, 1)

	env, err := c.GetEnvHealth(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.EnvMsg{Health: env})
	m = updated.(app.Model)
	assert.NotNil(t, m.EnvHealth)

	activity, err := c.GetActivity(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.ActivityMsg{Events: activity})
	m = updated.(app.Model)
	assert.Len(t, m.Activity, 3)
}

func TestAppViewRendersWithRealData(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 100
	m.Height = 30

	// Load data
	ws, _ := c.ListWorkspaces()
	updated, _ := m.Update(app.WorkspacesMsg{Workspaces: ws})
	m = updated.(app.Model)

	health, _ := c.Health()
	updated, _ = m.Update(app.HealthMsg{Response: health})
	m = updated.(app.Model)

	git, _ := c.GetGitStatus("alpha")
	updated, _ = m.Update(app.GitMsg{Status: git})
	m = updated.(app.Model)

	tasks, _ := c.GetTasks("alpha")
	updated, _ = m.Update(app.TasksMsg{Tasks: tasks})
	m = updated.(app.Model)

	procs, _ := c.GetProcesses("alpha")
	updated, _ = m.Update(app.ProcessesMsg{Processes: procs})
	m = updated.(app.Model)

	env, _ := c.GetEnvHealth("alpha")
	updated, _ = m.Update(app.EnvMsg{Health: env})
	m = updated.(app.Model)

	activity, _ := c.GetActivity("alpha")
	updated, _ = m.Update(app.ActivityMsg{Events: activity})
	m = updated.(app.Model)

	// Test all tab views render without panic
	for tab := 0; tab < 4; tab++ {
		m.ActiveTab = tab
		view := m.View()
		assert.NotEmpty(t, view, "tab %d should render content", tab)
	}
}

func TestAppNavigationWithRealData(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 100
	m.Height = 30

	// Load tasks
	tasks, _ := c.GetTasks("alpha")
	updated, _ := m.Update(app.TasksMsg{Tasks: tasks})
	m = updated.(app.Model)

	// Switch to tasks tab
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(app.Model)
	assert.Equal(t, 1, m.ActiveTab)

	// Navigate down
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(app.Model)
	assert.Equal(t, 1, m.SelectedTask)

	// Open detail
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(app.Model)
	assert.True(t, m.ShowDetail)

	view := m.View()
	assert.Contains(t, view, "fix-bug")

	// Close detail
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(app.Model)
	assert.False(t, m.ShowDetail)
}

// Verify the mock daemon is accessible (sanity check)
func TestMockDaemonIsAccessible(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()
	assert.NotEmpty(t, server.URL)

	c := client.NewHTTPClient(server.URL)
	health, err := c.Health()
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)
}
