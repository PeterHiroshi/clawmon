package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrapResponse wraps data in the standard ApiResponse envelope.
func wrapResponse(t *testing.T, data interface{}) []byte {
	t.Helper()
	resp := map[string]interface{}{
		"data":      data,
		"timestamp": "2026-03-18T19:00:00Z",
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	return b
}

func newMockServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}
	return httptest.NewServer(mux)
}

func TestHealth(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/health": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, map[string]interface{}{
				"status":         "ok",
				"uptime_seconds": 42,
				"version":        "0.1.0",
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	health, err := client.Health()
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)
	assert.Equal(t, uint64(42), health.UptimeSeconds)
	assert.Equal(t, "0.1.0", health.Version)
}

func TestListWorkspaces(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, []map[string]interface{}{
				{"id": "alpha", "path": "/ws/alpha", "name": "Alpha"},
				{"id": "beta", "path": "/ws/beta", "name": "Beta"},
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	workspaces, err := client.ListWorkspaces()
	require.NoError(t, err)
	require.Len(t, workspaces, 2)
	assert.Equal(t, "alpha", workspaces[0].ID)
	assert.Equal(t, "beta", workspaces[1].ID)
}

func TestGetGitStatus(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/git": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, map[string]interface{}{
				"branch":            "main",
				"is_clean":          true,
				"uncommitted_files": []string{},
				"ahead":             0,
				"behind":            0,
				"last_commit":       nil,
				"last_push_time":    nil,
				"recent_commits":    []interface{}{},
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	git, err := client.GetGitStatus("alpha")
	require.NoError(t, err)
	assert.Equal(t, "main", git.Branch)
	assert.True(t, git.IsClean)
}

func TestGetTasks(t *testing.T) {
	started := time.Date(2026, 3, 18, 18, 0, 0, 0, time.UTC)
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/tasks": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, []map[string]interface{}{
				{
					"name":             "test-task",
					"status":           "in_progress",
					"started_at":       started.Format(time.RFC3339),
					"completed_at":     nil,
					"model":            "opus",
					"agent_teams":      true,
					"current_step":     "building",
					"progress_percent": 50,
					"exit_code":        nil,
					"effective_model":  "opus",
					"duration_seconds": 120,
					"task_dir":         "/tmp/task",
				},
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	tasks, err := client.GetTasks("alpha")
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "test-task", tasks[0].Name)
	assert.Equal(t, models.TaskStatusInProgress, tasks[0].Status)
}

func TestGetProcesses(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/processes": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			cpu := 25.5
			mem := uint64(102400)
			uptime := uint64(3600)
			w.Write(wrapResponse(t, []map[string]interface{}{
				{
					"pid":            12345,
					"workdir":        "/tmp/work",
					"cpu_percent":    cpu,
					"memory_kb":      mem,
					"uptime_seconds": uptime,
					"state":          "running",
					"stalled":        false,
					"command":        "claude-code",
				},
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	procs, err := client.GetProcesses("alpha")
	require.NoError(t, err)
	require.Len(t, procs, 1)
	assert.Equal(t, uint32(12345), procs[0].PID)
	assert.Equal(t, models.ProcessStateRunning, procs[0].State)
}

func TestGetEnvHealth(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/env": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, map[string]interface{}{
				"tools": []map[string]interface{}{
					{"tool": "git", "installed": true, "version": "2.43.0", "path": "/usr/bin/git"},
				},
				"hooks": []map[string]interface{}{
					{"name": "notify-forge", "installed": false, "details": nil},
				},
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	env, err := client.GetEnvHealth("alpha")
	require.NoError(t, err)
	require.Len(t, env.Tools, 1)
	assert.True(t, env.Tools[0].Installed)
	require.Len(t, env.Hooks, 1)
	assert.False(t, env.Hooks[0].Installed)
}

func TestGetActivity(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/activity": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, []map[string]interface{}{
				{
					"timestamp":    "2026-03-18T19:00:00Z",
					"event_type":   "git_commit",
					"workspace_id": "alpha",
					"description":  "abc123: feat: something",
				},
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	events, err := client.GetActivity("alpha")
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, models.ActivityGitCommit, events[0].EventType)
}

func TestGetSystemResources(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/system": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(wrapResponse(t, map[string]interface{}{
				"cpu": map[string]interface{}{
					"model":          "Test CPU",
					"core_count":     4,
					"usage_percent":  50.0,
					"per_core_usage": []float32{45.0, 55.0, 48.0, 52.0},
					"load_avg_1m":    1.5,
					"load_avg_5m":    1.2,
					"load_avg_15m":   0.9,
				},
				"memory": map[string]interface{}{
					"total_bytes":     16000000000,
					"used_bytes":      8000000000,
					"available_bytes": 8000000000,
					"usage_percent":   50.0,
				},
				"swap": map[string]interface{}{
					"total_bytes":   4000000000,
					"used_bytes":    100000000,
					"usage_percent": 2.5,
				},
				"disks": []map[string]interface{}{
					{
						"mount_point":     "/",
						"filesystem":      "ext4",
						"total_bytes":     500000000000,
						"used_bytes":      250000000000,
						"available_bytes": 250000000000,
						"usage_percent":   50.0,
					},
				},
				"gpus":           []interface{}{},
				"uptime_seconds": 86400,
				"collected_at":   "2026-03-18T19:00:00Z",
			}))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	sys, err := client.GetSystemResources("alpha")
	require.NoError(t, err)
	assert.Equal(t, "Test CPU", sys.CPU.Model)
	assert.Equal(t, 4, sys.CPU.CoreCount)
	assert.InDelta(t, 50.0, float64(sys.CPU.UsagePercent), 0.1)
	assert.Equal(t, uint64(16000000000), sys.Memory.TotalBytes)
	require.Len(t, sys.Disks, 1)
	assert.Equal(t, "/", sys.Disks[0].MountPoint)
	assert.Empty(t, sys.GPUs)
	assert.Equal(t, uint64(86400), sys.UptimeSeconds)
}

func TestNotFoundError(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/missing/git": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"workspace not found: missing"}`))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	_, err := client.GetGitStatus("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestServerError(t *testing.T) {
	server := newMockServer(t, map[string]http.HandlerFunc{
		"/workspaces/alpha/git": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"git collector failed"}`))
		},
	})
	defer server.Close()

	client := NewHTTPClient(server.URL)
	_, err := client.GetGitStatus("alpha")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daemon error")
}

func TestDaemonUnreachable(t *testing.T) {
	client := NewHTTPClient("http://127.0.0.1:1") // unreachable port
	_, err := client.Health()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "daemon unreachable")
}
