# clawmon

[![Release](https://img.shields.io/github/v/release/PeterHiroshi/clawmon)](https://github.com/PeterHiroshi/clawmon/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/PeterHiroshi/clawmon/ci.yml?branch=develop)](https://github.com/PeterHiroshi/clawmon/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Terminal-based monitoring dashboard for [OpenClaw](https://openclaw.ai) AI agents. Track workspace sync, task dispatch, process health, and activity timelines — all from your terminal.

## Why clawmon?

When running AI agents across multiple workspaces, visibility into what's happening becomes critical. clawmon provides a real-time dashboard that shows:

- **Git sync status** — branch, dirty files, ahead/behind, recent commits
- **Task dispatch** — active tasks, progress, duration, models in use
- **Process health** — Claude Code processes, CPU/memory, stall detection
- **Environment** — tool availability, hooks, bootstrap health
- **Activity timeline** — unified feed of all events across workspaces

## Architecture

```
                         ┌──────────────────────────────────────────┐
                         │            Rust Daemon                   │
                         │         (axum + tokio)                   │
┌──────────────────┐     │                                          │
│    Go TUI        │     │  ┌──────────┐  ┌──────────┐             │
│  (Bubble Tea)    │     │  │   Git    │  │  Tasks   │             │
│                  │ HTTP │  │Collector │  │Collector │             │
│ ┌──────────────┐ │◄────┤  └──────────┘  └──────────┘             │
│ │  Dashboard   │ │ SSE │  ┌──────────┐  ┌──────────┐             │
│ │  Tasks       │ │────►│  │ Process  │  │   Env    │             │
│ │  Git         │ │     │  │Collector │  │Collector │             │
│ │  Activity    │ │     │  └──────────┘  └──────────┘             │
│ └──────────────┘ │     │                                          │
│                  │     │  ┌──────────┐  ┌──────────┐             │
│  Status Bar      │     │  │  File    │  │  Event   │             │
│  [workspace]     │     │  │ Watcher  │  │   Bus    │             │
└──────────────────┘     │  └──────────┘  └──────────┘             │
                         └──────────────────────────────────────────┘
```

- **Rust daemon** (`daemon/`) — collects data from the filesystem, git repos, and system processes. Serves a JSON REST API with Server-Sent Events for real-time updates.
- **Go TUI** (`tui/`) — connects to the daemon, renders a tabbed dashboard with live refresh and keyboard navigation.

## Installation

### Quick Install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/PeterHiroshi/clawmon/main/scripts/install.sh | bash
```

This auto-detects your OS/architecture, downloads the latest release, and installs to `~/.local/bin/`.

### OpenClaw Integration

#### Bare Metal (OpenClaw on host)

```bash
clawmon-integrate --mode bare-metal
```

Auto-detects your OpenClaw workspace, creates a systemd service for the daemon, and adds a `clawmon` shell alias.

#### Docker (OpenClaw in container)

```bash
clawmon-integrate --mode docker --container-name <your-container>
```

Copies binaries into the container, starts the daemon, and provides TUI access instructions.

### Binary Releases

Download pre-built binaries from [GitHub Releases](https://github.com/PeterHiroshi/clawmon/releases). Available for linux-amd64, linux-arm64, macos-amd64, and macos-arm64.

### Build from Source

Prerequisites: Rust 1.82+, Go 1.23+, Git, Make

```bash
git clone https://github.com/PeterHiroshi/clawmon.git
cd clawmon
make build
```

This builds:
- `daemon/target/release/clawmon-daemon`
- `tui/clawmon-tui`

## Quick Start

1. **Start the daemon** (in one terminal):

```bash
clawmon-daemon
# or from source:
make run-daemon
```

2. **Start the TUI** (in another terminal):

```bash
clawmon-tui
# or from source:
make run-tui
```

The TUI connects to the daemon at `http://127.0.0.1:9876` by default and begins displaying real-time data.

### Docker Mode (v0.2.0+)

For OpenClaw running inside Docker containers:

```bash
# Daemon inside container — binds to 0.0.0.0 for external TUI access
clawmon-daemon --mode docker

# TUI on host — connects to container's exposed port
clawmon-tui --daemon-url http://localhost:9876/api/v1
```

Use `--tui-only` with the install script to set up just the TUI on your host machine.

## Configuration

### CLI Flags

**Daemon:**

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `9876` | HTTP API port |
| `--bind` | `127.0.0.1` | Bind address (`0.0.0.0` for Docker) |
| `--mode` | `standalone` | Run mode: `standalone` or `docker` |
| `--workspace` | auto-detect | Workspace root path |
| `--poll-interval` | `30` | Poll interval in seconds |
| `--project-dirs` | auto-scan | Additional project directories (comma-separated) |

**TUI:**

| Flag | Default | Description |
|------|---------|-------------|
| `--daemon-url` | `http://127.0.0.1:9876/api/v1` | Daemon API base URL |

### Config File

Both daemon and TUI read from `~/.clawmon/config.toml`. CLI flags override config file values.

```toml
[daemon]
port = 9876
poll_interval = 30

[daemon.workspaces]
paths = [
  "~/.openclaw/workspace-main",
  "~/other-agent-workspace",
]

[tui]
daemon_url = "http://127.0.0.1:9876/api/v1"
refresh_interval = 5
theme = "dark"
```

A default config file is created on first run if none exists.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Switch tabs |
| `1`-`4` | Jump to tab (Dashboard, Tasks, Git, Activity) |
| `j` / `k` or arrows | Navigate up/down in lists |
| `Enter` | Open detail view |
| `Esc` | Close detail/help overlay |
| `w` | Cycle through workspaces |
| `r` | Manual refresh |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit |

## TUI Layout

```
 Dashboard   Tasks   Git   Activity
+---------------------------------------------+
| clawmon  ~/.openclaw/workspace-main          |
|                                              |
| +------------------+ +------------------+   |
| | Tasks            | | Git              |   |
| | Total:     3     | | Branch: develop  |   |
| | Running:   1     | | Clean            |   |
| | Completed: 2     | | ahead 0  behind 0|   |
| | Failed:    0     | | feat: add api... |   |
| +------------------+ +------------------+   |
| +------------------+ +------------------+   |
| | Processes        | | Environment      |   |
| | Running: 1       | | Tools: 9/9       |   |
| | CPU:     2.1%    | | All tools avail  |   |
| | Memory:  128 MB  | |                  |   |
| +------------------+ +------------------+   |
|                                              |
| Recent Activity                              |
|  5m ago  commit  abc1234: feat: add api...   |
|  1h ago  task    Task started: phase2-tui    |
+---------------------------------------------+
| * daemon  clawmon  refreshed: just now       |
+---------------------------------------------+
```

## API Reference

Base URL: `http://127.0.0.1:9876/api/v1`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Daemon health check |
| `/workspaces` | GET | List monitored workspaces |
| `/workspaces/{id}/git` | GET | Git sync status |
| `/workspaces/{id}/tasks` | GET | Task dispatch status |
| `/workspaces/{id}/processes` | GET | Claude Code process health |
| `/workspaces/{id}/env` | GET | Environment health |
| `/workspaces/{id}/activity` | GET | Recent activity timeline |
| `/events` | GET (SSE) | Server-sent events stream |

All JSON responses follow the envelope format:

```json
{
  "data": { ... },
  "timestamp": "2026-03-18T19:00:00Z"
}
```

## Error Handling

- **Daemon offline**: TUI shows "Daemon offline" with exponential backoff reconnection (1s, 2s, 4s, ..., max 30s)
- **Stale data**: When the daemon disconnects, existing data is preserved with a `[stale]` indicator
- **Collector failures**: Individual collectors can fail without affecting others; failed collectors return degraded/empty data

## Project Structure

```
clawmon/
+-- daemon/                    # Rust daemon
|   +-- src/
|   |   +-- main.rs            # Entry point, server startup
|   |   +-- config.rs          # CLI args, config resolution
|   |   +-- config_file.rs     # TOML config file support
|   |   +-- models.rs          # Shared data types
|   |   +-- error.rs           # Error types (thiserror)
|   |   +-- events.rs          # SSE event bus
|   |   +-- watcher.rs         # File watcher (notify)
|   |   +-- api/               # HTTP handlers + routes
|   |   +-- collectors/        # git, tasks, process, env
|   +-- Cargo.toml
+-- tui/                       # Go TUI
|   +-- cmd/clawmon-tui/       # Entry point
|   +-- internal/
|   |   +-- app/               # Bubble Tea model
|   |   +-- client/            # HTTP client + SSE + reconnect
|   |   +-- config/            # TOML config file
|   |   +-- models/            # Data models
|   |   +-- views/             # Tab renderers
|   +-- go.mod
+-- .github/workflows/ci.yml   # CI pipeline
+-- Makefile                   # Build/test/lint/run
+-- CLAUDE.md                  # AI coding standards
+-- feature-list.md            # Feature tracking
```

## Contributing

1. Fork the repository
2. Create a feature branch from `develop`: `git checkout -b feature/my-feature develop`
3. Follow the coding standards in [CLAUDE.md](CLAUDE.md)
4. Commit with conventional format: `type(scope): description`
   - Types: `feat`, `fix`, `test`, `docs`, `refactor`, `chore`
   - Scopes: `daemon`, `tui`, `api`, `build`
5. Ensure `make test` and `make lint` pass
6. Push and open a PR against `develop`

## License

[MIT](LICENSE)
