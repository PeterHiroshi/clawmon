package views

import (
	"testing"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderGitNil(t *testing.T) {
	result := RenderGit(nil, 0, 80, 24)
	assert.Contains(t, result, "No git data")
}

func TestRenderGitClean(t *testing.T) {
	git := &models.GitStatus{
		Branch:  "main",
		IsClean: true,
		RecentCommits: []models.CommitInfo{
			{Hash: "abc123def", ShortHash: "abc123d", Message: "feat: initial", Author: "Test", Time: time.Now()},
		},
	}
	result := RenderGit(git, 0, 80, 24)
	assert.Contains(t, result, "main")
	assert.Contains(t, result, "Clean")
	assert.Contains(t, result, "abc123d")
}

func TestRenderGitDirty(t *testing.T) {
	git := &models.GitStatus{
		Branch:           "develop",
		IsClean:          false,
		UncommittedFiles: []string{"src/main.rs", "Cargo.toml"},
		Ahead:            2,
		Behind:           1,
	}
	result := RenderGit(git, 0, 80, 24)
	assert.Contains(t, result, "2 uncommitted")
	assert.Contains(t, result, "ahead 2")
	assert.Contains(t, result, "behind 1")
	assert.Contains(t, result, "src/main.rs")
}

func TestRenderCommitDetail(t *testing.T) {
	commit := models.CommitInfo{
		Hash:      "abc123def456789",
		ShortHash: "abc123d",
		Message:   "feat: add something cool",
		Author:    "Test User",
		Time:      time.Date(2026, 3, 18, 19, 0, 0, 0, time.UTC),
	}
	result := RenderCommitDetail(commit, 80, 24)
	assert.Contains(t, result, "abc123d")
	assert.Contains(t, result, "abc123def456789")
	assert.Contains(t, result, "Test User")
	assert.Contains(t, result, "feat: add something cool")
}
