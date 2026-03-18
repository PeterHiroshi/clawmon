# Feature List — clawmon

## Status Legend
- 🔴 Not started
- 🟡 In progress
- 🟢 Complete
- ⚫ Blocked

## Phase 1 — Rust Daemon Core

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 1.1 | Project scaffolding (Cargo.toml, dir structure, Makefile) | 🔴 | — | — |
| 1.2 | Config module (CLI args, workspace paths, port, intervals) | 🔴 | — | — |
| 1.3 | Git collector (status, commits, ahead/behind, last push) | 🔴 | — | — |
| 1.4 | Task collector (parse .forge-task/ meta, status, progress) | 🔴 | — | — |
| 1.5 | Process collector (Claude Code PID detection, uptime, state) | 🔴 | — | — |
| 1.6 | Environment collector (bootstrap health, tool availability) | 🔴 | — | — |
| 1.7 | HTTP API server (axum, all endpoints, JSON responses) | 🔴 | — | — |
| 1.8 | SSE event stream (real-time push to TUI) | 🔴 | — | — |
| 1.9 | File watcher (notify crate, trigger collectors on change) | 🔴 | — | — |
| 1.10 | Integration tests (API + collectors) | 🔴 | — | — |
| 1.11 | E2E tests (start daemon, hit API, verify responses) | 🔴 | — | — |

## Phase 2 — Go TUI

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 2.1 | Project scaffolding (go.mod, dir structure, Makefile integration) | 🔴 | — | — |
| 2.2 | Daemon HTTP client (all endpoints, SSE subscription) | 🔴 | — | — |
| 2.3 | App shell (tabs, navigation, key bindings, status bar) | 🔴 | — | — |
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
