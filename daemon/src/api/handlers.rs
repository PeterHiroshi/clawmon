use std::collections::HashMap;
use std::convert::Infallible;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Instant;

use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{IntoResponse, Json, Response};
use tokio_stream::StreamExt;
use tokio_stream::wrappers::BroadcastStream;

use crate::collectors::env::EnvCollector;
use crate::collectors::git::GitCollector;
use crate::collectors::process::ProcessCollector;
use crate::collectors::tasks::TaskCollector;
use crate::config::{Config, VERSION};
use crate::events::EventBus;
use crate::models::{ActivityEvent, ApiResponse, HealthResponse, WorkspaceInfo};

/// Shared application state.
#[derive(Clone)]
pub struct AppState {
    /// Daemon configuration.
    pub config: Arc<Config>,
    /// Daemon start time for uptime calculation.
    pub start_time: Instant,
    /// Workspace registry: id -> path.
    pub workspaces: Arc<HashMap<String, WorkspaceInfo>>,
    /// Event bus for SSE.
    pub event_bus: EventBus,
}

impl AppState {
    /// Create new application state from config.
    pub fn new(config: Config) -> Self {
        let workspaces = build_workspace_map(&config);
        Self {
            config: Arc::new(config),
            start_time: Instant::now(),
            workspaces: Arc::new(workspaces),
            event_bus: EventBus::new(),
        }
    }
}

/// Build workspace map from config project dirs.
fn build_workspace_map(config: &Config) -> HashMap<String, WorkspaceInfo> {
    let mut map = HashMap::new();

    for project_dir in &config.project_dirs {
        let name = project_dir
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_else(|| "unknown".to_string());
        let id = slugify(&name);
        map.insert(
            id.clone(),
            WorkspaceInfo {
                id,
                path: project_dir.to_string_lossy().to_string(),
                name,
            },
        );
    }

    map
}

/// Convert a name to a URL-safe slug.
fn slugify(name: &str) -> String {
    name.to_lowercase()
        .chars()
        .map(|c| {
            if c.is_alphanumeric() || c == '-' {
                c
            } else {
                '-'
            }
        })
        .collect()
}

/// Helper: return 404 if workspace not found.
#[allow(clippy::result_large_err)]
fn find_workspace(state: &AppState, id: &str) -> std::result::Result<WorkspaceInfo, Response> {
    state.workspaces.get(id).cloned().ok_or_else(|| {
        let body = serde_json::json!({
            "error": format!("workspace not found: {}", id),
            "timestamp": chrono::Utc::now().to_rfc3339(),
        });
        (StatusCode::NOT_FOUND, Json(body)).into_response()
    })
}

/// GET /api/v1/health
pub async fn health(State(state): State<AppState>) -> Json<ApiResponse<HealthResponse>> {
    let uptime = state.start_time.elapsed().as_secs();
    Json(ApiResponse::new(HealthResponse {
        status: "ok".to_string(),
        uptime_seconds: uptime,
        version: VERSION.to_string(),
    }))
}

/// GET /api/v1/workspaces
pub async fn list_workspaces(
    State(state): State<AppState>,
) -> Json<ApiResponse<Vec<WorkspaceInfo>>> {
    let workspaces: Vec<WorkspaceInfo> = state.workspaces.values().cloned().collect();
    Json(ApiResponse::new(workspaces))
}

/// GET /api/v1/workspaces/{id}/git
pub async fn workspace_git(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<impl IntoResponse, Response> {
    let ws = find_workspace(&state, &id)?;
    let collector = GitCollector::new(std::path::Path::new(&ws.path));

    match collector.collect() {
        Ok(git_status) => Ok(Json(ApiResponse::new(git_status))),
        Err(err) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({
                "error": format!("git collector failed: {}", err),
                "timestamp": chrono::Utc::now().to_rfc3339(),
            })),
        )
            .into_response()),
    }
}

/// GET /api/v1/workspaces/{id}/tasks
pub async fn workspace_tasks(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<impl IntoResponse, Response> {
    let ws = find_workspace(&state, &id)?;
    let collector = TaskCollector::new(&[PathBuf::from(&ws.path)]);

    match collector.collect() {
        Ok(tasks) => Ok(Json(ApiResponse::new(tasks))),
        Err(err) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({
                "error": format!("task collector failed: {}", err),
                "timestamp": chrono::Utc::now().to_rfc3339(),
            })),
        )
            .into_response()),
    }
}

/// GET /api/v1/workspaces/{id}/processes
pub async fn workspace_processes(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<impl IntoResponse, Response> {
    let _ws = find_workspace(&state, &id)?;
    let collector = ProcessCollector::new();

    match collector.collect() {
        Ok(processes) => Ok(Json(ApiResponse::new(processes))),
        Err(err) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({
                "error": format!("process collector failed: {}", err),
                "timestamp": chrono::Utc::now().to_rfc3339(),
            })),
        )
            .into_response()),
    }
}

