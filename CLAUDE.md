# CLAUDE.md — clawmon

## Project Overview
**clawmon** is a terminal-based monitoring dashboard for OpenClaw AI agents. It provides real-time visibility into workspace sync status, task dispatch pipelines, process health, and activity timelines.

**Architecture:** Dual-language — Rust daemon (data collection + API) + Go TUI (visualization).

## Tech Stack

### Rust Daemon (`daemon/`)
- **Language:** Rust 1.82+ (2024 edition)
- **Key crates:**
  - `notify` — filesystem watching (cross-platform)
  - `git2` — native git operations (libgit2 bindings)
  - `axum` — lightweight HTTP API server
  - `tokio` — async runtime
  - `serde` / `serde_json` — serialization
  - `chrono` — timestamps
  - `clap` — CLI argument parsing
- **API:** Local HTTP (default `127.0.0.1:9876`) — JSON REST API
- **Build:** `cargo build --release`
- **Test:** `cargo test`

### Go TUI (`tui/`)
- **Language:** Go 1.23+
- **Key modules:**
  - `github.com/charmbracelet/bubbletea` — TUI framework
  - `github.com/charmbracelet/lipgloss` — styling
  - `github.com/charmbracelet/bubbles` — UI components (table, spinner, viewport)
- **Build:** `go build -o clawmon-tui ./cmd/clawmon-tui`
- **Test:** `go test ./...`

## Project Structure
```
clawmon/
├── daemon/                    # Rust daemon
│   ├── Cargo.toml
│   └── src/
│       ├── main.rs            # Entry point, CLI, server startup
│       ├── config.rs          # Configuration (workspace paths, intervals)
│       ├── api/               # HTTP API handlers
│       │   ├── mod.rs
│       │   ├── routes.rs      # Route definitions
│       │   └── handlers.rs    # Request handlers
│       ├── collectors/        # Data collection modules
│       │   ├── mod.rs
│       │   ├── git.rs         # Git status, commits, sync state
│       │   ├── tasks.rs       # forge-dispatch task status
│       │   ├── process.rs     # Claude Code process health
│       │   └── env.rs         # Environment/bootstrap health
│       ├── models.rs          # Shared data types
│       └── error.rs           # Error types
├── tui/                       # Go TUI
│   ├── go.mod
│   ├── cmd/clawmon-tui/
│   │   └── main.go           # Entry point
│   └── internal/
│       ├── client/            # HTTP client for daemon API
│       │   └── client.go
│       ├── models/            # Data models (mirrors daemon API)
│       │   └── models.go
│       ├── app/               # Application state & update loop
│       │   └── app.go
│       └── views/             # Tab views
│           ├── dashboard.go   # Overview tab
│           ├── tasks.go       # Task list tab
│           ├── git.go         # Git sync tab
│           └── activity.go    # Activity timeline tab
├── Makefile                   # Unified build/test/run commands
├── feature-list.md            # Feature progress tracking
├── CLAUDE.md                  # This file
├── LICENSE                    # MIT
└── README.md
```

## Coding Standards

### Rust
- Follow Rust 2024 edition idioms
- Use `thiserror` for error types, `anyhow` only in main/tests
- All public APIs must have doc comments (`///`)
- Use `clippy` lint level: `warn` — code must pass `cargo clippy` with no warnings
- Format with `rustfmt` (default config)
- Prefer `&str` over `String` in function parameters where ownership isn't needed
- Use `Result<T, E>` — never `unwrap()` in library code (ok in tests)
- Async functions use `tokio` runtime — no blocking calls in async context

### Go
- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `golangci-lint` if available
- Error handling: always check errors, wrap with context using `fmt.Errorf("...: %w", err)`
- Use interfaces for testability (e.g., `DaemonClient` interface for the HTTP client)
- Table-driven tests preferred
- No global mutable state — pass dependencies via struct fields

### General
- Constants over magic values — ALL literals (ports, timeouts, intervals, paths) go in config/constants
- Descriptive variable names — no single-letter vars except loop counters
- Comments explain WHY, not WHAT

## Testing Conventions

### Rust
- Unit tests: `#[cfg(test)] mod tests` in each source file
- Integration tests: `daemon/tests/` directory
- E2E tests: `daemon/tests/e2e/` — start real server, make HTTP requests, verify responses
- Use `tempdir` for tests that need filesystem state
- Use `tokio::test` for async tests
- Run: `cargo test` (all), `cargo test --test e2e_*` (E2E only)

### Go
- Unit tests: `*_test.go` alongside source files
- Integration tests: `tui/tests/integration/`
- E2E tests: `tui/tests/e2e/` — start daemon + TUI, verify rendering
- Use `testify/assert` for assertions
- Use `httptest` for mocking daemon API in unit tests
- Run: `go test ./...` (all), `go test ./tests/e2e/...` (E2E only)

### Coverage
- Aim for >80% on core logic (collectors, client, models)
- 100% on API contract (request/response shapes)
- E2E must cover all major user workflows

## Git Conventions
- Branch naming: `feature/<name>` from `develop`
- Commit format: `type(scope): description`
  - Types: `feat`, `fix`, `test`, `docs`, `refactor`, `chore`
  - Scopes: `daemon`, `tui`, `api`, `build`
  - Example: `feat(daemon): add git status collector`
- One commit per plan task — NOT one big commit at the end
- Commit feature-list.md alongside code changes

## Forbidden Patterns
- `.forge-task/` in commits — must be in `.gitignore`
- `unwrap()` / `expect()` in non-test Rust code (use `?` operator)
- Magic numbers/strings — use constants
- Blocking calls in async context (Rust: no `std::thread::sleep` in async, Go: no blocking in bubbletea Update)
- Global mutable state
- Hardcoded `127.0.0.1:9876` — always read from config/CLI args with this as default

## Build & Run
```bash
# Build everything
make build

# Run daemon
make run-daemon

# Run TUI (connects to running daemon)
make run-tui

# Run all tests
make test

# Run E2E tests only
make test-e2e

# Lint
make lint
```

## API Contract (Daemon → TUI)

Base URL: `http://127.0.0.1:9876/api/v1`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Daemon health check |
| `/workspaces` | GET | List monitored workspaces |
| `/workspaces/{id}/git` | GET | Git sync status for a workspace |
| `/workspaces/{id}/tasks` | GET | Task dispatch status |
| `/workspaces/{id}/processes` | GET | Claude Code process health |
| `/workspaces/{id}/env` | GET | Environment/bootstrap health |
| `/workspaces/{id}/activity` | GET | Recent activity timeline |
| `/events` | GET (SSE) | Server-sent events for real-time updates |

All responses follow:
```json
{
  "data": { ... },
  "timestamp": "2026-03-18T19:00:00Z"
}
```
