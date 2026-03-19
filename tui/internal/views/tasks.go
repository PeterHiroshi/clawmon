package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// Column widths for the tasks table.
const (
	colNameWidth     = 20
	colStatusWidth   = 14
	colDurationWidth = 10
	colModelWidth    = 10
	colTeamsWidth    = 6
	colStartedWidth  = 14
)

// RenderTasks renders the tasks tab with a scrollable table.
func RenderTasks(tasks []models.TaskInfo, selected int, width, height int) string {
	if len(tasks) == 0 {
		return lipgloss.Place(width, height/2, lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No tasks found"))
	}

	// Header
	header := renderTaskRow("Name", "Status", "Duration", "Model", "Teams", "Started", true)
	separator := DimStyle.Render(strings.Repeat("─", width))

	var rows []string
	rows = append(rows, header, separator)

	for i, task := range tasks {
		row := renderTaskTableRow(task, i == selected)
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

func renderTaskRow(name, status, duration, model, teams, started string, isHeader bool) string {
	style := DimStyle
	if isHeader {
		style = TitleStyle
	}
	return fmt.Sprintf("  %s  %s  %s  %s  %s  %s",
		style.Width(colNameWidth).Render(name),
		style.Width(colStatusWidth).Render(status),
		style.Width(colDurationWidth).Render(duration),
		style.Width(colModelWidth).Render(model),
		style.Width(colTeamsWidth).Render(teams),
		style.Width(colStartedWidth).Render(started),
	)
}

func renderTaskTableRow(task models.TaskInfo, isSelected bool) string {
	name := Truncate(task.Name, colNameWidth)
	status := string(task.Status)
	statusRendered := StatusColor(status).Width(colStatusWidth).Render(StatusIcon(status) + " " + status)

	duration := "—"
	if task.DurationSeconds != nil {
		duration = FormatDuration(*task.DurationSeconds)
	}

	model := "—"
	if task.EffectiveModel != nil {
		model = *task.EffectiveModel
	} else if task.Model != nil {
		model = *task.Model
	}

	teams := BoolYN(task.AgentTeams)
	started := "—"
	if task.StartedAt != nil {
		started = FormatTimestamp(*task.StartedAt)
	}

	row := fmt.Sprintf("  %s  %s  %s  %s  %s  %s",
		lipgloss.NewStyle().Width(colNameWidth).Render(name),
		statusRendered,
		lipgloss.NewStyle().Width(colDurationWidth).Render(duration),
		lipgloss.NewStyle().Width(colModelWidth).Render(model),
		lipgloss.NewStyle().Width(colTeamsWidth).Render(teams),
		DimStyle.Width(colStartedWidth).Render(started),
	)

	if isSelected {
		return SelectedStyle.Render(row)
	}
	return row
}

// RenderTaskDetail renders a detailed view for a single task.
func RenderTaskDetail(task models.TaskInfo, width, height int) string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Task: "+task.Name))
	lines = append(lines, "")

	statusStr := string(task.Status)
	lines = append(lines, fmt.Sprintf("  Status:   %s %s",
		StatusIcon(statusStr), StatusColor(statusStr).Render(statusStr)))

	if task.Model != nil {
		lines = append(lines, fmt.Sprintf("  Model:    %s", *task.Model))
	}
	if task.EffectiveModel != nil {
		lines = append(lines, fmt.Sprintf("  Effective: %s", *task.EffectiveModel))
	}

	lines = append(lines, fmt.Sprintf("  Teams:    %s", BoolYN(task.AgentTeams)))

	if task.StartedAt != nil {
		lines = append(lines, fmt.Sprintf("  Started:  %s", task.StartedAt.Format("2006-01-02 15:04:05")))
	}
	if task.CompletedAt != nil {
		lines = append(lines, fmt.Sprintf("  Ended:    %s", task.CompletedAt.Format("2006-01-02 15:04:05")))
	}
	if task.DurationSeconds != nil {
		lines = append(lines, fmt.Sprintf("  Duration: %s", FormatDuration(*task.DurationSeconds)))
	}
	if task.CurrentStep != nil {
		lines = append(lines, fmt.Sprintf("  Step:     %s", *task.CurrentStep))
	}
	if task.ProgressPercent != nil {
		lines = append(lines, fmt.Sprintf("  Progress: %d%%", *task.ProgressPercent))
	}
	if task.ExitCode != nil {
		lines = append(lines, fmt.Sprintf("  Exit:     %d", *task.ExitCode))
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Dir: "+task.TaskDir))
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Press Esc to close"))

	return CardStyle.Width(width - 4).Render(strings.Join(lines, "\n"))
}
