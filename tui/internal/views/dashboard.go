package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// DashboardData aggregates all data needed for the dashboard overview.
type DashboardData struct {
	WorkspacePath  string
	WorkspaceName  string
	WorkspaceCount int
	WorkspaceIndex int
	DaemonOnline   bool
	Uptime         uint64
	GitStatus      *models.GitStatus
	Tasks          []models.TaskInfo
	Processes      []models.ProcessInfo
	EnvHealth      *models.EnvHealth
	Activity       []models.ActivityEvent
}

// RenderDashboard renders the dashboard overview tab.
func RenderDashboard(data DashboardData, width, height int) string {
	var sections []string

	// Top row: workspace path + connection status
	header := renderDashboardHeader(data, width)
	sections = append(sections, header)

	// Cards (2x2 grid)
	cardWidth := (width - 6) / 2
	if cardWidth < MinCardWidth {
		cardWidth = MinCardWidth
	}
	if cardWidth > MaxCardWidth {
		cardWidth = MaxCardWidth
	}

	tasksCard := renderTasksCard(data.Tasks, cardWidth)
	gitCard := renderGitCard(data.GitStatus, cardWidth)
	processCard := renderProcessCard(data.Processes, cardWidth)
	envCard := renderEnvCard(data.EnvHealth, cardWidth)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, tasksCard, "  ", gitCard)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, processCard, "  ", envCard)
	sections = append(sections, topRow, bottomRow)

	// Quick activity feed
	activityFeed := renderQuickActivity(data.Activity)
	sections = append(sections, activityFeed)

	return strings.Join(sections, "\n\n")
}

func renderDashboardHeader(data DashboardData, width int) string {
	dot := ConnectionDot(data.DaemonOnline)
	status := "connected"
	if !data.DaemonOnline {
		status = "disconnected"
	}
	uptimeStr := FormatDuration(data.Uptime)

	wsLabel := data.WorkspacePath
	if data.WorkspaceCount > 1 {
		wsLabel = fmt.Sprintf("[%d/%d] %s", data.WorkspaceIndex+1, data.WorkspaceCount, data.WorkspaceName)
	}
	left := TitleStyle.Render("clawmon") + "  " + DimStyle.Render(wsLabel)
	right := fmt.Sprintf("%s %s  uptime: %s", dot, status, uptimeStr)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func renderTasksCard(tasks []models.TaskInfo, width int) string {
	var running, completed, failed, total int
	total = len(tasks)
	for _, t := range tasks {
		switch t.Status {
		case models.TaskStatusInProgress:
			running++
		case models.TaskStatusDone:
			completed++
		case models.TaskStatusFailed:
			failed++
		}
	}

	content := TitleStyle.Render("Tasks") + "\n\n"
	content += fmt.Sprintf("Total:     %d\n", total)
	content += WarningStyle.Render(fmt.Sprintf("Running:   %d", running)) + "\n"
	content += SuccessStyle.Render(fmt.Sprintf("Completed: %d", completed)) + "\n"
	content += ErrorStyle.Render(fmt.Sprintf("Failed:    %d", failed))

	return CardStyle.Width(width).Render(content)
}

func renderGitCard(git *models.GitStatus, width int) string {
	content := TitleStyle.Render("Git") + "\n\n"

	if git == nil {
		content += DimStyle.Render("No data")
		return CardStyle.Width(width).Render(content)
	}

	content += fmt.Sprintf("Branch: %s\n", git.Branch)

	if git.IsClean {
		content += SuccessStyle.Render("Clean") + "\n"
	} else {
		content += ErrorStyle.Render(fmt.Sprintf("Dirty (%d files)", len(git.UncommittedFiles))) + "\n"
	}

	if git.Ahead > 0 || git.Behind > 0 {
		content += WarningStyle.Render(fmt.Sprintf("ahead %d  behind %d", git.Ahead, git.Behind)) + "\n"
	}

	if git.LastCommit != nil {
		content += DimStyle.Render(Truncate(git.LastCommit.Message, width-8))
	}

	return CardStyle.Width(width).Render(content)
}

func renderProcessCard(procs []models.ProcessInfo, width int) string {
	content := TitleStyle.Render("Processes") + "\n\n"

	if len(procs) == 0 {
		content += DimStyle.Render("No CC processes")
		return CardStyle.Width(width).Render(content)
	}

	var totalCPU float64
	var totalMem uint64
	var stalledCount int

	for _, p := range procs {
		if p.CPUPercent != nil {
			totalCPU += *p.CPUPercent
		}
		if p.MemoryKB != nil {
			totalMem += *p.MemoryKB
		}
		if p.Stalled {
			stalledCount++
		}
	}

	content += fmt.Sprintf("Running: %d\n", len(procs))
	content += fmt.Sprintf("CPU:     %.1f%%\n", totalCPU)
	content += fmt.Sprintf("Memory:  %d MB\n", totalMem/1024)

	if stalledCount > 0 {
		content += WarningStyle.Render(fmt.Sprintf("Stalled: %d", stalledCount))
	}

	return CardStyle.Width(width).Render(content)
}

func renderEnvCard(env *models.EnvHealth, width int) string {
	content := TitleStyle.Render("Environment") + "\n\n"

	if env == nil {
		content += DimStyle.Render("No data")
		return CardStyle.Width(width).Render(content)
	}

	var installed, total int
	total = len(env.Tools)
	var missing []string
	for _, tool := range env.Tools {
		if tool.Installed {
			installed++
		} else {
			missing = append(missing, tool.Tool)
		}
	}

	content += fmt.Sprintf("Tools: %d/%d\n", installed, total)

	if len(missing) > 0 {
		content += ErrorStyle.Render("Missing: " + strings.Join(missing, ", "))
	} else {
		content += SuccessStyle.Render("All tools available")
	}

	return CardStyle.Width(width).Render(content)
}

func renderQuickActivity(events []models.ActivityEvent) string {
	if len(events) == 0 {
		return DimStyle.Render("No recent activity")
	}

	title := TitleStyle.Render("Recent Activity")
	var lines []string

	limit := 5
	if len(events) < limit {
		limit = len(events)
	}

	for _, event := range events[:limit] {
		icon := activityIcon(string(event.EventType))
		ts := FormatTimestamp(event.Timestamp)
		line := fmt.Sprintf("  %s %s  %s", DimStyle.Render(ts), icon, event.Description)
		lines = append(lines, line)
	}

	return title + "\n" + strings.Join(lines, "\n")
}
