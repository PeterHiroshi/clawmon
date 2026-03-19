package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// RenderGit renders the git sync tab.
func RenderGit(git *models.GitStatus, selected int, width, height int) string {
	if git == nil {
		return lipgloss.Place(width, height/2, lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No git data available"))
	}

	var sections []string

	// Top: branch and sync status
	sections = append(sections, renderGitHeader(git, width))

	// Dirty files
	if !git.IsClean && len(git.UncommittedFiles) > 0 {
		sections = append(sections, renderDirtyFiles(git.UncommittedFiles))
	}

	// Recent commits table
	sections = append(sections, renderCommitsTable(git.RecentCommits, selected))

	// Bottom: last push time
	sections = append(sections, renderGitFooter(git))

	return strings.Join(sections, "\n\n")
}

func renderGitHeader(git *models.GitStatus, width int) string {
	branch := TitleStyle.Render("Branch: " + git.Branch)

	var badge string
	if git.IsClean {
		badge = SuccessStyle.Render(" Clean")
	} else {
		badge = ErrorStyle.Render(fmt.Sprintf(" %d uncommitted", len(git.UncommittedFiles)))
	}

	var syncInfo string
	if git.Ahead > 0 || git.Behind > 0 {
		syncInfo = "  " + WarningStyle.Render(fmt.Sprintf("ahead %d  behind %d", git.Ahead, git.Behind))
	}

	return branch + "  " + badge + syncInfo
}

func renderDirtyFiles(files []string) string {
	title := ErrorStyle.Render("Uncommitted Files:")
	var lines []string
	for _, f := range files {
		lines = append(lines, "  "+ErrorStyle.Render(f))
	}
	return title + "\n" + strings.Join(lines, "\n")
}

func renderCommitsTable(commits []models.CommitInfo, selected int) string {
	if len(commits) == 0 {
		return DimStyle.Render("No recent commits")
	}

	title := TitleStyle.Render("Recent Commits")
	var rows []string

	for i, c := range commits {
		hash := DimStyle.Render(c.ShortHash)
		msg := Truncate(c.Message, 50)
		author := DimStyle.Render(c.Author)
		ts := DimStyle.Render(FormatTimestamp(c.Time))

		row := fmt.Sprintf("  %s  %s  %s  %s", hash, msg, author, ts)
		if i == selected {
			row = SelectedStyle.Render(row)
		}
		rows = append(rows, row)
	}

	return title + "\n" + strings.Join(rows, "\n")
}

func renderGitFooter(git *models.GitStatus) string {
	var parts []string
	if git.LastPushTime != nil {
		parts = append(parts, DimStyle.Render("Last push: "+FormatTimestamp(*git.LastPushTime)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  ")
}

// RenderCommitDetail renders a detail view for a single commit.
func RenderCommitDetail(commit models.CommitInfo, width, height int) string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Commit: "+commit.ShortHash))
	lines = append(lines, "")
	lines = append(lines, "  Hash:    "+commit.Hash)
	lines = append(lines, "  Author:  "+commit.Author)
	lines = append(lines, "  Date:    "+commit.Time.Format("2006-01-02 15:04:05"))
	lines = append(lines, "")
	lines = append(lines, "  "+commit.Message)
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Press Esc to close"))

	return CardStyle.Width(width - 4).Render(strings.Join(lines, "\n"))
}
