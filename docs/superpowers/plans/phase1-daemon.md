# Phase 1: Rust Daemon Implementation Plan

## Task 1: Project Scaffolding
**Files:** `daemon/Cargo.toml`, `daemon/src/main.rs`, `daemon/src/error.rs`, `daemon/src/lib.rs`
**Steps:**
- [ ] Create `daemon/Cargo.toml` with all dependencies (axum, tokio, serde, git2, notify, clap, chrono, thiserror, anyhow)
- [ ] Create `daemon/src/error.rs` with `DaemonError` enum using thiserror
- [ ] Create `daemon/src/lib.rs` re-exporting modules
- [ ] Create `daemon/src/main.rs` with minimal entry point that prints version
- [ ] Verify: `cargo build`, `cargo test`, `cargo clippy`
**Commit:** `chore(daemon): scaffold project structure with Cargo.toml and error types`

## Task 2: Configuration Module
**Files:** `daemon/src/config.rs`
**Steps:**
- [ ] Write tests for config parsing (default values, CLI overrides, workspace detection)
- [ ] Implement `Config` struct with clap derive: port, workspace, poll_interval, project_dirs
- [ ] Implement workspace auto-detection (OPENCLAW_WORKSPACE env var, fallback to ~/.openclaw/workspace-main)
- [ ] Implement project directory scanning (find subdirs with .git/)
- [ ] Verify: `cargo test config`, `cargo clippy`
**Commit:** `feat(daemon): add configuration module with CLI args and workspace detection`

## Task 3: Models Module
**Files:** `daemon/src/models.rs`
**Steps:**
- [ ] Define `ApiResponse<T>` wrapper with data + timestamp
- [ ] Define `HealthResponse`, `WorkspaceInfo`
- [ ] Define `GitStatus` (branch, clean/dirty, uncommitted files, ahead/behind, last commit, recent commits)
- [ ] Define `TaskInfo` (name, status enum, started_at, completed_at, model, progress, exit_code)
- [ ] Define `ProcessInfo` (pid, workdir, cpu, memory, uptime, state enum, stalled)
- [ ] Define `EnvCheck` (tool name, installed, version, path)
- [ ] Define `ActivityEvent` (timestamp, event_type, workspace, description)
- [ ] Write serialization round-trip tests
- [ ] Verify: `cargo test models`, `cargo clippy`
**Commit:** `feat(daemon): add shared data models with serde serialization`

## Task 4: Git Collector
**Files:** `daemon/src/collectors/mod.rs`, `daemon/src/collectors/git.rs`
**Steps:**
- [ ] Write tests using tempdir with real git repos (init, add, commit, check status)
- [ ] Implement `GitCollector::new(path)` and `GitCollector::collect() -> Result<GitStatus>`
- [ ] Implement repo status detection (clean/dirty, uncommitted files list)
- [ ] Implement branch info (current branch, ahead/behind tracking branch)
- [ ] Implement commit history (last commit details, recent 20 commits)
- [ ] Implement last push time from reflog
- [ ] Verify: `cargo test git`, `cargo clippy`
**Commit:** `feat(daemon): add git status collector with native git2 operations`

## Task 5: Task Collector
**Files:** `daemon/src/collectors/tasks.rs`
**Steps:**
- [ ] Write tests with tempdir containing mock .forge-task/ directories
- [ ] Implement `TaskCollector::new(workspace_path)` and `collect() -> Result<Vec<TaskInfo>>`
- [ ] Parse task-spec.json for name, status, started_at, completed_at, model, agent_teams
- [ ] Parse progress.md for current step and percentage
- [ ] Parse meta.json for exit_code, effective_model, duration
- [ ] Sort tasks by most recent first
- [ ] Verify: `cargo test tasks`, `cargo clippy`
**Commit:** `feat(daemon): add task collector for .forge-task/ metadata parsing`

## Task 6: Process Collector
**Files:** `daemon/src/collectors/process.rs`
**Steps:**
- [ ] Write tests with mock /proc data (or integration tests for live system)
- [ ] Implement `ProcessCollector::new()` and `collect() -> Result<Vec<ProcessInfo>>`
- [ ] Implement /proc-based process detection (scan for claude/claude-code processes)
- [ ] Extract PID, cwd (from /proc/PID/cwd symlink), state, uptime
- [ ] Implement CPU/memory reading from /proc/PID/stat and /proc/PID/status
- [ ] Implement stall detection (no file mods in workdir for >15min)
- [ ] Add ps aux fallback for non-Linux
- [ ] Verify: `cargo test process`, `cargo clippy`
**Commit:** `feat(daemon): add process collector for Claude Code detection`

