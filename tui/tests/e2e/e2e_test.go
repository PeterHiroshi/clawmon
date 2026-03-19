// Package e2e tests the full TUI pipeline from client through app to view rendering.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PeterHiroshi/clawmon/tui/internal/app"
	"github.com/PeterHiroshi/clawmon/tui/internal/client"
	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// buildFullMockDaemon creates a realistic mock daemon with all endpoints + SSE.
func buildFullMockDaemon(t *testing.T) *httptest.Server {
	t.Helper()

	started := time.Date(2026, 3, 18, 17, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 3, 18, 18, 0, 0, 0, time.UTC)
	modelStr := "opus"
	teams := true
	dur := uint64(3600)
	exit := 0
	cpu := 25.0
	mem := uint64(102400)
	uptime := uint64(3600)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"status": "ok", "uptime_seconds": 7200, "version": "0.1.0",
		})
	})

	mux.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"id": "clawmon", "path": "/ws/clawmon", "name": "clawmon"},
		})
	})

	mux.HandleFunc("/workspaces/clawmon/git", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"branch":            "feature/phase2-tui",
			"is_clean":          false,
			"uncommitted_files": []string{"tui/internal/app/app.go"},
			"ahead":             5,
			"behind":            0,
			"last_commit": map[string]interface{}{
				"hash": "abc123", "short_hash": "abc123", "message": "feat(tui): add views",
				"author": "Dev", "time": completed.Format(time.RFC3339),
			},
			"last_push_time": started.Format(time.RFC3339),
			"recent_commits": []map[string]interface{}{
				{"hash": "abc123", "short_hash": "abc123", "message": "feat(tui): add views", "author": "Dev", "time": completed.Format(time.RFC3339)},
				{"hash": "def456", "short_hash": "def456", "message": "feat(tui): add client", "author": "Dev", "time": started.Format(time.RFC3339)},
			},
		})
	})

	mux.HandleFunc("/workspaces/clawmon/tasks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"name": "build-tui", "status": "done", "started_at": started.Format(time.RFC3339), "completed_at": completed.Format(time.RFC3339), "model": modelStr, "agent_teams": teams, "current_step": nil, "progress_percent": 100, "exit_code": exit, "effective_model": modelStr, "duration_seconds": dur, "task_dir": "/ws/.forge-task/build"},
			{"name": "fix-tests", "status": "in_progress", "started_at": completed.Format(time.RFC3339), "completed_at": nil, "model": nil, "agent_teams": nil, "current_step": "running tests", "progress_percent": 60, "exit_code": nil, "effective_model": nil, "duration_seconds": 300, "task_dir": "/ws/.forge-task/fix"},
			{"name": "deploy", "status": "pending", "started_at": nil, "completed_at": nil, "model": nil, "agent_teams": nil, "current_step": nil, "progress_percent": nil, "exit_code": nil, "effective_model": nil, "duration_seconds": nil, "task_dir": "/ws/.forge-task/deploy"},
		})
	})

	mux.HandleFunc("/workspaces/clawmon/processes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"pid": 9999, "workdir": "/ws/clawmon", "cpu_percent": cpu, "memory_kb": mem, "uptime_seconds": uptime, "state": "running", "stalled": false, "command": "claude-code"},
		})
	})

	mux.HandleFunc("/workspaces/clawmon/env", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"tools": []map[string]interface{}{
				{"tool": "git", "installed": true, "version": "2.43.0", "path": "/usr/bin/git"},
				{"tool": "go", "installed": true, "version": "1.24.1", "path": "/usr/local/go/bin/go"},
				{"tool": "cargo", "installed": true, "version": "1.82.0", "path": "/usr/bin/cargo"},
			},
			"hooks": []map[string]interface{}{
				{"name": "superpowers", "installed": true, "details": "v5.0"},
			},
		})
	})

	mux.HandleFunc("/workspaces/clawmon/activity", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"timestamp": completed.Format(time.RFC3339), "event_type": "task_completed", "workspace_id": "clawmon", "description": "Task completed: build-tui"},
			{"timestamp": started.Format(time.RFC3339), "event_type": "git_commit", "workspace_id": "clawmon", "description": "abc123: feat(tui): add views"},
			{"timestamp": started.Format(time.RFC3339), "event_type": "task_started", "workspace_id": "clawmon", "description": "Task started: build-tui"},
		})
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		fmt.Fprintf(w, "event:git_status\ndata:{\"event_type\":\"git_status\",\"workspace_id\":\"clawmon\",\"data\":{}}\n\n")
		flusher.Flush()
		<-r.Context().Done()
	})

	return httptest.NewServer(mux)
}

