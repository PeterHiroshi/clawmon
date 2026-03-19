package app

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// mockClient implements client.DaemonClient for testing.
type mockClient struct {
	healthResp     *models.HealthResponse
	healthErr      error
	workspaces     []models.WorkspaceInfo
	workspacesErr  error
	gitStatus      *models.GitStatus
	gitErr         error
	tasks          []models.TaskInfo
	tasksErr       error
	processes      []models.ProcessInfo
	processesErr   error
	envHealth      *models.EnvHealth
	envErr         error
	activity       []models.ActivityEvent
	activityErr    error
	systemRes      *models.SystemResources
	systemErr      error
}

func (m *mockClient) Health() (*models.HealthResponse, error) {
	return m.healthResp, m.healthErr
}
func (m *mockClient) ListWorkspaces() ([]models.WorkspaceInfo, error) {
	return m.workspaces, m.workspacesErr
}
func (m *mockClient) GetGitStatus(id string) (*models.GitStatus, error) {
	return m.gitStatus, m.gitErr
}
func (m *mockClient) GetTasks(id string) ([]models.TaskInfo, error) {
	return m.tasks, m.tasksErr
}
func (m *mockClient) GetProcesses(id string) ([]models.ProcessInfo, error) {
	return m.processes, m.processesErr
}
func (m *mockClient) GetEnvHealth(id string) (*models.EnvHealth, error) {
	return m.envHealth, m.envErr
}
func (m *mockClient) GetActivity(id string) ([]models.ActivityEvent, error) {
	return m.activity, m.activityErr
}
func (m *mockClient) GetSystemResources(id string) (*models.SystemResources, error) {
	return m.systemRes, m.systemErr
}
func (m *mockClient) SubscribeEvents(ctx context.Context) (<-chan models.SseEvent, error) {
	return make(chan models.SseEvent), nil
}

func defaultMock() *mockClient {
	return &mockClient{
		healthResp: &models.HealthResponse{Status: "ok", UptimeSeconds: 100, Version: "0.1.0"},
		workspaces: []models.WorkspaceInfo{
			{ID: "alpha", Path: "/ws/alpha", Name: "Alpha"},
		},
		gitStatus: &models.GitStatus{Branch: "main", IsClean: true},
		tasks:     []models.TaskInfo{},
		processes: []models.ProcessInfo{},
		envHealth: &models.EnvHealth{Tools: []models.EnvCheck{}, Hooks: []models.HookCheck{}},
		activity:  []models.ActivityEvent{},
	}
}

func TestNewModel(t *testing.T) {
	m := NewModel(defaultMock())
	assert.Equal(t, TabDashboard, m.ActiveTab)
	assert.True(t, m.Loading["init"])
	assert.NotNil(t, m.Client)
	assert.Nil(t, m.Workspaces)
}

