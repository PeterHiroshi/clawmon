# Plan: System Resource Monitoring

## Task 1: Add system resource models (Rust)
- Add `sysinfo` crate to Cargo.toml
- Add SystemResources, CpuInfo, MemoryInfo, SwapInfo, DiskInfo, GpuInfo structs to models.rs
- Add unit tests for serialization

## Task 2: Implement system collector (Rust)
- Create `collectors/system.rs` using sysinfo crate
- Implement CPU, memory, swap, disk collection
- Implement GPU collection via nvidia-smi (graceful fallback)
- Register in collectors/mod.rs
- Add unit tests

## Task 3: Add system API endpoint (Rust)
- Add `GET /workspaces/{id}/system` route
- Add handler with graceful degradation
- Add `system_update` SSE event type
- Add integration/handler tests

## Task 4: Add Go models + client method
- Add SystemResources Go structs to models/models.go
- Add `GetSystemResources(id string)` to DaemonClient interface
- Implement in client.go
- Add client unit tests

## Task 5: Implement system tab view (Go TUI)
- Create views/system.go with color-coded progress bars
- Add threshold constants and bar rendering
- Add unit tests for bar rendering and color selection

## Task 6: Integrate system tab into app shell
- Add Tab 5 "System" to app.go
- Update tab constants, key bindings, navigation
- Add SystemResources to Model state
- Add SystemMsg message type and fetch/update logic
- Update common.go TabNames

## Task 7: Add system card to dashboard
- Add system resource card to dashboard.go
- Show CPU + memory mini-bars, GPU status
- Add unit tests

## Task 8: Update feature-list.md + E2E tests
- Add Phase 4 section to feature-list.md
- Add E2E tests for /system endpoint
- Verify all existing tests still pass

## Task 9: Final verification
- Run `cargo clippy`, `cargo test`
- Run `go vet ./...`, `go test ./...`
- Rebase on develop, push branch
