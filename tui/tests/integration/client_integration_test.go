package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/client"
	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRealisticMockDaemon creates an httptest server with all endpoints populated with realistic data.
func buildRealisticMockDaemon(t *testing.T) *httptest.Server {
	t.Helper()

	started := time.Date(2026, 3, 18, 17, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 3, 18, 18, 0, 0, 0, time.UTC)
	pushTime := time.Date(2026, 3, 18, 16, 0, 0, 0, time.UTC)
	modelStr := "opus"
	teams := true
	dur := uint64(3600)
	exit := 0
	cpu := 35.2
	mem := uint64(204800)
	uptime := uint64(7200)
	version := "2.43.0"
	gitPath := "/usr/bin/git"
	workdir := "/ws/alpha"

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"status":         "ok",
			"uptime_seconds": 3600,
			"version":        "0.1.0",
		})
	})

	mux.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{"id": "alpha", "path": "/ws/alpha", "name": "Alpha Project"},
			{"id": "beta", "path": "/ws/beta", "name": "Beta Project"},
		})
	})

	mux.HandleFunc("/workspaces/alpha/git", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"branch":            "develop",
			"is_clean":          false,
			"uncommitted_files": []string{"src/main.rs", "Cargo.toml"},
			"ahead":             3,
			"behind":            1,
			"last_commit": map[string]interface{}{
				"hash":       "abc123def456789",
				"short_hash": "abc123d",
				"message":    "feat(daemon): add git collector",
				"author":     "Test User",
				"time":       started.Format(time.RFC3339),
			},
			"last_push_time": pushTime.Format(time.RFC3339),
			"recent_commits": []map[string]interface{}{
				{
					"hash":       "abc123def456789",
					"short_hash": "abc123d",
					"message":    "feat(daemon): add git collector",
					"author":     "Test User",
					"time":       started.Format(time.RFC3339),
				},
				{
					"hash":       "def456abc789012",
					"short_hash": "def456a",
					"message":    "chore: scaffold project",
					"author":     "Test User",
					"time":       pushTime.Format(time.RFC3339),
				},
			},
		})
	})

	mux.HandleFunc("/workspaces/alpha/tasks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{
				"name":             "build-feature",
				"status":           "done",
				"started_at":       started.Format(time.RFC3339),
				"completed_at":     completed.Format(time.RFC3339),
				"model":            modelStr,
				"agent_teams":      teams,
				"current_step":     nil,
				"progress_percent": 100,
				"exit_code":        exit,
				"effective_model":  modelStr,
				"duration_seconds": dur,
				"task_dir":         "/ws/alpha/.forge-task/build-feature",
			},
			{
				"name":             "fix-bug",
				"status":           "in_progress",
				"started_at":       completed.Format(time.RFC3339),
				"completed_at":     nil,
				"model":            nil,
				"agent_teams":      nil,
				"current_step":     "debugging",
				"progress_percent": 30,
				"exit_code":        nil,
				"effective_model":  nil,
				"duration_seconds": 600,
				"task_dir":         "/ws/alpha/.forge-task/fix-bug",
			},
		})
	})

	mux.HandleFunc("/workspaces/alpha/processes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{
				"pid":            12345,
				"workdir":        workdir,
				"cpu_percent":    cpu,
				"memory_kb":      mem,
				"uptime_seconds": uptime,
				"state":          "running",
				"stalled":        false,
				"command":        "claude-code --workspace /ws/alpha",
			},
		})
	})

	mux.HandleFunc("/workspaces/alpha/env", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{
			"tools": []map[string]interface{}{
				{"tool": "git", "installed": true, "version": version, "path": gitPath},
				{"tool": "cargo", "installed": true, "version": "1.82.0", "path": "/usr/bin/cargo"},
				{"tool": "go", "installed": true, "version": "1.23.0", "path": "/usr/local/go/bin/go"},
				{"tool": "node", "installed": false, "version": nil, "path": nil},
			},
			"hooks": []map[string]interface{}{
				{"name": "notify-forge", "installed": true, "details": "v1.0"},
				{"name": "superpowers", "installed": false, "details": nil},
			},
		})
	})

	mux.HandleFunc("/workspaces/alpha/activity", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []map[string]interface{}{
			{
				"timestamp":    completed.Format(time.RFC3339),
				"event_type":   "task_completed",
				"workspace_id": "alpha",
				"description":  "Task completed: build-feature",
			},
			{
				"timestamp":    started.Format(time.RFC3339),
				"event_type":   "task_started",
				"workspace_id": "alpha",
				"description":  "Task started: build-feature",
			},
			{
				"timestamp":    started.Format(time.RFC3339),
				"event_type":   "git_commit",
				"workspace_id": "alpha",
				"description":  "abc123d: feat(daemon): add git collector",
			},
		})
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

