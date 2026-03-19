use std::fs;
use std::path::PathBuf;
use std::time::Duration;

use clawmon_daemon::api::handlers::AppState;
use clawmon_daemon::api::routes::build_router;
use clawmon_daemon::config::Config;

use axum::body::Body;
use axum::http::{Request, StatusCode};
use tempfile::TempDir;
use tower::ServiceExt;

fn test_config_with_projects(project_dirs: Vec<PathBuf>) -> Config {
    Config {
        port: 0,
        bind: "127.0.0.1".to_string(),
        mode: clawmon_daemon::config::DaemonMode::Standalone,
        workspace_path: PathBuf::from("/tmp/test-workspace"),
        poll_interval: Duration::from_secs(30),
        project_dirs,
    }
}

fn setup_git_repo(dir: &std::path::Path) {
    let repo = git2::Repository::init(dir).unwrap();
    let mut config = repo.config().unwrap();
    config.set_str("user.name", "Test").unwrap();
    config.set_str("user.email", "test@test.com").unwrap();

    // Make initial commit
    let file_path = dir.join("README.md");
    fs::write(&file_path, "# Test").unwrap();
    let mut index = repo.index().unwrap();
    index.add_path(std::path::Path::new("README.md")).unwrap();
    index.write().unwrap();
    let tree_id = index.write_tree().unwrap();
    let tree = repo.find_tree(tree_id).unwrap();
    let sig = repo.signature().unwrap();
    repo.commit(Some("HEAD"), &sig, &sig, "initial commit", &tree, &[])
        .unwrap();
}

fn setup_forge_task(dir: &std::path::Path) {
    let forge_dir = dir.join(".forge-task");
    fs::create_dir_all(&forge_dir).unwrap();
    fs::write(
        forge_dir.join("task-spec.json"),
        r#"{"name": "test-task", "status": "in_progress", "model": "opus"}"#,
    )
    .unwrap();
}

#[tokio::test]
async fn test_health_endpoint() {
    let config = test_config_with_projects(vec![]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/health")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    assert_eq!(json["data"]["status"], "ok");
    assert_eq!(json["data"]["version"], "0.1.0");
    assert!(json["timestamp"].is_string());
}

#[tokio::test]
async fn test_workspaces_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project_a = tmp.path().join("project-a");
    fs::create_dir_all(&project_a).unwrap();

    let config = test_config_with_projects(vec![project_a]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    let workspaces = json["data"].as_array().unwrap();
    assert_eq!(workspaces.len(), 1);
    assert_eq!(workspaces[0]["name"], "project-a");
}

#[tokio::test]
async fn test_workspace_git_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project = tmp.path().join("myproject");
    fs::create_dir_all(&project).unwrap();
    setup_git_repo(&project);

    let config = test_config_with_projects(vec![project]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/myproject/git")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    assert!(json["data"]["branch"].is_string());
    assert!(json["data"]["is_clean"].is_boolean());
    assert!(json["data"]["recent_commits"].is_array());
}

#[tokio::test]
async fn test_workspace_tasks_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project = tmp.path().join("taskproject");
    fs::create_dir_all(&project).unwrap();
    setup_forge_task(&project);

    let config = test_config_with_projects(vec![project]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/taskproject/tasks")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    let tasks = json["data"].as_array().unwrap();
    assert_eq!(tasks.len(), 1);
    assert_eq!(tasks[0]["name"], "test-task");
    assert_eq!(tasks[0]["status"], "in_progress");
}

#[tokio::test]
async fn test_workspace_not_found() {
    let config = test_config_with_projects(vec![]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/nonexistent/git")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::NOT_FOUND);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    assert!(
        json["error"]
            .as_str()
            .unwrap()
            .contains("workspace not found")
    );
}

#[tokio::test]
async fn test_workspace_env_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project = tmp.path().join("envproject");
    fs::create_dir_all(&project).unwrap();

    let config = test_config_with_projects(vec![project]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/envproject/env")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    assert!(json["data"]["tools"].is_array());
    assert!(json["data"]["hooks"].is_array());
}

#[tokio::test]
async fn test_workspace_processes_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project = tmp.path().join("procproject");
    fs::create_dir_all(&project).unwrap();

    let config = test_config_with_projects(vec![project]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/procproject/processes")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    assert!(json["data"].is_array());
}

#[tokio::test]
async fn test_workspace_system_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project = tmp.path().join("sysproject");
    fs::create_dir_all(&project).unwrap();

    let config = test_config_with_projects(vec![project]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/sysproject/system")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    assert!(json["data"]["cpu"]["core_count"].is_number());
    assert!(json["data"]["memory"]["total_bytes"].is_number());
    assert!(json["data"]["disks"].is_array());
    assert!(json["data"]["gpus"].is_array());
    assert!(json["data"]["uptime_seconds"].is_number());
    assert!(json["data"]["collected_at"].is_string());
    assert!(json["timestamp"].is_string());
}

#[tokio::test]
async fn test_workspace_activity_endpoint() {
    let tmp = TempDir::new().unwrap();
    let project = tmp.path().join("actproject");
    fs::create_dir_all(&project).unwrap();
    setup_git_repo(&project);
    setup_forge_task(&project);

    let config = test_config_with_projects(vec![project]);
    let state = AppState::new(config);
    let app = build_router(state);

    let response = app
        .oneshot(
            Request::builder()
                .uri("/api/v1/workspaces/actproject/activity")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    let body = axum::body::to_bytes(response.into_body(), usize::MAX)
        .await
        .unwrap();
    let json: serde_json::Value = serde_json::from_slice(&body).unwrap();
    let events = json["data"].as_array().unwrap();
    // Should have at least the git commit event
    assert!(!events.is_empty());
}
