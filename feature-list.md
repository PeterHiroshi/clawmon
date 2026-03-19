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
| 2.4 | Dashboard tab (overview: workspace count, task summary, git summary) | 🟢 | feature/phase2-tui | 2x2 card grid, quick activity feed |
| 2.5 | Tasks tab (table: task name, status, duration, model, teams mode) | 🟢 | feature/phase2-tui | Color-coded table, detail view |
| 2.6 | Git tab (sync status, recent commits, ahead/behind, dirty files) | 🟢 | feature/phase2-tui | Branch badges, commit table, dirty file list |
| 2.7 | Activity tab (timeline of recent events, color-coded by type) | 🟢 | feature/phase2-tui | Vertical timeline with type icons |
| 2.8 | Real-time updates (SSE → live refresh) | 🟢 | feature/phase2-tui | SSE subscription, targeted refresh |
| 2.9 | Detail views (Enter on task/commit for full info) | 🟢 | feature/phase2-tui | Task detail, commit detail overlays |
| 2.10 | Integration tests (client ↔ mock daemon) | 🟢 | feature/phase2-tui | 12 integration tests (client + app + nav) |
| 2.11 | E2E tests (daemon + TUI interaction) | 🟢 | feature/phase2-tui | 10 E2E tests (all tabs, detail views, SSE, workflow) |

## Phase 3 — Polish & Documentation

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 3.1 | README with screenshots, install guide, usage | 🟢 | feature/phase3-polish | Full docs: arch diagram, API ref, TUI mockup, config guide |
| 3.2 | Multi-workspace support (monitor multiple agents) | 🟢 | feature/phase3-polish | w key cycles workspaces, [n/m] indicator, data refetch |
| 3.3 | Error handling & graceful degradation (daemon offline, partial data) | 🟢 | feature/phase3-polish | Reconnect w/ exp backoff, stale indicators, degraded data |
| 3.4 | Configuration file support (~/.clawmon/config.toml) | 🟢 | feature/phase3-polish | toml crate (Rust), BurntSushi/toml (Go), CLI overrides file |
| 3.5 | Cross-platform build (Linux, macOS) | 🟢 | feature/phase3-polish | CI workflow, Makefile targets, /proc fallback to ps |

## Phase 4 — System Resources

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 4.1 | System resource collector (sysinfo crate) | 🟢 | feature/system-resources | CPU, memory, disk, GPU |
| 4.2 | System API endpoint + SSE events | 🟢 | feature/system-resources | GET /workspaces/{id}/system + system_update |
| 4.3 | System tab with color-coded bars | 🟢 | feature/system-resources | Tab 5, thresholds: green/yellow/orange/red |
| 4.4 | Dashboard system card | 🟢 | feature/system-resources | CPU + RAM mini-bars, GPU status |
| 4.5 | Unit + integration + E2E tests | 🟢 | feature/system-resources | Rust + Go tests, full coverage |

## Phase 5 — Release Packaging & Documentation

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 5.1 | Version command (--version) for both binaries | 🟢 | feature/release-packaging | Cargo.toml version + Go ldflags |
| 5.2 | GitHub Actions release workflow (4 platform tarballs) | 🟢 | feature/release-packaging | v* tag trigger, SHA256 checksums |
| 5.3 | One-line install script (curl pipe bash) | 🟢 | feature/release-packaging | OS/arch detect, curl/wget, NO_COLOR |
| 5.4 | OpenClaw integration script (bare-metal mode) | 🟢 | feature/release-packaging | Systemd service, shell alias |
| 5.5 | OpenClaw integration script (docker mode) | 🟢 | feature/release-packaging | Container copy, workspace detect |
| 5.6 | Example config + README update | 🟢 | feature/release-packaging | config.example.toml, install docs |
| 5.7 | Static linking for Docker (musl target) | 🔴 | — | Future: musl cross-compile |
| 5.8 | Tests for install/integrate scripts | 🟢 | feature/release-packaging | Version, YAML validation, script logic |

## Phase 6 — Docker Integration

| # | Feature | Status | Branch | Notes |
|---|---------|--------|--------|-------|
| 6.1 | Daemon configurable bind address (--bind) | 🟢 | feature/docker-integration | Default 127.0.0.1, Docker 0.0.0.0 |
| 6.2 | Daemon --mode docker flag | 🟢 | feature/docker-integration | standalone/docker, auto-sets bind |
| 6.3 | Enhanced Docker integration script | 🟢 | feature/docker-integration | Binary verify, config, port detect, bootstrap |
| 6.4 | Port mapping detection and guidance | 🟢 | feature/docker-integration | 3 options: remap, socat, in-container |
| 6.5 | Container restart bootstrap hook | 🟢 | feature/docker-integration | clawmon-bootstrap.sh auto-start |
| 6.6 | TUI-only install option (--tui-only) | 🟢 | feature/docker-integration | Skip daemon, print remote connect instructions |
| 6.7 | Health endpoint enhancement | 🟢 | feature/docker-integration | bind_address, workspaces_count, mode fields |
| 6.8 | Tests (bind, docker mode, install --tui-only) | 🔴 | — | — |
