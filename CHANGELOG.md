# Changelog

All notable changes to this project will be documented in this file.

## [v0.2.0] — 2026-03-19

### Added
- **Configurable bind address** (`--bind`) — daemon can listen on any interface (default `127.0.0.1`, Docker mode uses `0.0.0.0`)
- **Docker mode** (`--mode docker`) — auto-configures daemon for containerized environments (bind `0.0.0.0`, relaxed timeouts)
- **Enhanced Docker integration script** — binary verification, auto-config generation, port mapping detection with 3 connectivity options (remap, socat, in-container TUI)
- **Container restart bootstrap hook** (`clawmon-bootstrap.sh`) — auto-starts daemon on container restart
- **TUI-only install** (`--tui-only`) — installs only the TUI binary with remote daemon connection instructions
- **Enhanced health endpoint** — now includes `bind_address`, `workspaces_count`, and `mode` fields
- **Script tests** — 20 new tests for Docker integration and `--tui-only` install flows

### Changed
- Install script now supports `--tui-only` flag and improved Docker detection
- OpenClaw integration script rewritten with better error handling and port mapping guidance

### Fixed
- Clippy warning: renamed `from_str` to `parse` to avoid confusion with `FromStr` trait

## [v0.1.0] — 2026-03-19

Initial release with full monitoring stack.

### Added
- **Rust daemon** (axum + tokio) — 4 collectors (Git, Task, Process, Environment), HTTP API, SSE streaming, file watcher
- **Go TUI** (Bubble Tea) — 5 tabs (Dashboard, Tasks, Git, Activity, System), real-time SSE updates, detail views
- **System resource monitoring** — CPU, memory, disk, GPU with color-coded progress bars
- **Multi-workspace support** — monitor multiple OpenClaw agents from one TUI
- **Configuration** — TOML config file + CLI overrides
- **Release packaging** — GitHub Actions CI/CD, 4-platform tarballs (linux-amd64/arm64, macos-amd64/arm64), SHA256 checksums
- **Install scripts** — one-line curl installer + OpenClaw integration scripts (bare-metal & Docker)
- **Comprehensive tests** — unit, integration, and E2E tests for both Rust and Go components
