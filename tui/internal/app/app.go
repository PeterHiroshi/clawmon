// Package app implements the Bubble Tea application model for the clawmon TUI.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/PeterHiroshi/clawmon/tui/internal/client"
	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/PeterHiroshi/clawmon/tui/internal/views"
)

// Tab indices.
const (
	TabDashboard = 0
	TabTasks     = 1
	TabGit       = 2
	TabActivity  = 3
	TabSystem    = 4
	TabCount     = 5
)

// Model is the Bubble Tea model for the clawmon TUI.
type Model struct {
	// Navigation
	ActiveTab int
	Width     int
	Height    int

	// Workspace state
	Workspaces        []models.WorkspaceInfo
	SelectedWorkspace int

	// Per-tab data
	GitStatus       *models.GitStatus
	Tasks           []models.TaskInfo
	Processes       []models.ProcessInfo
	EnvHealth       *models.EnvHealth
	Activity        []models.ActivityEvent
	SystemResources *models.SystemResources

	// UI state
	Loading      map[string]bool
	Errors       map[string]error
	DaemonOnline bool
	Uptime       uint64
	LastRefresh  time.Time
	ShowHelp     bool
	ShowDetail   bool
	DetailContent string

	// Stale data tracking
	StaleData        map[string]bool
	LastSuccessful   map[string]time.Time

	// Reconnection state
	Reconnect *client.ReconnectState

	// Per-tab selection indices
	SelectedTask     int
	SelectedCommit   int
	SelectedActivity int

	// Client
	Client client.DaemonClient
}

// NewModel creates a new app model with the given daemon client.
func NewModel(c client.DaemonClient) Model {
	return Model{
		ActiveTab:      TabDashboard,
		Loading:        map[string]bool{"init": true},
		Errors:         make(map[string]error),
		StaleData:      make(map[string]bool),
		LastSuccessful: make(map[string]time.Time),
		Reconnect:      client.NewReconnectState(),
		Client:         c,
	}
}

// Message types for tea.Msg.

// HealthMsg carries the result of a health check.
type HealthMsg struct {
	Response *models.HealthResponse
	Err      error
}

// WorkspacesMsg carries the list of workspaces.
type WorkspacesMsg struct {
	Workspaces []models.WorkspaceInfo
	Err        error
}

// GitMsg carries git status data.
type GitMsg struct {
	Status *models.GitStatus
	Err    error
}

// TasksMsg carries task list data.
type TasksMsg struct {
	Tasks []models.TaskInfo
	Err   error
}

// ProcessesMsg carries process list data.
type ProcessesMsg struct {
	Processes []models.ProcessInfo
	Err       error
}

// EnvMsg carries environment health data.
type EnvMsg struct {
	Health *models.EnvHealth
	Err    error
}

// ActivityMsg carries activity timeline data.
type ActivityMsg struct {
	Events []models.ActivityEvent
	Err    error
}

// SystemMsg carries system resource data.
type SystemMsg struct {
	Resources *models.SystemResources
	Err       error
}

// TickMsg triggers a periodic refresh.
type TickMsg time.Time

// SseMsg carries a server-sent event.
type SseMsg struct {
	Event models.SseEvent
}

// ErrMsg carries a generic error.
type ErrMsg struct {
	Err error
}

// Init returns the initial commands to run on startup.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchHealth(),
		m.fetchWorkspaces(),
		tickCmd(),
	)
}

// fetchHealth creates a command to check daemon health.
func (m Model) fetchHealth() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.Client.Health()
		return HealthMsg{Response: resp, Err: err}
	}
}

// fetchWorkspaces creates a command to list workspaces.
func (m Model) fetchWorkspaces() tea.Cmd {
	return func() tea.Msg {
		ws, err := m.Client.ListWorkspaces()
		return WorkspacesMsg{Workspaces: ws, Err: err}
	}
}

// fetchWorkspaceData returns commands to fetch all data for the selected workspace.
func (m Model) fetchWorkspaceData() tea.Cmd {
	if len(m.Workspaces) == 0 || m.SelectedWorkspace >= len(m.Workspaces) {
		return nil
	}
	wsID := m.Workspaces[m.SelectedWorkspace].ID
	return tea.Batch(
		m.fetchGit(wsID),
		m.fetchTasks(wsID),
		m.fetchProcesses(wsID),
		m.fetchEnv(wsID),
		m.fetchActivity(wsID),
		m.fetchSystem(wsID),
	)
}

