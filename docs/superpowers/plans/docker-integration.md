# Plan: Docker Integration

## Tasks (one commit each)

### Task 1: Daemon configurable bind address (--bind) + --mode docker flag
- Add `--bind` CLI arg to `CliArgs` (default: `127.0.0.1`)
- Add `bind` field to `DaemonFileConfig` in config_file.rs
- Add `--mode` CLI arg (standalone/docker) to `CliArgs`
- Add `bind` and `mode` fields to `Config` struct
- Wire through `from_args_with_config`: CLI > config file > default
- When mode=docker, default bind to `0.0.0.0` unless explicitly set
- In main.rs, use `config.bind` instead of hardcoded `[127, 0, 0, 1]`
- Log mode on startup
- Tests: bind address config resolution, mode flag parsing

### Task 2: Health endpoint enhancement
- Add fields to `HealthResponse`: `bind_address`, `workspaces_count`, `mode`
- Update health handler to populate new fields from AppState
- AppState needs access to bind and mode
- Tests: verify health response contains new fields

### Task 3: Enhanced Docker integration script
- Rewrite `integrate_docker()` in openclaw-integrate.sh
- Add port mapping detection (`docker port`)
- Add bootstrap hook creation
- Add detailed guidance for port mapping options
- Print enhanced summary

### Task 4: TUI-only install option (--tui-only)
- Add `--tui-only` flag parsing to install.sh
- When set, only download/install clawmon-tui
- Print appropriate instructions

### Task 5: Update feature-list.md with Phase 6
- Add Phase 6 table with all items
- Mark completed items as done

### Task 6: Tests for new features
- Rust: test bind config, mode flag, health response
- Shell: test --tui-only flag, docker integration logic
- Ensure all existing tests pass

### Task 7: Format and verify
- cargo fmt, gofmt
- cargo test, go test ./...
- cargo clippy
