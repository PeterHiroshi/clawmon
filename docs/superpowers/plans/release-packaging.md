# Plan: Phase 5 — Release Packaging & Documentation

## Tasks

### Task 1: Add --version flag to both binaries
- Daemon: already uses `VERSION` from `env!("CARGO_PKG_VERSION")` — add `#[command(version = VERSION)]` to clap
- TUI: add `-version` flag, inject version via Go ldflags (`-ldflags "-X main.version=0.1.0"`)
- Update Makefile build-tui to use ldflags
- Test: verify both `--version` flags work

### Task 2: Create config.example.toml
- Create `config.example.toml` at project root with documented example config

### Task 3: Create install script (scripts/install.sh)
- Detect OS/ARCH, fetch latest release, download tarball, extract binaries
- Install to ~/.local/bin or /usr/local/bin
- Create default config, verify, print instructions
- Support NO_COLOR, curl/wget fallback

### Task 4: Create OpenClaw integration script (scripts/openclaw-integrate.sh)
- Mode A: bare-metal — auto-detect workspace, create systemd service, add alias
- Mode B: docker — detect container, copy binaries, start daemon, print instructions
- Both modes: idempotent, colorful output, NO_COLOR support

### Task 5: Create GitHub Actions release workflow (.github/workflows/release.yml)
- Trigger on v* tags
- Build matrix: linux-amd64, linux-arm64, macos-amd64, macos-arm64
- Package tarballs with binaries + install.sh + config.example.toml + README + LICENSE
- Create release with checksums

### Task 6: Update README.md with installation section
- Add quick install, OpenClaw integration, binary releases sections

### Task 7: Update feature-list.md with Phase 5
- Add all 5.x items

### Task 8: Add tests for version commands and script logic
- Test --version flag for both binaries
- Validate release.yml YAML syntax
- Test script OS/arch detection functions

### Task 9: Format and verify
- Run cargo fmt, gofmt
- Run all tests
- Final commit if needed
