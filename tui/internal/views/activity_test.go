package views

import (
	"testing"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderActivityEmpty(t *testing.T) {
	result := RenderActivity(nil, 0, 80, 24)
	assert.Contains(t, result, "No activity events")
}

func TestRenderActivityWithEvents(t *testing.T) {
	events := []models.ActivityEvent{
		{Timestamp: time.Now(), EventType: models.ActivityGitCommit, Description: "abc: feat: add feature"},
		{Timestamp: time.Now(), EventType: models.ActivityTaskStarted, Description: "Task started: build"},
		{Timestamp: time.Now(), EventType: models.ActivityTaskCompleted, Description: "Task completed: build"},
		{Timestamp: time.Now(), EventType: models.ActivityTaskFailed, Description: "Task failed: test"},
		{Timestamp: time.Now(), EventType: models.ActivityProcessStarted, Description: "CC process started"},
	}

	result := RenderActivity(events, 0, 80, 24)
	assert.Contains(t, result, "Activity Timeline")
	assert.Contains(t, result, "feat: add feature")
	assert.Contains(t, result, "Task started: build")
}

func TestRenderActivitySelectedRow(t *testing.T) {
	events := []models.ActivityEvent{
		{Timestamp: time.Now(), EventType: models.ActivityGitCommit, Description: "first"},
		{Timestamp: time.Now(), EventType: models.ActivityGitPush, Description: "second"},
	}

	// Should not panic with selected=1
	result := RenderActivity(events, 1, 80, 24)
	assert.Contains(t, result, "second")
}

func TestActivityIcon(t *testing.T) {
	tests := []struct {
		eventType string
	}{
		{string(models.ActivityGitCommit)},
		{string(models.ActivityGitPush)},
		{string(models.ActivityTaskStarted)},
		{string(models.ActivityTaskCompleted)},
		{string(models.ActivityTaskFailed)},
		{string(models.ActivityProcessStarted)},
		{string(models.ActivityProcessStopped)},
		{string(models.ActivityFileChanged)},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			icon := activityIcon(tt.eventType)
			assert.NotEmpty(t, icon)
		})
	}
}
