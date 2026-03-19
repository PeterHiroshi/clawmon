# Phase 2: Go TUI Implementation Plan

## Task 1: Project Scaffolding (Feature 2.1)
**Files:** `tui/go.mod`, `tui/cmd/clawmon-tui/main.go`, `tui/internal/models/models.go`
**Steps:**
- [ ] Create `tui/go.mod` with module `github.com/PeterHiroshi/clawmon/tui`, Go 1.23+
- [ ] Run `go mod tidy` to initialize, add bubbletea, lipgloss, bubbles dependencies
- [ ] Create `tui/internal/models/models.go` mirroring daemon API types (ApiResponse, HealthResponse, WorkspaceInfo, GitStatus, CommitInfo, TaskInfo, TaskStatus, ProcessInfo, ProcessState, EnvHealth, EnvCheck, HookCheck, ActivityEvent, ActivityType, SseEvent)
- [ ] Create `tui/internal/models/models_test.go` — test JSON deserialization of each model (table-driven, match daemon's serde output)
- [ ] Create `tui/cmd/clawmon-tui/main.go` — minimal entry point with `--daemon-url` flag and bubbletea program stub
- [ ] Verify: `go build ./...`, `go test ./...`, `go vet ./...`
- [ ] Update feature-list.md: 2.1 → 🟢
**Commit:** `feat(tui): scaffold project structure with models and entry point`

## Task 2: HTTP Client — Interface & REST Methods (Feature 2.2a)
**Files:** `tui/internal/client/client.go`, `tui/internal/client/client_test.go`
**Steps:**
- [ ] Define `DaemonClient` interface with all 7 REST methods (Health, ListWorkspaces, GetGitStatus, GetTasks, GetProcesses, GetEnvHealth, GetActivity)
- [ ] Write test first: `client_test.go` using httptest mock server with realistic JSON responses for each endpoint (table-driven)
- [ ] Implement `HTTPClient` struct with base URL, http.Client (5s timeout)
- [ ] Implement `apiGet[T]` generic helper for GET+JSON decode with error wrapping
- [ ] Implement all 7 REST methods using the helper
- [ ] Add typed error types: `ErrDaemonUnreachable`, `ErrNotFound`, `ErrServerError`
- [ ] Verify: all tests pass, `go vet`
**Commit:** `feat(tui): implement daemon HTTP client with REST methods`

## Task 3: HTTP Client — SSE Subscription (Feature 2.2b)
**Files:** `tui/internal/client/client.go`, `tui/internal/client/client_test.go`, `tui/internal/client/sse.go`
**Steps:**
- [ ] Write test: SSE mock server that sends 3 events, verify channel receives them
- [ ] Implement `SubscribeEvents(ctx context.Context) (<-chan SseEvent, error)` in HTTPClient
- [ ] Implement SSE line parser (handle `event:`, `data:`, empty line flush)
- [ ] Use separate http.Client with no timeout for SSE
- [ ] Verify context cancellation stops the stream cleanly
- [ ] Update feature-list.md: 2.2 → 🟢
**Commit:** `feat(tui): add SSE event subscription to daemon client`

## Task 4: Common Styles & Constants (Feature 2.3a)
**Files:** `tui/internal/views/common.go`, `tui/internal/views/common_test.go`
**Steps:**
- [ ] Define color constants: Green (#00ff00), Yellow (#ffff00), Red (#ff0000), Cyan (#00ffff), Gray (#888888)
- [ ] Define lipgloss styles: TabActive, TabInactive, CardStyle (rounded border), StatusBar, TitleStyle
- [ ] Define constants: RefreshInterval (5s), DefaultDaemonURL, TabNames
- [ ] Write tests: verify style rendering doesn't panic, status color mapping works
- [ ] Create helper functions: StatusColor(status string) lipgloss.Style, FormatTimestamp, FormatDuration, Truncate(s string, max int)
**Commit:** `feat(tui): add shared styles, constants, and view helpers`

## Task 5: App Shell — Model & Init (Feature 2.3b)
**Files:** `tui/internal/app/app.go`, `tui/internal/app/app_test.go`
**Steps:**
- [ ] Write test: model creation with mock client, verify initial state (tab=0, loading=true)
- [ ] Define `Model` struct: activeTab, workspaces, selectedWorkspace, per-tab data (git, tasks, processes, env, activity), loading/error maps, showHelp, showDetail, detailContent, lastRefresh, daemonOnline, client
- [ ] Implement `NewModel(client DaemonClient) Model`
- [ ] Implement `Init() tea.Cmd` — return batch of: fetchHealth, fetchWorkspaces, tick(5s)
- [ ] Define tea.Msg types: HealthMsg, WorkspacesMsg, GitMsg, TasksMsg, ProcessesMsg, EnvMsg, ActivityMsg, TickMsg, SseMsg, ErrMsg
- [ ] Verify tests pass
**Commit:** `feat(tui): implement app model with init and message types`

## Task 6: App Shell — Update (Key Handling + Data Fetch) (Feature 2.3c)
**Files:** `tui/internal/app/app.go`, `tui/internal/app/app_test.go`
**Steps:**
- [ ] Write tests: tab switching (Tab, Shift+Tab, 1-4), quit (q), refresh (r), help toggle (?), esc closes detail
- [ ] Implement `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
  - KeyMsg: tab switching, navigation (j/k/arrows), Enter, Esc, q, r, ?
  - HealthMsg: update daemonOnline, uptime
  - WorkspacesMsg: store workspaces, select first, trigger per-workspace data fetch
  - GitMsg, TasksMsg, ProcessesMsg, EnvMsg, ActivityMsg: store data, clear loading
  - TickMsg: re-fetch all data, schedule next tick
  - SseMsg: trigger targeted refresh based on event type
  - WindowSizeMsg: store terminal dimensions
  - ErrMsg: set error state
- [ ] Implement fetch commands (fetchGit, fetchTasks, etc.) as tea.Cmd functions
- [ ] Verify all tests pass
**Commit:** `feat(tui): implement app update loop with key handling and data fetching`

## Task 7: App Shell — View (Tab Bar + Status Bar + Routing) (Feature 2.3d)
**Files:** `tui/internal/app/app.go`, `tui/internal/app/app_test.go`
**Steps:**
- [ ] Write test: View() returns string containing tab names, status bar
- [ ] Implement `View() string`:
  - Tab bar: render 4 tabs with active highlighting
  - Content area: route to appropriate tab view based on activeTab
  - Status bar: daemon status (green/red dot), last refresh time, workspace name, tab name
  - Help overlay: show key bindings when showHelp=true
  - Loading state: spinner when data loading
- [ ] Verify tests pass
- [ ] Update feature-list.md: 2.3 → 🟢
**Commit:** `feat(tui): implement app view with tab bar, status bar, and routing`

## Task 8: Dashboard Tab View (Feature 2.4)
**Files:** `tui/internal/views/dashboard.go`, `tui/internal/views/dashboard_test.go`
**Steps:**
- [ ] Write test: render dashboard with mock data, verify cards contain expected content
- [ ] Implement `RenderDashboard(data DashboardData, width, height int) string`:
  - Top row: workspace path, daemon uptime, connection dot
  - Tasks Card: total, running (yellow), completed (green), failed (red)
  - Git Card: branch, sync icon, last commit
  - Processes Card: count, CPU/mem totals, stalled warning
  - Env Card: tools available/total, missing highlighted
  - Bottom: last 5 activity events (one-line each)
- [ ] Define `DashboardData` struct aggregating all tab data
- [ ] Test empty/nil data gracefully handled
- [ ] Update feature-list.md: 2.4 → 🟢
**Commit:** `feat(tui): implement dashboard tab with summary cards`

## Task 9: Tasks Tab View (Feature 2.5)
**Files:** `tui/internal/views/tasks.go`, `tui/internal/views/tasks_test.go`
**Steps:**
- [ ] Write test: render tasks table with mock data, verify columns and color-coded status
- [ ] Implement `RenderTasks(tasks []TaskInfo, selected int, width, height int) string`:
  - Table with columns: Name, Status, Duration, Model, Teams, Started, Branch
  - Status color-coding: done=green, in_progress=yellow, failed=red, pending=gray
  - Selected row highlighting
  - Empty state: "No tasks found"
- [ ] Implement `RenderTaskDetail(task TaskInfo, width, height int) string` for Enter detail view
- [ ] Test empty task list, single task, multiple tasks
- [ ] Update feature-list.md: 2.5 → 🟢
**Commit:** `feat(tui): implement tasks tab with table and detail view`

## Task 10: Git Tab View (Feature 2.6)
**Files:** `tui/internal/views/git.go`, `tui/internal/views/git_test.go`
**Steps:**
- [ ] Write test: render git view with mock GitStatus, verify branch/sync badges/commits
- [ ] Implement `RenderGit(git *GitStatus, selected int, width, height int) string`:
  - Top: branch name, sync status badge (clean/dirty/ahead-behind)
  - Dirty: list uncommitted files in red
  - Middle: recent commits table (hash, message, author, time)
  - Bottom: last push time, remote info
- [ ] Implement `RenderCommitDetail(commit CommitInfo, width, height int) string`
- [ ] Test clean repo, dirty repo, nil git status
- [ ] Update feature-list.md: 2.6 → 🟢
**Commit:** `feat(tui): implement git tab with sync status and commit list`

## Task 11: Activity Tab View (Feature 2.7)
**Files:** `tui/internal/views/activity.go`, `tui/internal/views/activity_test.go`
**Steps:**
- [ ] Write test: render activity timeline with mixed event types, verify icons and colors
- [ ] Implement `RenderActivity(events []ActivityEvent, selected int, width, height int) string`:
  - Vertical timeline, most recent first
  - Each: timestamp | type icon | description
  - Icons: commit=blue, task=yellow/green/red, process=cyan, env=gray
  - Scrollable (respect selected for viewport offset)
- [ ] Test empty events, single event, mixed types
- [ ] Update feature-list.md: 2.7 → 🟢
**Commit:** `feat(tui): implement activity tab with color-coded timeline`

## Task 12: SSE Real-Time Updates Integration (Feature 2.8)
**Files:** `tui/internal/app/app.go`, `tui/internal/app/app_test.go`
**Steps:**
- [ ] Write test: SseMsg triggers appropriate data refresh
- [ ] Implement SSE subscription as background tea.Cmd in Init
- [ ] Map SSE event types to targeted refreshes (git_commit → fetchGit, task_* → fetchTasks, etc.)
- [ ] Handle SSE reconnection on disconnect (retry after 3s)
- [ ] Update feature-list.md: 2.8 → 🟢
**Commit:** `feat(tui): integrate SSE for real-time dashboard updates`

## Task 13: Detail Views & Navigation Polish (Feature 2.9)
**Files:** `tui/internal/app/app.go`, `tui/internal/views/tasks.go`, `tui/internal/views/git.go`
**Steps:**
- [ ] Write test: Enter on task row opens detail, Esc closes
- [ ] Wire detail views into app Update/View: track selected indices per tab
- [ ] Implement j/k scroll within tasks table, git commits, activity timeline
- [ ] Implement detail overlay rendering in View()
- [ ] Test navigation: up/down boundaries, enter/esc transitions
- [ ] Update feature-list.md: 2.9 → 🟢
**Commit:** `feat(tui): add detail views and keyboard navigation polish`

## Task 14: Integration Tests (Feature 2.10)
**Files:** `tui/tests/integration/client_integration_test.go`, `tui/tests/integration/app_integration_test.go`
**Steps:**
- [ ] Create `tui/tests/integration/` directory
- [ ] Write client integration test: httptest server with full realistic responses, test all 7 endpoints
- [ ] Write app integration test: create model with real HTTPClient (mock server), simulate Init → Update cycle, verify state
- [ ] Write view integration test: render each view with real-shaped data, verify no panics
- [ ] Verify: `go test ./tests/integration/...`
- [ ] Update feature-list.md: 2.10 → 🟢
**Commit:** `test(tui): add integration tests for client and app`

## Task 15: E2E Tests (Feature 2.11)
**Files:** `tui/tests/e2e/e2e_test.go`
**Steps:**
- [ ] Create `tui/tests/e2e/` directory
- [ ] Write E2E test: start mock daemon (httptest), create full app model, run Init+Update+View cycle
- [ ] Verify dashboard renders with mock data
- [ ] Verify tab switching works
- [ ] Verify data appears in each tab view
- [ ] Test error state when daemon is unreachable
- [ ] Verify: `go test ./tests/e2e/...`
- [ ] Update feature-list.md: 2.11 → 🟢
**Commit:** `test(tui): add E2E tests for full TUI pipeline`

## Task 16: Final Cleanup & main.go Polish
**Files:** `tui/cmd/clawmon-tui/main.go`
**Steps:**
- [ ] Polish main.go: proper flag parsing, error output, graceful shutdown
- [ ] Verify `make build-tui` works
- [ ] Verify `make test-tui` passes all tests
- [ ] Verify `make lint-tui` passes (go vet)
- [ ] Run full `go test ./... -count=1 -v` for final verification
- [ ] Ensure all feature-list.md Phase 2 items are 🟢
**Commit:** `chore(tui): polish entry point and verify all builds pass`
