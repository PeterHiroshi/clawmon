package views

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"done"},
		{"in_progress"},
		{"failed"},
		{"pending"},
		{"unknown"},
		{"running"},
		{"sleeping"},
		{"dirty"},
		{"zombie"},
		{"stopped"},
		{"clean"},
		{"other"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			style := StatusColor(tt.status)
			// Should not panic and should return a valid style
			result := style.Render("test")
			assert.NotEmpty(t, result)
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"done", "●"},
		{"in_progress", "●"},
		{"failed", "●"},
		{"pending", "○"},
		{"unknown", "○"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := StatusIcon(tt.status)
			assert.Contains(t, icon, tt.contains)
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"zero", time.Time{}, "—"},
		{"just now", time.Now().Add(-10 * time.Second), "just now"},
		{"minutes ago", time.Now().Add(-5 * time.Minute), "5m ago"},
		{"hours ago", time.Now().Add(-3 * time.Hour), "3h ago"},
		{"days ago", time.Now().Add(-48 * time.Hour), time.Now().Add(-48 * time.Hour).Format("Jan 02 15:04")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimestamp(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  uint64
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{60, "1m 0s"},
		{90, "1m 30s"},
		{3600, "1h 0m"},
		{3661, "1h 1m"},
		{7200, "2h 0m"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.seconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 5, "hello"},
		{"abcdef", 6, "abcdef"},
		{"abcdefg", 6, "abc..."},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Truncate(tt.input, tt.max)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectionDot(t *testing.T) {
	online := ConnectionDot(true)
	assert.Contains(t, online, "●")
	offline := ConnectionDot(false)
	assert.Contains(t, offline, "●")
}

func TestRenderTabBar(t *testing.T) {
	result := RenderTabBar(0, 80)
	assert.Contains(t, result, "Dashboard")
	assert.Contains(t, result, "Tasks")
	assert.Contains(t, result, "Git")
	assert.Contains(t, result, "Activity")
}

func TestBoolYN(t *testing.T) {
	trueVal := true
	falseVal := false
	assert.Equal(t, "Y", BoolYN(&trueVal))
	assert.Equal(t, "N", BoolYN(&falseVal))
	assert.Equal(t, "—", BoolYN(nil))
}
