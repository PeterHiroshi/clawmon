package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthResponseDeserialization(t *testing.T) {
	raw := `{"data":{"status":"ok","uptime_seconds":42,"version":"0.1.0"},"timestamp":"2026-03-18T19:00:00Z"}`
	var resp ApiResponse[HealthResponse]
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Data.Status)
	assert.Equal(t, uint64(42), resp.Data.UptimeSeconds)
	assert.Equal(t, "0.1.0", resp.Data.Version)
	assert.Equal(t, "2026-03-18T19:00:00Z", resp.Timestamp)
}

func TestWorkspaceInfoDeserialization(t *testing.T) {
	raw := `{"data":[{"id":"my-project","path":"/tmp/workspace","name":"My Project"}],"timestamp":"2026-03-18T19:00:00Z"}`
	var resp ApiResponse[[]WorkspaceInfo]
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "my-project", resp.Data[0].ID)
	assert.Equal(t, "/tmp/workspace", resp.Data[0].Path)
	assert.Equal(t, "My Project", resp.Data[0].Name)
}

func TestGitStatusDeserialization(t *testing.T) {
	raw := `{
		"data": {
			"branch": "main",
			"is_clean": true,
			"uncommitted_files": [],
			"ahead": 0,
			"behind": 0,
			"last_commit": null,
			"last_push_time": null,
			"recent_commits": []
		},
		"timestamp": "2026-03-18T19:00:00Z"
	}`
	var resp ApiResponse[GitStatus]
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Equal(t, "main", resp.Data.Branch)
	assert.True(t, resp.Data.IsClean)
	assert.Empty(t, resp.Data.UncommittedFiles)
	assert.Nil(t, resp.Data.LastCommit)
}

func TestGitStatusWithCommits(t *testing.T) {
	raw := `{
		"data": {
			"branch": "feature/test",
			"is_clean": false,
			"uncommitted_files": ["src/main.rs", "Cargo.toml"],
			"ahead": 2,
			"behind": 1,
			"last_commit": {
				"hash": "abc123def456",
				"short_hash": "abc123d",
				"message": "feat: add something",
				"author": "Test User",
				"time": "2026-03-18T18:00:00Z"
			},
			"last_push_time": "2026-03-18T17:00:00Z",
			"recent_commits": [
				{
					"hash": "abc123def456",
					"short_hash": "abc123d",
					"message": "feat: add something",
					"author": "Test User",
					"time": "2026-03-18T18:00:00Z"
				}
			]
		},
		"timestamp": "2026-03-18T19:00:00Z"
	}`
	var resp ApiResponse[GitStatus]
	err := json.Unmarshal([]byte(raw), &resp)
	require.NoError(t, err)
	assert.Equal(t, "feature/test", resp.Data.Branch)
	assert.False(t, resp.Data.IsClean)
	assert.Len(t, resp.Data.UncommittedFiles, 2)
	assert.Equal(t, 2, resp.Data.Ahead)
	assert.Equal(t, 1, resp.Data.Behind)
	require.NotNil(t, resp.Data.LastCommit)
	assert.Equal(t, "abc123d", resp.Data.LastCommit.ShortHash)
	require.NotNil(t, resp.Data.LastPushTime)
	require.Len(t, resp.Data.RecentCommits, 1)
}

func TestTaskInfoDeserialization(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		status TaskStatus
	}{
		{
			name:   "pending task",
			json:   `{"name":"test-task","status":"pending","started_at":null,"completed_at":null,"model":null,"agent_teams":null,"current_step":null,"progress_percent":null,"exit_code":null,"effective_model":null,"duration_seconds":null,"task_dir":"/tmp/task"}`,
			status: TaskStatusPending,
		},
		{
			name:   "in_progress task",
			json:   `{"name":"running-task","status":"in_progress","started_at":"2026-03-18T18:00:00Z","completed_at":null,"model":"opus","agent_teams":true,"current_step":"building","progress_percent":50,"exit_code":null,"effective_model":"opus","duration_seconds":120,"task_dir":"/tmp/task2"}`,
			status: TaskStatusInProgress,
		},
		{
			name:   "done task",
			json:   `{"name":"done-task","status":"done","started_at":"2026-03-18T17:00:00Z","completed_at":"2026-03-18T18:00:00Z","model":"sonnet","agent_teams":false,"current_step":null,"progress_percent":100,"exit_code":0,"effective_model":"sonnet","duration_seconds":3600,"task_dir":"/tmp/task3"}`,
			status: TaskStatusDone,
		},
		{
			name:   "failed task",
			json:   `{"name":"fail-task","status":"failed","started_at":"2026-03-18T17:00:00Z","completed_at":"2026-03-18T17:30:00Z","model":null,"agent_teams":null,"current_step":null,"progress_percent":null,"exit_code":1,"effective_model":null,"duration_seconds":1800,"task_dir":"/tmp/task4"}`,
			status: TaskStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var task TaskInfo
			err := json.Unmarshal([]byte(tt.json), &task)
			require.NoError(t, err)
			assert.Equal(t, tt.status, task.Status)
		})
	}
}

func TestProcessInfoDeserialization(t *testing.T) {
	raw := `{
		"pid": 12345,
		"workdir": "/tmp/work",
		"cpu_percent": 25.5,
		"memory_kb": 102400,
		"uptime_seconds": 3600,
		"state": "running",
		"stalled": false,
		"command": "claude-code"
	}`
	var proc ProcessInfo
	err := json.Unmarshal([]byte(raw), &proc)
	require.NoError(t, err)
	assert.Equal(t, uint32(12345), proc.PID)
	assert.Equal(t, ProcessStateRunning, proc.State)
	assert.False(t, proc.Stalled)
	require.NotNil(t, proc.CPUPercent)
	assert.InDelta(t, 25.5, *proc.CPUPercent, 0.01)
}

func TestEnvHealthDeserialization(t *testing.T) {
	raw := `{
		"tools": [
			{"tool": "git", "installed": true, "version": "2.43.0", "path": "/usr/bin/git"},
			{"tool": "cargo", "installed": false, "version": null, "path": null}
		],
		"hooks": [
			{"name": "notify-forge", "installed": false, "details": null}
		]
	}`
	var health EnvHealth
	err := json.Unmarshal([]byte(raw), &health)
	require.NoError(t, err)
	require.Len(t, health.Tools, 2)
	assert.True(t, health.Tools[0].Installed)
	assert.False(t, health.Tools[1].Installed)
	require.Len(t, health.Hooks, 1)
	assert.False(t, health.Hooks[0].Installed)
}

func TestActivityEventDeserialization(t *testing.T) {
	raw := `{
		"timestamp": "2026-03-18T19:00:00Z",
		"event_type": "git_commit",
		"workspace_id": "my-project",
		"description": "abc123d: feat: add something"
	}`
	var event ActivityEvent
	err := json.Unmarshal([]byte(raw), &event)
	require.NoError(t, err)
	assert.Equal(t, ActivityGitCommit, event.EventType)
	assert.Equal(t, "my-project", event.WorkspaceID)
	expected := time.Date(2026, 3, 18, 19, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, event.Timestamp)
}

func TestSseEventDeserialization(t *testing.T) {
	raw := `{
		"event_type": "git_status",
		"workspace_id": "my-project",
		"data": {"branch": "main"}
	}`
	var event SseEvent
	err := json.Unmarshal([]byte(raw), &event)
	require.NoError(t, err)
	assert.Equal(t, "git_status", event.EventType)
	require.NotNil(t, event.WorkspaceID)
	assert.Equal(t, "my-project", *event.WorkspaceID)
}