/// GET /api/v1/workspaces/{id}/env
pub async fn workspace_env(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<impl IntoResponse, Response> {
    let _ws = find_workspace(&state, &id)?;
    let collector = EnvCollector::new();

    match collector.collect() {
        Ok(env_health) => Ok(Json(ApiResponse::new(env_health))),
        Err(err) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({
                "error": format!("env collector failed: {}", err),
                "timestamp": chrono::Utc::now().to_rfc3339(),
            })),
        )
            .into_response()),
    }
}

/// GET /api/v1/workspaces/{id}/activity
pub async fn workspace_activity(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> std::result::Result<impl IntoResponse, Response> {
    let _ws = find_workspace(&state, &id)?;

    // Activity is a merged timeline — for now return recent git commits as activity events
    let ws = state
        .workspaces
        .get(&id)
        .cloned()
        .unwrap_or_else(|| WorkspaceInfo {
            id: id.clone(),
            path: String::new(),
            name: String::new(),
        });

    let mut events: Vec<ActivityEvent> = Vec::new();

    // Pull git commits as activity
    let git_collector = GitCollector::new(std::path::Path::new(&ws.path));
    if let Ok(git_status) = git_collector.collect() {
        for commit in &git_status.recent_commits {
            events.push(ActivityEvent {
                timestamp: commit.time,
                event_type: crate::models::ActivityType::GitCommit,
                workspace_id: id.clone(),
                description: format!("{}: {}", commit.short_hash, commit.message),
            });
        }
    }

    // Pull task events as activity
    let task_collector = TaskCollector::new(&[PathBuf::from(&ws.path)]);
    if let Ok(tasks) = task_collector.collect() {
        for task in &tasks {
            if let Some(started) = task.started_at {
                events.push(ActivityEvent {
                    timestamp: started,
                    event_type: crate::models::ActivityType::TaskStarted,
                    workspace_id: id.clone(),
                    description: format!("Task started: {}", task.name),
                });
            }
            if let Some(completed) = task.completed_at {
                let event_type = if task.status == crate::models::TaskStatus::Failed {
                    crate::models::ActivityType::TaskFailed
                } else {
                    crate::models::ActivityType::TaskCompleted
                };
                events.push(ActivityEvent {
                    timestamp: completed,
                    event_type,
                    workspace_id: id.clone(),
                    description: format!("Task completed: {}", task.name),
                });
            }
        }
    }

    // Sort by timestamp descending
    events.sort_by(|a, b| b.timestamp.cmp(&a.timestamp));

    Ok(Json(ApiResponse::new(events)))
}

/// GET /api/v1/events — Server-Sent Events stream.
pub async fn sse_events(
    State(state): State<AppState>,
) -> Sse<impl tokio_stream::Stream<Item = std::result::Result<Event, Infallible>>> {
    let receiver = state.event_bus.subscribe();
    let stream = BroadcastStream::new(receiver).filter_map(|result| match result {
        Ok(sse_event) => {
            let data = serde_json::to_string(&sse_event).unwrap_or_default();
            Some(Ok(Event::default().event(sse_event.event_type).data(data)))
        }
        Err(_) => None,
    });

    Sse::new(stream).keep_alive(KeepAlive::default())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_config() -> Config {
        Config {
            port: 0,
            workspace_path: PathBuf::from("/tmp/test"),
            poll_interval: std::time::Duration::from_secs(30),
            project_dirs: vec![],
        }
    }

    #[test]
    fn test_slugify() {
        assert_eq!(slugify("My Project"), "my-project");
        assert_eq!(slugify("hello_world"), "hello-world");
        assert_eq!(slugify("already-slug"), "already-slug");
    }

    #[test]
    fn test_build_workspace_map_empty() {
        let config = test_config();
        let map = build_workspace_map(&config);
        assert!(map.is_empty());
    }

    #[test]
    fn test_build_workspace_map_with_dirs() {
        let mut config = test_config();
        config.project_dirs = vec![
            PathBuf::from("/workspace/projects/alpha"),
            PathBuf::from("/workspace/projects/beta"),
        ];
        let map = build_workspace_map(&config);
        assert_eq!(map.len(), 2);
        assert!(map.contains_key("alpha"));
        assert!(map.contains_key("beta"));
    }

    #[test]
    fn test_app_state_creation() {
        let config = test_config();
        let state = AppState::new(config);
        assert!(state.workspaces.is_empty());
    }

    #[test]
    fn test_find_workspace_not_found() {
        let config = test_config();
        let state = AppState::new(config);
        let result = find_workspace(&state, "nonexistent");
        assert!(result.is_err());
    }
}
