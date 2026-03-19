// Package views provides TUI view renderers for the clawmon dashboard.
package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Color constants for the terminal dashboard.
var (
	ColorGreen  = lipgloss.Color("#00ff00")
	ColorYellow = lipgloss.Color("#ffff00")
	ColorRed    = lipgloss.Color("#ff0000")
	ColorCyan   = lipgloss.Color("#00ffff")
	ColorGray   = lipgloss.Color("#888888")
	ColorWhite  = lipgloss.Color("#ffffff")
	ColorDim    = lipgloss.Color("#555555")
)

// Tab names for the dashboard.
var TabNames = []string{"Dashboard", "Tasks", "Git", "Activity", "System"}

// Constants for timing and layout.
const (
	RefreshInterval  = 5 * time.Second
	DefaultDaemonURL = "http://127.0.0.1:9876/api/v1"
	MaxCardWidth     = 40
	MinCardWidth     = 30
)

// Styles for the TUI.
var (
	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(lipgloss.Color("#333333")).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorGray).
				Padding(0, 2)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(1, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	SelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(ColorWhite)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Padding(1, 2)
)

// StatusColor returns the appropriate style for a task/process status string.
func StatusColor(status string) lipgloss.Style {
	switch status {
	case "done", "running", "clean":
		return SuccessStyle
	case "in_progress", "sleeping":
		return WarningStyle
	case "failed", "dirty", "zombie", "stopped":
		return ErrorStyle
	case "pending", "unknown":
		return DimStyle
	default:
		return DimStyle
	}
}

// StatusIcon returns a status indicator icon.
func StatusIcon(status string) string {
	switch status {
	case "done":
		return SuccessStyle.Render("●")
	case "in_progress":
		return WarningStyle.Render("●")
	case "failed":
		return ErrorStyle.Render("●")
	case "pending":
		return DimStyle.Render("○")
	default:
		return DimStyle.Render("○")
	}
}

// FormatTimestamp formats a time for display (relative if recent, absolute otherwise).
func FormatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	elapsed := time.Since(t)
	switch {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	default:
		return t.Format("Jan 02 15:04")
	}
}

// FormatDuration formats a duration in seconds to a human-readable string.
func FormatDuration(seconds uint64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
}

// Truncate limits a string to max characters, adding ellipsis if truncated.
func Truncate(s string, max int) string {
	if max <= 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

// ConnectionDot returns a colored dot indicating daemon connection state.
func ConnectionDot(online bool) string {
	if online {
		return SuccessStyle.Render("●")
	}
	return ErrorStyle.Render("●")
}

// RenderTabBar renders the tab bar with the active tab highlighted.
func RenderTabBar(activeTab int, width int) string {
	var tabs []string
	for i, name := range TabNames {
		if i == activeTab {
			tabs = append(tabs, TabActiveStyle.Render(name))
		} else {
			tabs = append(tabs, TabInactiveStyle.Render(name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// BoolYN formats a bool pointer as Y/N/—.
func BoolYN(b *bool) string {
	if b == nil {
		return "—"
	}
	if *b {
		return "Y"
	}
	return "N"
}