func (m Model) fetchGit(id string) tea.Cmd {
	return func() tea.Msg {
		status, err := m.Client.GetGitStatus(id)
		return GitMsg{Status: status, Err: err}
	}
}

func (m Model) fetchTasks(id string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.Client.GetTasks(id)
		return TasksMsg{Tasks: tasks, Err: err}
	}
}

func (m Model) fetchProcesses(id string) tea.Cmd {
	return func() tea.Msg {
		procs, err := m.Client.GetProcesses(id)
		return ProcessesMsg{Processes: procs, Err: err}
	}
}

func (m Model) fetchEnv(id string) tea.Cmd {
	return func() tea.Msg {
		health, err := m.Client.GetEnvHealth(id)
		return EnvMsg{Health: health, Err: err}
	}
}

func (m Model) fetchActivity(id string) tea.Cmd {
	return func() tea.Msg {
		events, err := m.Client.GetActivity(id)
		return ActivityMsg{Events: events, Err: err}
	}
}

func (m Model) fetchSystem(id string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.Client.GetSystemResources(id)
		return SystemMsg{Resources: res, Err: err}
	}
}

// tickCmd returns a command that sends a TickMsg after the refresh interval.
func tickCmd() tea.Cmd {
	return tea.Tick(views.RefreshInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages and returns the updated model and any commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case HealthMsg:
		if msg.Err != nil {
			m.DaemonOnline = false
			m.Reconnect.MarkDisconnected()
			m.Errors["health"] = msg.Err
			// Mark all data as stale when daemon goes offline
			for _, key := range []string{"git", "tasks", "processes", "env", "activity", "system"} {
				if _, ok := m.LastSuccessful[key]; ok {
					m.StaleData[key] = true
				}
			}
		} else {
			wasOffline := !m.DaemonOnline
			m.DaemonOnline = true
			m.Reconnect.MarkConnected()
			m.Uptime = msg.Response.UptimeSeconds
			delete(m.Errors, "health")
			// Clear stale flags on reconnection
			if wasOffline {
				for key := range m.StaleData {
					delete(m.StaleData, key)
				}
				return m, m.fetchWorkspaceData()
			}
		}
		return m, nil

	case WorkspacesMsg:
		if msg.Err != nil {
			m.Errors["workspaces"] = msg.Err
		} else {
			m.Workspaces = msg.Workspaces
			delete(m.Errors, "workspaces")
			delete(m.Loading, "init")
			return m, m.fetchWorkspaceData()
		}
		return m, nil

	case GitMsg:
		if msg.Err != nil {
			m.Errors["git"] = msg.Err
		} else {
			m.GitStatus = msg.Status
			delete(m.Errors, "git")
			delete(m.StaleData, "git")
			m.LastSuccessful["git"] = time.Now()
		}
		m.LastRefresh = time.Now()
		return m, nil

	case TasksMsg:
		if msg.Err != nil {
			m.Errors["tasks"] = msg.Err
		} else {
			m.Tasks = msg.Tasks
			delete(m.Errors, "tasks")
			delete(m.StaleData, "tasks")
			m.LastSuccessful["tasks"] = time.Now()
		}
		return m, nil

	case ProcessesMsg:
		if msg.Err != nil {
			m.Errors["processes"] = msg.Err
		} else {
			m.Processes = msg.Processes
			delete(m.Errors, "processes")
			delete(m.StaleData, "processes")
			m.LastSuccessful["processes"] = time.Now()
		}
		return m, nil

	case EnvMsg:
		if msg.Err != nil {
			m.Errors["env"] = msg.Err
		} else {
			m.EnvHealth = msg.Health
			delete(m.Errors, "env")
			delete(m.StaleData, "env")
			m.LastSuccessful["env"] = time.Now()
		}
		return m, nil

	case ActivityMsg:
		if msg.Err != nil {
			m.Errors["activity"] = msg.Err
		} else {
			m.Activity = msg.Events
			delete(m.Errors, "activity")
			delete(m.StaleData, "activity")
			m.LastSuccessful["activity"] = time.Now()
		}
		return m, nil

	case SystemMsg:
		if msg.Err != nil {
			m.Errors["system"] = msg.Err
		} else {
			m.SystemResources = msg.Resources
			delete(m.Errors, "system")
			delete(m.StaleData, "system")
			m.LastSuccessful["system"] = time.Now()
		}
		return m, nil

	case TickMsg:
		return m, tea.Batch(
			m.fetchHealth(),
			m.fetchWorkspaceData(),
			tickCmd(),
		)

	case SseMsg:
		return m.handleSseMsg(msg)
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If help overlay is showing, any key closes it
	if m.ShowHelp && msg.String() != "?" {
		m.ShowHelp = false
		return m, nil
	}

	// If detail view is showing, Esc closes it
	if m.ShowDetail {
		if msg.String() == "esc" || msg.String() == "q" {
			m.ShowDetail = false
			m.DetailContent = ""
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.ActiveTab = (m.ActiveTab + 1) % TabCount
		return m, nil

	case "shift+tab":
		m.ActiveTab = (m.ActiveTab - 1 + TabCount) % TabCount
		return m, nil

	case "1":
		m.ActiveTab = TabDashboard
		return m, nil
	case "2":
		m.ActiveTab = TabTasks
		return m, nil
	case "3":
		m.ActiveTab = TabGit
		return m, nil
	case "4":
		m.ActiveTab = TabActivity
		return m, nil
	case "5":
		m.ActiveTab = TabSystem
		return m, nil

	case "j", "down":
		m.navigateDown()
		return m, nil

	case "k", "up":
		m.navigateUp()
		return m, nil

	case "enter":
		m.openDetail()
		return m, nil

	case "esc":
		m.ShowDetail = false
		m.DetailContent = ""
		return m, nil

	case "w":
		if len(m.Workspaces) > 1 {
			m.SelectedWorkspace = (m.SelectedWorkspace + 1) % len(m.Workspaces)
			return m, m.fetchWorkspaceData()
		}
		return m, nil

	case "r":
		return m, tea.Batch(m.fetchHealth(), m.fetchWorkspaceData())

	case "?":
		m.ShowHelp = !m.ShowHelp
		return m, nil
	}

	return m, nil
}

func (m *Model) navigateDown() {
	switch m.ActiveTab {
	case TabTasks:
		if m.SelectedTask < len(m.Tasks)-1 {
			m.SelectedTask++
		}
	case TabGit:
		if m.GitStatus != nil && m.SelectedCommit < len(m.GitStatus.RecentCommits)-1 {
			m.SelectedCommit++
		}
	case TabActivity:
		if m.SelectedActivity < len(m.Activity)-1 {
			m.SelectedActivity++
		}
	}
}

func (m *Model) navigateUp() {
	switch m.ActiveTab {
	case TabTasks:
		if m.SelectedTask > 0 {
			m.SelectedTask--
		}
	case TabGit:
		if m.SelectedCommit > 0 {
			m.SelectedCommit--
		}
	case TabActivity:
		if m.SelectedActivity > 0 {
			m.SelectedActivity--
		}
	}
}

func (m *Model) openDetail() {
	switch m.ActiveTab {
	case TabTasks:
		if m.SelectedTask < len(m.Tasks) {
			m.ShowDetail = true
		}
	case TabGit:
		if m.GitStatus != nil && m.SelectedCommit < len(m.GitStatus.RecentCommits) {
			m.ShowDetail = true
		}
	}
}

func (m Model) handleSseMsg(msg SseMsg) (tea.Model, tea.Cmd) {
	if len(m.Workspaces) == 0 || m.SelectedWorkspace >= len(m.Workspaces) {
		return m, nil
	}
	wsID := m.Workspaces[m.SelectedWorkspace].ID

	switch msg.Event.EventType {
	case "git_status", "git_commit", "git_push":
		return m, m.fetchGit(wsID)
	case "task_update", "task_started", "task_completed", "task_failed":
		return m, m.fetchTasks(wsID)
	case "process_update", "process_started", "process_stopped":
		return m, m.fetchProcesses(wsID)
	case "env_update":
		return m, m.fetchEnv(wsID)
	case "system_update":
		return m, m.fetchSystem(wsID)
	default:
		return m, m.fetchWorkspaceData()
	}
}

// View renders the current state of the model.
func (m Model) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	var sections []string

	// Tab bar
	tabBar := views.RenderTabBar(m.ActiveTab, m.Width)
	sections = append(sections, tabBar)

	// Content area height (minus tab bar and status bar)
	contentHeight := m.Height - 4

	// Help overlay
	if m.ShowHelp {
		sections = append(sections, m.renderHelp())
		sections = append(sections, m.renderStatusBar())
		return strings.Join(sections, "\n")
	}

	// Detail overlay
	if m.ShowDetail {
		sections = append(sections, m.renderDetail())
		sections = append(sections, m.renderStatusBar())
		return strings.Join(sections, "\n")
	}

	// Loading state
	if m.Loading["init"] {
		sections = append(sections, "\n  Loading...")
		sections = append(sections, m.renderStatusBar())
		return strings.Join(sections, "\n")
	}

	// Tab content
	switch m.ActiveTab {
	case TabDashboard:
		wsPath := ""
		wsName := ""
		if len(m.Workspaces) > 0 && m.SelectedWorkspace < len(m.Workspaces) {
			wsPath = m.Workspaces[m.SelectedWorkspace].Path
			wsName = m.Workspaces[m.SelectedWorkspace].Name
		}
		sections = append(sections, views.RenderDashboard(views.DashboardData{
			WorkspacePath:  wsPath,
			WorkspaceName:  wsName,
			WorkspaceCount: len(m.Workspaces),
			WorkspaceIndex: m.SelectedWorkspace,
			DaemonOnline:   m.DaemonOnline,
			Uptime:         m.Uptime,
			GitStatus:      m.GitStatus,
			Tasks:          m.Tasks,
			Processes:      m.Processes,
			EnvHealth:      m.EnvHealth,
			Activity:       m.Activity,
		}, m.Width, contentHeight))

	case TabTasks:
		sections = append(sections, views.RenderTasks(m.Tasks, m.SelectedTask, m.Width, contentHeight))

	case TabGit:
		sections = append(sections, views.RenderGit(m.GitStatus, m.SelectedCommit, m.Width, contentHeight))

	case TabActivity:
		sections = append(sections, views.RenderActivity(m.Activity, m.SelectedActivity, m.Width, contentHeight))

	case TabSystem:
		sections = append(sections, views.RenderSystem(m.SystemResources, m.Width, contentHeight))
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return strings.Join(sections, "\n")
}

func (m Model) renderStatusBar() string {
	dot := views.ConnectionDot(m.DaemonOnline)
	refreshStr := "never"
	if !m.LastRefresh.IsZero() {
		refreshStr = views.FormatTimestamp(m.LastRefresh)
	}

	wsName := "—"
	if len(m.Workspaces) > 0 && m.SelectedWorkspace < len(m.Workspaces) {
		wsName = m.Workspaces[m.SelectedWorkspace].Name
		if len(m.Workspaces) > 1 {
			wsName = fmt.Sprintf("[%d/%d] %s", m.SelectedWorkspace+1, len(m.Workspaces), wsName)
		}
	}

	tabName := views.TabNames[m.ActiveTab]

	// Show reconnection status or stale data indicator
	statusExtra := ""
	if !m.DaemonOnline {
		if msg := m.Reconnect.RetryMessage(); msg != "" {
			statusExtra = "  " + msg
		}
	} else if m.hasStaleData() {
		statusExtra = "  [stale]"
	}

	left := fmt.Sprintf(" %s daemon  %s  refreshed: %s%s", dot, wsName, refreshStr, statusExtra)
	right := fmt.Sprintf("%s  ? help  q quit ", tabName)

	gap := m.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return views.StatusBarStyle.Width(m.Width).Render(left + strings.Repeat(" ", gap) + right)
}

// hasStaleData returns true if any data source is marked as stale.
func (m Model) hasStaleData() bool {
	for _, stale := range m.StaleData {
		if stale {
			return true
		}
	}
	return false
}

func (m Model) renderHelp() string {
	help := `
  Key Bindings

  Tab / Shift+Tab   Switch tabs
  1-5               Jump to tab
  j/k               Navigate up/down
  Enter             Open detail view
  Esc               Close detail/help
  w                 Cycle workspaces
  r                 Refresh data
  ?                 Toggle help
  q                 Quit
`
	return views.HelpStyle.Render(help)
}

func (m Model) renderDetail() string {
	switch m.ActiveTab {
	case TabTasks:
		if m.SelectedTask < len(m.Tasks) {
			return views.RenderTaskDetail(m.Tasks[m.SelectedTask], m.Width, m.Height)
		}
	case TabGit:
		if m.GitStatus != nil && m.SelectedCommit < len(m.GitStatus.RecentCommits) {
			return views.RenderCommitDetail(m.GitStatus.RecentCommits[m.SelectedCommit], m.Width, m.Height)
		}
	}
	return ""
}