## Task 7: Environment Collector
**Files:** `daemon/src/collectors/env.rs`
**Steps:**
- [ ] Write tests for tool checking (mock which/command -v)
- [ ] Implement `EnvCollector::new()` and `collect() -> Result<Vec<EnvCheck>>`
- [ ] Check tool availability: claude, go, rustc, cargo, gh, glab, git, make, jq
- [ ] Check hook installation: ~/.claude/hooks/notify-forge.sh exists + executable
- [ ] Check plugin status: read ~/.claude/settings.json for Superpowers
- [ ] Verify: `cargo test env`, `cargo clippy`
**Commit:** `feat(daemon): add environment collector for tool and health checks`

## Task 8: HTTP API Server
**Files:** `daemon/src/api/mod.rs`, `daemon/src/api/routes.rs`, `daemon/src/api/handlers.rs`
**Steps:**
- [ ] Write handler unit tests with mock collector data
- [ ] Implement shared AppState (collectors, config, start_time)
- [ ] Implement route definitions: /health, /workspaces, /workspaces/{id}/git|tasks|processes|env|activity
- [ ] Implement each handler returning ApiResponse<T> JSON
- [ ] Implement 404 for unknown workspace, 500 for collector failures
- [ ] Wire into main.rs with axum Router + graceful shutdown
- [ ] Verify: `cargo test api`, `cargo clippy`
**Commit:** `feat(api): add HTTP REST API with all endpoints and handlers`

## Task 9: SSE Event Stream
**Files:** `daemon/src/api/handlers.rs` (add SSE handler), `daemon/src/events.rs`
**Steps:**
- [ ] Write tests for event channel and SSE formatting
- [ ] Implement `EventBus` with tokio broadcast channel
- [ ] Implement SSE handler at GET /events using axum's Sse
- [ ] Wire collectors to publish events on data changes
- [ ] Verify: `cargo test events`, `cargo clippy`
**Commit:** `feat(api): add SSE event stream for real-time updates`

## Task 10: File Watcher
**Files:** `daemon/src/watcher.rs`
**Steps:**
- [ ] Write tests for watcher setup and event filtering
- [ ] Implement `FileWatcher::new(paths)` using notify crate
- [ ] Filter events to relevant changes (.forge-task/, git objects, source files)
- [ ] Trigger collector refresh on relevant file changes
- [ ] Integrate with main loop and event bus
- [ ] Verify: `cargo test watcher`, `cargo clippy`
**Commit:** `feat(daemon): add filesystem watcher with notify crate`

## Task 11: Integration Tests
**Files:** `daemon/tests/integration_test.rs`
**Steps:**
- [ ] Test collector -> API handler flow with real tempdir workspaces
- [ ] Test multiple workspace handling
- [ ] Test partial data scenarios (missing .forge-task/, no git repo)
- [ ] Verify: `cargo test --test integration_test`
**Commit:** `test(daemon): add integration tests for collector-to-API flow`

## Task 12: E2E Tests
**Files:** `daemon/tests/e2e/e2e_daemon.rs`
**Steps:**
- [ ] Start actual daemon binary on random port
- [ ] Test GET /health returns valid response
- [ ] Test GET /workspaces returns workspace list
- [ ] Test all workspace sub-endpoints return valid JSON
- [ ] Test 404 for unknown workspace
- [ ] Test SSE /events endpoint connects and receives data
- [ ] Verify: `cargo test --test e2e_daemon`
**Commit:** `test(daemon): add E2E tests for full daemon HTTP API`

## Task 13: Final Cleanup & Verification
**Steps:**
- [ ] Run full `cargo test`
- [ ] Run `cargo clippy -- -W warnings` — zero warnings
- [ ] Run `cargo fmt --check` — properly formatted
- [ ] Update feature-list.md: all Phase 1 items 🟢
- [ ] Rebase on develop, push feature branch
**Commit:** `chore(daemon): final cleanup and feature-list update`