func writeJSON(t *testing.T, w http.ResponseWriter, data interface{}) {
	t.Helper()
	resp := map[string]interface{}{
		"data":      data,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(resp)
	require.NoError(t, err)
}

// loadAllData fetches all data from the mock daemon and loads it into the model.
func loadAllData(t *testing.T, m app.Model, c *client.HTTPClient) app.Model {
	t.Helper()

	health, err := c.Health()
	require.NoError(t, err)
	updated, _ := m.Update(app.HealthMsg{Response: health})
	m = updated.(app.Model)

	ws, err := c.ListWorkspaces()
	require.NoError(t, err)
	updated, _ = m.Update(app.WorkspacesMsg{Workspaces: ws})
	m = updated.(app.Model)

	wsID := m.Workspaces[0].ID

	git, err := c.GetGitStatus(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.GitMsg{Status: git})
	m = updated.(app.Model)

	tasks, err := c.GetTasks(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.TasksMsg{Tasks: tasks})
	m = updated.(app.Model)

	procs, err := c.GetProcesses(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.ProcessesMsg{Processes: procs})
	m = updated.(app.Model)

	env, err := c.GetEnvHealth(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.EnvMsg{Health: env})
	m = updated.(app.Model)

	activity, err := c.GetActivity(wsID)
	require.NoError(t, err)
	updated, _ = m.Update(app.ActivityMsg{Events: activity})
	m = updated.(app.Model)

	return m
}

func TestE2EDashboardRendering(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	m.ActiveTab = 0
	view := m.View()

	// Dashboard should show all key information
	assert.Contains(t, view, "clawmon")
	assert.Contains(t, view, "Tasks")
	assert.Contains(t, view, "Git")
	assert.Contains(t, view, "Processes")
	assert.Contains(t, view, "Environment")
	assert.Contains(t, view, "daemon")
}

func TestE2ETasksTabRendering(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	m.ActiveTab = 1
	view := m.View()

	assert.Contains(t, view, "build-tui")
	assert.Contains(t, view, "fix-tests")
	assert.Contains(t, view, "deploy")
}

func TestE2EGitTabRendering(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	m.ActiveTab = 2
	view := m.View()

	assert.Contains(t, view, "feature/phase2-tui")
	assert.Contains(t, view, "1 uncommitted")
	assert.Contains(t, view, "ahead 5")
	assert.Contains(t, view, "abc123")
}

func TestE2EActivityTabRendering(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	m.ActiveTab = 3
	view := m.View()

	assert.Contains(t, view, "Activity Timeline")
	assert.Contains(t, view, "build-tui")
}

func TestE2ETabSwitching(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	// Cycle through all tabs
	for i := 1; i <= 4; i++ {
		key := fmt.Sprintf("%d", i)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = updated.(app.Model)
		assert.Equal(t, i-1, m.ActiveTab)
		view := m.View()
		assert.NotEmpty(t, view)
	}
}

func TestE2ETaskDetailView(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	// Go to tasks tab, select second task, open detail
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(app.Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(app.Model)
	assert.Equal(t, 1, m.SelectedTask)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(app.Model)
	assert.True(t, m.ShowDetail)

	view := m.View()
	assert.Contains(t, view, "fix-tests")
	assert.Contains(t, view, "running tests")
}

func TestE2ECommitDetailView(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40
	m = loadAllData(t, m, c)

	// Go to git tab, open commit detail
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = updated.(app.Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(app.Model)
	assert.True(t, m.ShowDetail)

	view := m.View()
	assert.Contains(t, view, "abc123")
	assert.Contains(t, view, "feat(tui): add views")
}

func TestE2EDaemonUnreachable(t *testing.T) {
	c := client.NewHTTPClient("http://127.0.0.1:1")
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40

	// Simulate health failure
	_, err := c.Health()
	require.Error(t, err)

	updated, _ := m.Update(app.HealthMsg{Err: err})
	m = updated.(app.Model)
	assert.False(t, m.DaemonOnline)
	assert.NotNil(t, m.Errors["health"])
}

func TestE2ESSESubscription(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ch, err := c.SubscribeEvents(ctx)
	require.NoError(t, err)

	select {
	case event := <-ch:
		assert.Equal(t, "git_status", event.EventType)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}

func TestE2EFullWorkflow(t *testing.T) {
	server := buildFullMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	m := app.NewModel(c)
	m.Width = 120
	m.Height = 40

	// Simulate the full Init -> Update -> View lifecycle

	// 1. Health check
	health, err := c.Health()
	require.NoError(t, err)
	updated, _ := m.Update(app.HealthMsg{Response: health})
	m = updated.(app.Model)
	assert.True(t, m.DaemonOnline)

	// 2. Load workspaces
	ws, err := c.ListWorkspaces()
	require.NoError(t, err)
	updated, _ = m.Update(app.WorkspacesMsg{Workspaces: ws})
	m = updated.(app.Model)
	require.Len(t, m.Workspaces, 1)

	// 3. Fetch all workspace data
	m = loadAllData(t, m, c)

	// 4. View dashboard
	view := m.View()
	assert.NotEmpty(t, view)

	// 5. Switch to tasks, navigate, view detail
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(app.Model)
	view = m.View()
	assert.Contains(t, view, "build-tui")

	// 6. Open help
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(app.Model)
	view = m.View()
	assert.Contains(t, view, "Key Bindings")

	// 7. Close help
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(app.Model)
	assert.False(t, m.ShowHelp)

	// 8. SSE event triggers refresh
	wsID := "clawmon"
	sseEvent := models.SseEvent{EventType: "git_status", WorkspaceID: &wsID}
	updated, cmd := m.Update(app.SseMsg{Event: sseEvent})
	m = updated.(app.Model)
	assert.NotNil(t, cmd) // Should trigger git fetch
}