func TestInit(t *testing.T) {
	m := NewModel(defaultMock())
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestHealthMsg(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24

	// Success case
	updated, _ := m.Update(HealthMsg{
		Response: &models.HealthResponse{Status: "ok", UptimeSeconds: 42, Version: "0.1.0"},
	})
	model := updated.(Model)
	assert.True(t, model.DaemonOnline)
	assert.Equal(t, uint64(42), model.Uptime)

	// Error case
	updated, _ = model.Update(HealthMsg{Err: assert.AnError})
	model = updated.(Model)
	assert.False(t, model.DaemonOnline)
	assert.NotNil(t, model.Errors["health"])
}

func TestWorkspacesMsg(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80

	updated, cmd := m.Update(WorkspacesMsg{
		Workspaces: []models.WorkspaceInfo{
			{ID: "alpha", Path: "/ws/alpha", Name: "Alpha"},
		},
	})
	model := updated.(Model)
	require.Len(t, model.Workspaces, 1)
	assert.Equal(t, "alpha", model.Workspaces[0].ID)
	assert.False(t, model.Loading["init"])
	assert.NotNil(t, cmd) // Should trigger workspace data fetch
}

func TestGitMsg(t *testing.T) {
	m := NewModel(defaultMock())
	updated, _ := m.Update(GitMsg{
		Status: &models.GitStatus{Branch: "develop", IsClean: false},
	})
	model := updated.(Model)
	require.NotNil(t, model.GitStatus)
	assert.Equal(t, "develop", model.GitStatus.Branch)
	assert.False(t, model.GitStatus.IsClean)
}

func TestTasksMsg(t *testing.T) {
	m := NewModel(defaultMock())
	updated, _ := m.Update(TasksMsg{
		Tasks: []models.TaskInfo{
			{Name: "test-task", Status: models.TaskStatusDone, TaskDir: "/tmp"},
		},
	})
	model := updated.(Model)
	require.Len(t, model.Tasks, 1)
	assert.Equal(t, "test-task", model.Tasks[0].Name)
}

func TestProcessesMsg(t *testing.T) {
	m := NewModel(defaultMock())
	updated, _ := m.Update(ProcessesMsg{
		Processes: []models.ProcessInfo{
			{PID: 123, State: models.ProcessStateRunning, Command: "cc"},
		},
	})
	model := updated.(Model)
	require.Len(t, model.Processes, 1)
}

func TestEnvMsg(t *testing.T) {
	m := NewModel(defaultMock())
	updated, _ := m.Update(EnvMsg{
		Health: &models.EnvHealth{
			Tools: []models.EnvCheck{{Tool: "git", Installed: true}},
		},
	})
	model := updated.(Model)
	require.NotNil(t, model.EnvHealth)
	require.Len(t, model.EnvHealth.Tools, 1)
}

func TestActivityMsg(t *testing.T) {
	m := NewModel(defaultMock())
	now := time.Now()
	updated, _ := m.Update(ActivityMsg{
		Events: []models.ActivityEvent{
			{Timestamp: now, EventType: models.ActivityGitCommit, Description: "test"},
		},
	})
	model := updated.(Model)
	require.Len(t, model.Activity, 1)
}

func TestSystemMsg(t *testing.T) {
	m := NewModel(defaultMock())
	updated, _ := m.Update(SystemMsg{
		Resources: &models.SystemResources{
			CPU:    models.CpuInfo{Model: "Test CPU", CoreCount: 4, UsagePercent: 50.0},
			Memory: models.MemoryInfo{TotalBytes: 16000000000, UsedBytes: 8000000000, AvailableBytes: 8000000000, UsagePercent: 50.0},
		},
	})
	model := updated.(Model)
	require.NotNil(t, model.SystemResources)
	assert.Equal(t, "Test CPU", model.SystemResources.CPU.Model)

	// Error case
	updated, _ = m.Update(SystemMsg{Err: assert.AnError})
	model = updated.(Model)
	assert.NotNil(t, model.Errors["system"])
}

func TestTabSwitching(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		startTab int
		expected int
	}{
		{"tab forward", "tab", TabDashboard, TabTasks},
		{"tab wrap", "tab", TabSystem, TabDashboard},
		{"shift+tab back", "shift+tab", TabTasks, TabDashboard},
		{"shift+tab wrap", "shift+tab", TabDashboard, TabSystem},
		{"press 1", "1", TabTasks, TabDashboard},
		{"press 2", "2", TabDashboard, TabTasks},
		{"press 3", "3", TabDashboard, TabGit},
		{"press 4", "4", TabDashboard, TabActivity},
		{"press 5", "5", TabDashboard, TabSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(defaultMock())
			m.ActiveTab = tt.startTab
			m.Width = 80

			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			// Handle special keys
			if tt.key == "tab" {
				updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
			} else if tt.key == "shift+tab" {
				updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
			}
			model := updated.(Model)
			assert.Equal(t, tt.expected, model.ActiveTab)
		})
	}
}

func TestQuit(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.NotNil(t, cmd)
}

func TestHelpToggle(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	model := updated.(Model)
	assert.True(t, model.ShowHelp)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	model = updated.(Model)
	assert.False(t, model.ShowHelp)
}

func TestRefresh(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Workspaces = []models.WorkspaceInfo{{ID: "alpha"}}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	assert.NotNil(t, cmd)
}

