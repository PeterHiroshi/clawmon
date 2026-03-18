use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Standard API response wrapper.
#[derive(Debug, Serialize, Deserialize)]
pub struct ApiResponse<T: Serialize> {
    pub data: T,
    pub timestamp: DateTime<Utc>,
}

impl<T: Serialize> ApiResponse<T> {
    /// Create a new API response with the current timestamp.
    pub fn new(data: T) -> Self {
        Self {
            data,
            timestamp: Utc::now(),
        }
    }
}

/// Health check response.
#[derive(Debug, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub uptime_seconds: u64,
    pub version: String,
}

/// Workspace summary info.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkspaceInfo {
    pub id: String,
    pub path: String,
    pub name: String,
}

/// Git sync status for a workspace.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GitStatus {
    pub branch: String,
    pub is_clean: bool,
    pub uncommitted_files: Vec<String>,
    pub ahead: usize,
    pub behind: usize,
    pub last_commit: Option<CommitInfo>,
    pub last_push_time: Option<DateTime<Utc>>,
    pub recent_commits: Vec<CommitInfo>,
}

/// Single commit details.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommitInfo {
    pub hash: String,
    pub short_hash: String,
    pub message: String,
    pub author: String,
    pub time: DateTime<Utc>,
}

/// Task status enum.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TaskStatus {
    Pending,
    InProgress,
    Done,
    Failed,
    Unknown,
}

/// Task information from .forge-task/ directories.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaskInfo {
    pub name: String,
    pub status: TaskStatus,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
    pub model: Option<String>,
    pub agent_teams: Option<bool>,
    pub current_step: Option<String>,
    pub progress_percent: Option<u8>,
    pub exit_code: Option<i32>,
    pub effective_model: Option<String>,
    pub duration_seconds: Option<u64>,
    pub task_dir: String,
}

/// Process state enum.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ProcessState {
    Running,
    Sleeping,
    Zombie,
    Stopped,
    Unknown,
}

/// Claude Code process info.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProcessInfo {
    pub pid: u32,
    pub workdir: Option<String>,
    pub cpu_percent: Option<f64>,
    pub memory_kb: Option<u64>,
    pub uptime_seconds: Option<u64>,
    pub state: ProcessState,
    pub stalled: bool,
    pub command: String,
}

/// Environment tool check result.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvCheck {
    pub tool: String,
    pub installed: bool,
    pub version: Option<String>,
    pub path: Option<String>,
}

/// Hook/plugin check result.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HookCheck {
    pub name: String,
    pub installed: bool,
    pub details: Option<String>,
}

/// Full environment health report.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvHealth {
    pub tools: Vec<EnvCheck>,
    pub hooks: Vec<HookCheck>,
}

/// Activity event types.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ActivityType {
    GitCommit,
    GitPush,
    TaskStarted,
    TaskCompleted,
    TaskFailed,
    ProcessStarted,
    ProcessStopped,
    FileChanged,
}

/// Single activity event in the timeline.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActivityEvent {
    pub timestamp: DateTime<Utc>,
    pub event_type: ActivityType,
    pub workspace_id: String,
    pub description: String,
}

/// SSE event data.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SseEvent {
    pub event_type: String,
    pub workspace_id: Option<String>,
    pub data: serde_json::Value,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_api_response_serialization() {
        let response = ApiResponse::new(HealthResponse {
            status: "ok".to_string(),
            uptime_seconds: 42,
            version: "0.1.0".to_string(),
        });
        let json = serde_json::to_string(&response).unwrap();
        assert!(json.contains("\"status\":\"ok\""));
        assert!(json.contains("\"uptime_seconds\":42"));
        assert!(json.contains("\"timestamp\""));
    }

    #[test]
    fn test_task_status_serialization() {
        let status = TaskStatus::InProgress;
        let json = serde_json::to_string(&status).unwrap();
        assert_eq!(json, "\"in_progress\"");

        let deserialized: TaskStatus = serde_json::from_str(&json).unwrap();
        assert_eq!(deserialized, TaskStatus::InProgress);
    }

    #[test]
    fn test_process_state_serialization() {
        let state = ProcessState::Running;
        let json = serde_json::to_string(&state).unwrap();
        assert_eq!(json, "\"running\"");
    }

    #[test]
    fn test_activity_type_serialization() {
        let event_type = ActivityType::GitCommit;
        let json = serde_json::to_string(&event_type).unwrap();
        assert_eq!(json, "\"git_commit\"");
    }

    #[test]
    fn test_workspace_info_roundtrip() {
        let info = WorkspaceInfo {
            id: "test-ws".to_string(),
            path: "/tmp/workspace".to_string(),
            name: "Test Workspace".to_string(),
        };
        let json = serde_json::to_string(&info).unwrap();
        let deserialized: WorkspaceInfo = serde_json::from_str(&json).unwrap();
        assert_eq!(deserialized.id, "test-ws");
        assert_eq!(deserialized.path, "/tmp/workspace");
    }

    #[test]
    fn test_git_status_serialization() {
        let status = GitStatus {
            branch: "main".to_string(),
            is_clean: true,
            uncommitted_files: vec![],
            ahead: 0,
            behind: 0,
            last_commit: None,
            last_push_time: None,
            recent_commits: vec![],
        };
        let json = serde_json::to_string(&status).unwrap();
        assert!(json.contains("\"is_clean\":true"));
        assert!(json.contains("\"branch\":\"main\""));
    }

    #[test]
    fn test_env_health_serialization() {
        let health = EnvHealth {
            tools: vec![EnvCheck {
                tool: "git".to_string(),
                installed: true,
                version: Some("2.43.0".to_string()),
                path: Some("/usr/bin/git".to_string()),
            }],
            hooks: vec![HookCheck {
                name: "notify-forge".to_string(),
                installed: false,
                details: None,
            }],
        };
        let json = serde_json::to_string(&health).unwrap();
        assert!(json.contains("\"tool\":\"git\""));
        assert!(json.contains("\"installed\":true"));
    }
}
