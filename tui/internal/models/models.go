// Package models defines data types mirroring the clawmon daemon API responses.
package models

import "time"

// ApiResponse is the standard wrapper for all daemon API responses.
type ApiResponse[T any] struct {
	Data      T      `json:"data"`
	Timestamp string `json:"timestamp"`
}

// HealthResponse is returned by GET /api/v1/health.
type HealthResponse struct {
	Status        string `json:"status"`
	UptimeSeconds uint64 `json:"uptime_seconds"`
	Version       string `json:"version"`
}

// WorkspaceInfo describes a monitored workspace.
type WorkspaceInfo struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
}

// GitStatus represents git sync state for a workspace.
type GitStatus struct {
	Branch           string       `json:"branch"`
	IsClean          bool         `json:"is_clean"`
	UncommittedFiles []string     `json:"uncommitted_files"`
	Ahead            int          `json:"ahead"`
	Behind           int          `json:"behind"`
	LastCommit       *CommitInfo  `json:"last_commit"`
	LastPushTime     *time.Time   `json:"last_push_time"`
	RecentCommits    []CommitInfo `json:"recent_commits"`
}

// CommitInfo describes a single git commit.
type CommitInfo struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Time      time.Time `json:"time"`
}

// TaskStatus represents the state of a task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusUnknown    TaskStatus = "unknown"
)

// TaskInfo describes a forge-dispatch task.
type TaskInfo struct {
	Name            string     `json:"name"`
	Status          TaskStatus `json:"status"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	Model           *string    `json:"model"`
	AgentTeams      *bool      `json:"agent_teams"`
	CurrentStep     *string    `json:"current_step"`
	ProgressPercent *uint8     `json:"progress_percent"`
	ExitCode        *int       `json:"exit_code"`
	EffectiveModel  *string    `json:"effective_model"`
	DurationSeconds *uint64    `json:"duration_seconds"`
	TaskDir         string     `json:"task_dir"`
}

// ProcessState represents a system process state.
type ProcessState string

const (
	ProcessStateRunning  ProcessState = "running"
	ProcessStateSleeping ProcessState = "sleeping"
	ProcessStateZombie   ProcessState = "zombie"
	ProcessStateStopped  ProcessState = "stopped"
	ProcessStateUnknown  ProcessState = "unknown"
)

// ProcessInfo describes a Claude Code process.
type ProcessInfo struct {
	PID           uint32       `json:"pid"`
	Workdir       *string      `json:"workdir"`
	CPUPercent    *float64     `json:"cpu_percent"`
	MemoryKB      *uint64      `json:"memory_kb"`
	UptimeSeconds *uint64      `json:"uptime_seconds"`
	State         ProcessState `json:"state"`
	Stalled       bool         `json:"stalled"`
	Command       string       `json:"command"`
}

// EnvCheck describes a tool availability check.
type EnvCheck struct {
	Tool      string  `json:"tool"`
	Installed bool    `json:"installed"`
	Version   *string `json:"version"`
	Path      *string `json:"path"`
}

// HookCheck describes a hook/plugin availability check.
type HookCheck struct {
	Name      string  `json:"name"`
	Installed bool    `json:"installed"`
	Details   *string `json:"details"`
}

// EnvHealth is the full environment health report.
type EnvHealth struct {
	Tools []EnvCheck  `json:"tools"`
	Hooks []HookCheck `json:"hooks"`
}

// ActivityType categorizes activity events.
type ActivityType string

const (
	ActivityGitCommit      ActivityType = "git_commit"
	ActivityGitPush        ActivityType = "git_push"
	ActivityTaskStarted    ActivityType = "task_started"
	ActivityTaskCompleted  ActivityType = "task_completed"
	ActivityTaskFailed     ActivityType = "task_failed"
	ActivityProcessStarted ActivityType = "process_started"
	ActivityProcessStopped ActivityType = "process_stopped"
	ActivityFileChanged    ActivityType = "file_changed"
)

// ActivityEvent is a single item in the activity timeline.
type ActivityEvent struct {
	Timestamp   time.Time    `json:"timestamp"`
	EventType   ActivityType `json:"event_type"`
	WorkspaceID string       `json:"workspace_id"`
	Description string       `json:"description"`
}

// SystemResources contains host system resource metrics.
type SystemResources struct {
	CPU           CpuInfo    `json:"cpu"`
	Memory        MemoryInfo `json:"memory"`
	Swap          SwapInfo   `json:"swap"`
	Disks         []DiskInfo `json:"disks"`
	GPUs          []GpuInfo  `json:"gpus"`
	UptimeSeconds uint64     `json:"uptime_seconds"`
	CollectedAt   string     `json:"collected_at"`
}

// CpuInfo contains CPU identification and usage metrics.
type CpuInfo struct {
	Model        string    `json:"model"`
	CoreCount    int       `json:"core_count"`
	UsagePercent float32   `json:"usage_percent"`
	PerCoreUsage []float32 `json:"per_core_usage"`
	LoadAvg1m    float64   `json:"load_avg_1m"`
	LoadAvg5m    float64   `json:"load_avg_5m"`
	LoadAvg15m   float64   `json:"load_avg_15m"`
}

// MemoryInfo contains RAM usage metrics.
type MemoryInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float32 `json:"usage_percent"`
}

// SwapInfo contains swap usage metrics.
type SwapInfo struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float32 `json:"usage_percent"`
}

// DiskInfo contains filesystem usage metrics.
type DiskInfo struct {
	MountPoint     string  `json:"mount_point"`
	Filesystem     string  `json:"filesystem"`
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float32 `json:"usage_percent"`
}

// GpuInfo contains GPU metrics (from nvidia-smi).
type GpuInfo struct {
	Name               string  `json:"name"`
	MemoryUsedMB       uint64  `json:"memory_used_mb"`
	MemoryTotalMB      uint64  `json:"memory_total_mb"`
	MemoryUsagePercent float32 `json:"memory_usage_percent"`
	UtilizationPercent float32 `json:"utilization_percent"`
	TemperatureCelsius float32 `json:"temperature_celsius"`
}

// SseEvent represents a server-sent event from the daemon.
type SseEvent struct {
	EventType   string      `json:"event_type"`
	WorkspaceID *string     `json:"workspace_id"`
	Data        interface{} `json:"data"`
}