func TestNavigateDown(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.ActiveTab = TabTasks
	m.Tasks = []models.TaskInfo{
		{Name: "a", TaskDir: "/tmp"},
		{Name: "b", TaskDir: "/tmp"},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := updated.(Model)
	assert.Equal(t, 1, model.SelectedTask)

	// Should not go past end
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model = updated.(Model)
	assert.Equal(t, 1, model.SelectedTask)
}

func TestNavigateUp(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.ActiveTab = TabTasks
	m.Tasks = []models.TaskInfo{
		{Name: "a", TaskDir: "/tmp"},
		{Name: "b", TaskDir: "/tmp"},
	}
	m.SelectedTask = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model := updated.(Model)
	assert.Equal(t, 0, model.SelectedTask)

	// Should not go below 0
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model = updated.(Model)
	assert.Equal(t, 0, model.SelectedTask)
}

func TestEnterDetail(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.ActiveTab = TabTasks
	m.Tasks = []models.TaskInfo{{Name: "test", TaskDir: "/tmp"}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	assert.True(t, model.ShowDetail)
}

func TestEscClosesDetail(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.ShowDetail = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := updated.(Model)
	assert.False(t, model.ShowDetail)
}

func TestWindowSizeMsg(t *testing.T) {
	m := NewModel(defaultMock())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)
	assert.Equal(t, 120, model.Width)
	assert.Equal(t, 40, model.Height)
}

func TestTickMsg(t *testing.T) {
	m := NewModel(defaultMock())
	m.Workspaces = []models.WorkspaceInfo{{ID: "alpha"}}
	_, cmd := m.Update(TickMsg(time.Now()))
	assert.NotNil(t, cmd)
}

func TestSseMsg(t *testing.T) {
	m := NewModel(defaultMock())
	m.Workspaces = []models.WorkspaceInfo{{ID: "alpha"}}

	tests := []struct {
		eventType string
	}{
		{"git_status"},
		{"git_commit"},
		{"task_update"},
		{"process_update"},
		{"env_update"},
		{"system_update"},
		{"unknown_event"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			_, cmd := m.Update(SseMsg{Event: models.SseEvent{EventType: tt.eventType}})
			assert.NotNil(t, cmd)
		})
	}
}

func TestSseMsgNoWorkspaces(t *testing.T) {
	m := NewModel(defaultMock())
	_, cmd := m.Update(SseMsg{Event: models.SseEvent{EventType: "test"}})
	assert.Nil(t, cmd)
}

func TestViewRendersTabBar(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24
	m.Workspaces = []models.WorkspaceInfo{{ID: "alpha", Path: "/ws/alpha", Name: "Alpha"}}

	view := m.View()
	assert.Contains(t, view, "Dashboard")
	assert.Contains(t, view, "Tasks")
	assert.Contains(t, view, "Git")
	assert.Contains(t, view, "Activity")
	assert.Contains(t, view, "System")
}

func TestViewStatusBar(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24
	m.DaemonOnline = true
	m.Workspaces = []models.WorkspaceInfo{{ID: "alpha", Path: "/ws/alpha", Name: "Alpha"}}

	view := m.View()
	assert.Contains(t, view, "daemon")
	assert.Contains(t, view, "Alpha")
}

func TestViewLoadingState(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestViewHelpOverlay(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24
	m.ShowHelp = true

	view := m.View()
	assert.Contains(t, view, "Key Bindings")
	assert.Contains(t, view, "Tab")
}

func TestViewDetailOverlay(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24
	m.ShowDetail = true
	m.ActiveTab = TabTasks
	m.Tasks = []models.TaskInfo{{Name: "detail-task", Status: models.TaskStatusDone, TaskDir: "/tmp"}}

	view := m.View()
	assert.Contains(t, view, "detail-task")
}

func TestViewAllTabs(t *testing.T) {
	m := NewModel(defaultMock())
	m.Width = 80
	m.Height = 24
	m.Workspaces = []models.WorkspaceInfo{{ID: "alpha", Path: "/ws/alpha", Name: "Alpha"}}
	m.GitStatus = &models.GitStatus{Branch: "main", IsClean: true}

	for tab := 0; tab < TabCount; tab++ {
		m.ActiveTab = tab
		view := m.View()
		assert.NotEmpty(t, view, "tab %d should render", tab)
	}
}

func TestViewZeroWidth(t *testing.T) {
	m := NewModel(defaultMock())
	view := m.View()
	assert.Equal(t, "Loading...", view)
}

func TestErrorStates(t *testing.T) {
	m := NewModel(defaultMock())

	// Each error message should store in Errors map
	msgs := []tea.Msg{
		GitMsg{Err: assert.AnError},
		TasksMsg{Err: assert.AnError},
		ProcessesMsg{Err: assert.AnError},
		EnvMsg{Err: assert.AnError},
		ActivityMsg{Err: assert.AnError},
		SystemMsg{Err: assert.AnError},
	}

	for _, msg := range msgs {
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}

	assert.Len(t, m.Errors, 6)
}
