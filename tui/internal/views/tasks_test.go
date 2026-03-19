package views

import (
	"testing"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderTasksEmpty(t *testing.T) {
	result := RenderTasks(nil, 0, 80, 24)
	assert.Contains(t, result, "No tasks found")
}

func TestRenderTasksWithData(t *testing.T) {
	started := time.Now().Add(-1 * time.Hour)
	dur := uint64(3600)
	model := "opus"
	teams := true
	tasks := []models.TaskInfo{
		{
			Name:            "build-feature",
			Status:          models.TaskStatusDone,
			StartedAt:       &started,
			DurationSeconds: &dur,
			Model:           &model,
			AgentTeams:      &teams,
			TaskDir:         "/tmp/task1",
		},
		{
			Name:    "fix-bug",
			Status:  models.TaskStatusInProgress,
			TaskDir: "/tmp/task2",
		},
	}

	result := RenderTasks(tasks, 0, 100, 24)
	assert.Contains(t, result, "Name")
	assert.Contains(t, result, "Status")
	assert.Contains(t, result, "build-feature")
	assert.Contains(t, result, "fix-bug")
}

func TestRenderTaskDetail(t *testing.T) {
	started := time.Now()
	dur := uint64(120)
	model := "opus"
	step := "building"
	progress := uint8(50)
	exit := 0
	teams := true

	task := models.TaskInfo{
		Name:            "test-task",
		Status:          models.TaskStatusDone,
		StartedAt:       &started,
		CompletedAt:     &started,
		DurationSeconds: &dur,
		Model:           &model,
		EffectiveModel:  &model,
		AgentTeams:      &teams,
		CurrentStep:     &step,
		ProgressPercent: &progress,
		ExitCode:        &exit,
		TaskDir:         "/tmp/task",
	}

	result := RenderTaskDetail(task, 80, 24)
	assert.Contains(t, result, "test-task")
	assert.Contains(t, result, "opus")
	assert.Contains(t, result, "building")
	assert.Contains(t, result, "50%")
	assert.Contains(t, result, "Esc")
}
