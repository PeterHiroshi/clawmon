# Phase 3 — Polish & Documentation Implementation Plan

## Task 1: Configuration file support (~/.clawmon/config.toml)
**Scope:** 3.4 — Foundation for other features

### Daemon (Rust)
1. Add `toml` crate to Cargo.toml
2. Create `daemon/src/config_file.rs`:
   - Define `ConfigFile`, `DaemonConfig`, `WorkspacesConfig`, `TuiConfig` structs (serde Deserialize)
   - `load_config_file()` → reads `~/.clawmon/config.toml`, returns `Option<ConfigFile>`
   - `create_default_config()` → writes default config if none exists
   - Tilde expansion helper for paths
3. Update `daemon/src/config.rs`:
   - `Config::from_args()` loads config file first, CLI args override
   - Port, poll_interval, workspace paths merge from file + CLI
4. Unit tests: TOML parsing, defaults, CLI override, missing file, malformed file
5. Export config_file module from `lib.rs`

### TUI (Go)
1. Add `github.com/BurntSushi/toml` to go.mod
2. Create `tui/internal/config/config.go`:
   - `ConfigFile` struct with Daemon and TUI sections
   - `LoadConfig()` → reads `~/.clawmon/config.toml`
   - `CreateDefaultConfig()` → writes default if missing
3. Update `tui/cmd/clawmon-tui/main.go`:
   - Load config file, CLI flags override
4. Unit tests: TOML loading, defaults, CLI override

**Commit:** `feat(build): add configuration file support (~/.clawmon/config.toml)`

## Task 2: Error handling & graceful degradation
**Scope:** 3.3

### Daemon
1. Update each collector to catch and wrap errors gracefully:
   - `git.rs`: handle non-git-repo case → return default GitStatus with error field
   - `process.rs`: handle /proc unavailable → return empty list
   - `tasks.rs`: handle malformed JSON → log warning, skip task
2. Update handlers to return partial data + error messages instead of 500s
3. Unit tests for error cases

### TUI
1. Create `tui/internal/client/reconnect.go`:
   - Exponential backoff logic: 1s, 2s, 4s, ..., max 30s
   - `ReconnectState` struct tracking attempts, last attempt time, backoff
2. Update `app.go`:
   - Add `StaleData` flag, `LastSuccessfulFetch` timestamp per data type
   - On daemon offline: show "Daemon offline — retrying in Ns" in status bar
   - On reconnect: clear stale flags, resume normal operation
   - Each tab independently handles missing data with error message
3. Update views to show "stale" indicator when data is stale
4. Unit tests: backoff calculation, stale data display, partial data handling

**Commit:** `feat(daemon): add graceful error handling for collector failures`
**Commit:** `feat(tui): add reconnection logic and graceful degradation`

## Task 3: Multi-workspace support
**Scope:** 3.2

### TUI
1. Update `app.go`:
   - Add `Workspaces []WorkspaceInfo`, `SelectedWorkspace int`
   - `w` key cycles through workspaces
   - Number keys `1-9` in workspace picker (if multiple)
   - Refetch data when workspace changes
2. Update dashboard view:
   - Show workspace name in header
   - If multiple workspaces: show aggregate stats or per-workspace cards
3. Update all tab views to show current workspace name
4. Update status bar to show workspace indicator
5. Tests for workspace switching

**Commit:** `feat(tui): add multi-workspace support with workspace switcher`

## Task 4: Cross-platform build
**Scope:** 3.5

### Daemon
1. Update `process.rs`:
   - Add `#[cfg(target_os = "linux")]` and `#[cfg(target_os = "macos")]` guards
   - macOS: use `sysctl`/`ps` based process collection
   - Ensure fallback path works on macOS
2. Update `config.rs` / `config_file.rs`:
   - Cross-platform tilde expansion (use `dirs` crate or env-based)
3. Verify no Linux-only code without fallback

### Build
1. Add Makefile targets: `build-linux`, `build-macos`
2. Create `.github/workflows/ci.yml`:
   - Matrix: [ubuntu-latest, macos-latest]
   - Setup Rust + Go
   - make build, make test, make lint
3. Tests compile on both platforms (can only verify Linux here)

**Commit:** `feat(build): add cross-platform support and CI workflow`

## Task 5: README with documentation
**Scope:** 3.1

1. Rewrite `README.md`:
   - Project description and motivation
   - Architecture diagram (ASCII art)
   - Prerequisites
   - Build from source
   - Quick start guide
   - Configuration (CLI flags + config.toml)
   - Keyboard shortcuts table
   - API reference
   - Contributing section
   - License
   - ASCII art TUI mockup
2. No tests needed

**Commit:** `docs: comprehensive README with install guide, usage, and API reference`

## Task 6: Final verification & feature-list update
1. Run all tests: `make test`
2. Run lints: `make lint`
3. Update feature-list.md: all 3.x items → 🟢
4. Update progress.md

**Commit:** `chore: mark Phase 3 complete in feature-list`