func TestClientIntegrationHealth(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	health, err := c.Health()
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)
	assert.Equal(t, uint64(3600), health.UptimeSeconds)
}

func TestClientIntegrationWorkspaces(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	ws, err := c.ListWorkspaces()
	require.NoError(t, err)
	require.Len(t, ws, 2)
	assert.Equal(t, "alpha", ws[0].ID)
}

func TestClientIntegrationGit(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	git, err := c.GetGitStatus("alpha")
	require.NoError(t, err)
	assert.Equal(t, "develop", git.Branch)
	assert.False(t, git.IsClean)
	assert.Len(t, git.UncommittedFiles, 2)
	assert.Equal(t, 3, git.Ahead)
	assert.Equal(t, 1, git.Behind)
	require.NotNil(t, git.LastCommit)
	assert.Equal(t, "abc123d", git.LastCommit.ShortHash)
	require.Len(t, git.RecentCommits, 2)
	require.NotNil(t, git.LastPushTime)
}

func TestClientIntegrationTasks(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	tasks, err := c.GetTasks("alpha")
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	assert.Equal(t, "build-feature", tasks[0].Name)
	assert.Equal(t, models.TaskStatusDone, tasks[0].Status)
	require.NotNil(t, tasks[0].Model)
	assert.Equal(t, "opus", *tasks[0].Model)

	assert.Equal(t, "fix-bug", tasks[1].Name)
	assert.Equal(t, models.TaskStatusInProgress, tasks[1].Status)
}

func TestClientIntegrationProcesses(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	procs, err := c.GetProcesses("alpha")
	require.NoError(t, err)
	require.Len(t, procs, 1)
	assert.Equal(t, uint32(12345), procs[0].PID)
	assert.Equal(t, models.ProcessStateRunning, procs[0].State)
	assert.False(t, procs[0].Stalled)
}

func TestClientIntegrationEnv(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	env, err := c.GetEnvHealth("alpha")
	require.NoError(t, err)
	require.Len(t, env.Tools, 4)
	require.Len(t, env.Hooks, 2)

	// Count installed
	installed := 0
	for _, tool := range env.Tools {
		if tool.Installed {
			installed++
		}
	}
	assert.Equal(t, 3, installed)
}

func TestClientIntegrationActivity(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)
	events, err := c.GetActivity("alpha")
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, models.ActivityTaskCompleted, events[0].EventType)
}

func TestClientIntegrationAllEndpoints(t *testing.T) {
	server := buildRealisticMockDaemon(t)
	defer server.Close()

	c := client.NewHTTPClient(server.URL)

	// Test all endpoints in sequence — realistic usage pattern
	health, err := c.Health()
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)

	ws, err := c.ListWorkspaces()
	require.NoError(t, err)
	require.NotEmpty(t, ws)

	wsID := ws[0].ID
	git, err := c.GetGitStatus(wsID)
	require.NoError(t, err)
	assert.NotEmpty(t, git.Branch)

	tasks, err := c.GetTasks(wsID)
	require.NoError(t, err)
	assert.NotNil(t, tasks)

	procs, err := c.GetProcesses(wsID)
	require.NoError(t, err)
	assert.NotNil(t, procs)

	env, err := c.GetEnvHealth(wsID)
	require.NoError(t, err)
	assert.NotNil(t, env)

	activity, err := c.GetActivity(wsID)
	require.NoError(t, err)
	assert.NotNil(t, activity)
}
