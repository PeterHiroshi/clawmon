# Feature List — clawmon

## Status Legend
- 🔴 Not started
- 🟡 In progress
- 🟢 Complete
- ⚫ Blocked

## Phase 1 — Rust Daemon Core

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 1.1 | Project scaffolding (Cargo.toml, dir structure, Makefile) | 🟢 | feature/phase1-daemon | Cargo.toml, src/ structure, error types |
| 1.2 | Config module (CLI args, workspace paths, port, intervals) | 🟢 | feature/phase1-daemon | clap CLI, workspace auto-detect, project scanning |
| 1.3 | Git collector (status, commits, ahead/behind, last push) | 🟢 | feature/phase1-daemon | git2 native ops, 6 unit tests |
| 1.4 | Task collector (parse .forge-task/ meta, status, progress) | 🟢 | feature/phase1-daemon | task-spec.json, meta.json, progress.md parsing |
| 1.5 | Process collector (Claude Code PID detection, uptime, state) | 🟢 | feature/phase1-daemon | /proc + ps fallback, stall detection |
| 1.6 | Environment collector (bootstrap health, tool availability) | 🟢 | feature/phase1-daemon | 9 tools, hooks, superpowers check |
| 1.7 | HTTP API server (axum, all endpoints, JSON responses) | 🟢 | feature/phase1-daemon | All 7 endpoints + SSE |
| 1.8 | SSE event stream (real-time push to TUI) | 🟢 | feature/phase1-daemon | broadcast channel, tokio-stream |
| 1.9 | File watcher (notify crate, trigger collectors on change) | 🟢 | feature/phase1-daemon | notify v7, filtered events |
| 1.10 | Integration tests (API + collectors) | 🟢 | feature/phase1-daemon | 8 integration tests |
| 1.11 | E2E tests (start daemon, hit API, verify responses) | 🟢 | feature/phase1-daemon | 8 E2E tests, real HTTP |

## Phase 2 — Go TUI

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 2.1 | Project scaffolding (go.mod, dir structure, Makefile integration) | 🟢 | feature/phase2-tui | go.mod, models, main.go stub, 10 tests |
| 2.2 | Daemon HTTP client (all endpoints, SSE subscription) | 🟢 | feature/phase2-tui | 14 tests (10 REST + 4 SSE) |
| 2.3 | App shell (tabs, navigation, key bindings, status bar) | 🟢 | feature/phase2-tui | Bubble Tea model, Update, View, tab bar, status bar, help overlay |
| 2.4 | Dashboard tab (overview: workspace count, task summary, git summary) | 🔴 | — | — |
| 2.5 | Tasks tab (table: task name, status, duration, model, teams mode) | 🔴 | — | — |
| 2.6 | Git tab (sync status, recent commits, ahead/behind, dirty files) | 🔴 | — | — |
| 2.7 | Activity tab (timeline of recent events, color-coded by type) | 🔴 | — | — |
| 2.8 | Real-time updates (SSE → live refresh) | 🔴 | — | — |
| 2.9 | Detail views (Enter on task/commit for full info) | 🔴 | — | — |
| 2.10 | Integration tests (client ↔ mock daemon) | 🔴 | — | — |
| 2.11 | E2E tests (daemon + TUI interaction) | 🔴 | — | — |

## Phase 3 — Polish & Documentation

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 3.1 | README with screenshots, install guide, usage | 🔴 | — | — |
| 3.2 | Multi-workspace support (monitor multiple agents) | 🔴 | — | — |
| 3.3 | Error handling & graceful degradation (daemon offline, partial data) | 🔴 | — | — |
| 3.4 | Configuration file support (~/.clawmon/config.toml) | 🔴 | — | — |
| 3.5 | Cross-platform build (Linux, macOS) | 🔴 | — | — |
