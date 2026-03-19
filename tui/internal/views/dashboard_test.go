package views

import (
	"testing"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderDashboard(t *testing.T) {
	data := DashboardData{
		WorkspacePath: "/ws/alpha",
		DaemonOnline:  true,
		Uptime:        3600,
		GitStatus: &models.GitStatus{
			Branch:  "main",
			IsClean: true,
			LastCommit: &models.CommitInfo{
				ShortHash: "abc123",
				Message:   "feat: initial",
				Author:    "Test",
				Time:      time.Now(),
			},
		},
		Tasks: []models.TaskInfo{
			{Name: "task1", Status: models.TaskStatusDone, TaskDir: "/tmp"},
			{Name: "task2", Status: models.TaskStatusInProgress, TaskDir: "/tmp"},
			{Name: "task3", Status: models.TaskStatusFailed, TaskDir: "/tmp"},
		},
		Processes: []models.ProcessInfo{},
		EnvHealth: &models.EnvHealth{
			Tools: []models.EnvCheck{
				{Tool: "git", Installed: true},
				{Tool: "cargo", Installed: false},
			},
		},
		Activity: []models.ActivityEvent{
			{Timestamp: time.Now(), EventType: models.ActivityGitCommit, Description: "test commit"},
		},
	}

	result := RenderDashboard(data, 80, 24)
	assert.Contains(t, result, "clawmon")
	assert.Contains(t, result, "Tasks")
	assert.Contains(t, result, "Git")
	assert.Contains(t, result, "Processes")
	assert.Contains(t, result, "Environment")
}

func TestRenderDashboardEmptyData(t *testing.T) {
	data := DashboardData{
		DaemonOnline: false,
	}
	result := RenderDashboard(data, 80, 24)
	assert.Contains(t, result, "No data")
	assert.Contains(t, result, "No CC processes")
}

func TestRenderDashboardWithProcesses(t *testing.T) {
	cpu := 50.0
	mem := uint64(204800)
	data := DashboardData{
		DaemonOnline: true,
		Processes: []models.ProcessInfo{
			{PID: 1, State: models.ProcessStateRunning, CPUPercent: &cpu, MemoryKB: &mem, Command: "cc", Stalled: true},
		},
	}
	result := RenderDashboard(data, 80, 24)
	assert.Contains(t, result, "Running: 1")
	assert.Contains(t, result, "Stalled")
}
