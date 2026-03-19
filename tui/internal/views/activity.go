package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// activityIcon returns a colored icon for an event type.
func activityIcon(eventType string) string {
	switch models.ActivityType(eventType) {
	case models.ActivityGitCommit, models.ActivityGitPush:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#5555ff")).Render("commit")
	case models.ActivityTaskStarted:
		return WarningStyle.Render("task")
	case models.ActivityTaskCompleted:
		return SuccessStyle.Render("task")
	case models.ActivityTaskFailed:
		return ErrorStyle.Render("task")
	case models.ActivityProcessStarted, models.ActivityProcessStopped:
		return lipgloss.NewStyle().Foreground(ColorCyan).Render("proc")
	case models.ActivityFileChanged:
		return DimStyle.Render("file")
	default:
		return DimStyle.Render("event")
	}
}

// RenderActivity renders the activity timeline tab.
func RenderActivity(events []models.ActivityEvent, selected int, width, height int) string {
	if len(events) == 0 {
		return lipgloss.Place(width, height/2, lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No activity events"))
	}

	title := TitleStyle.Render("Activity Timeline")
	var rows []string

	for i, event := range events {
		ts := DimStyle.Render(FormatTimestamp(event.Timestamp))
		icon := activityIcon(string(event.EventType))
		desc := event.Description

		row := fmt.Sprintf("  %s  %s  %s", ts, icon, desc)
		if i == selected {
			row = SelectedStyle.Render(row)
		}
		rows = append(rows, row)
	}

	return title + "\n\n" + strings.Join(rows, "\n")
}
