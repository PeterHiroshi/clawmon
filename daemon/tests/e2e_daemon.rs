use std::fs;
use std::net::TcpListener;
use std::path::PathBuf;
use std::time::Duration;

use clawmon_daemon::api::handlers::AppState;
use clawmon_daemon::api::routes::build_router;
use clawmon_daemon::config::Config;
use tempfile::TempDir;

/// Find a free port for testing.
fn free_port() -> u16 {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    listener.local_addr().unwrap().port()
}

/// Start the daemon on a free port and return the base URL.
async fn start_daemon(project_dirs: Vec<PathBuf>) -> (String, tokio::task::JoinHandle<()>) {
    let port = free_port();
    let config = Config {
        port,
        workspace_path: PathBuf::from("/tmp/e2e-test"),
        poll_interval: Duration::from_secs(300),
        project_dirs,
    };

    let state = AppState::new(config);
    let app = build_router(state);
    let base_url = format!("http://127.0.0.1:{}", port);

    let listener = tokio::net::TcpListener::bind(format!("127.0.0.1:{}", port))
        .await
        .unwrap();

    let handle = tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });

    // Give server time to start
    tokio::time::sleep(Duration::from_millis(100)).await;

    (base_url, handle)
}

fn setup_git_repo(dir: &std::path::Path) {
    let repo = git2::Repository::init(dir).unwrap();
    let mut config = repo.config().unwrap();
    config.set_str("user.name", "E2E Test").unwrap();
    config.set_str("user.email", "e2e@test.com").unwrap();

    let file_path = dir.join("file.txt");
    fs::write(&file_path, "content").unwrap();
    let mut index = repo.index().unwrap();
    index
        .add_path(std::path::Path::new("file.txt"))
        .unwrap();
    index.write().unwrap();
    let tree_id = index.write_tree().unwrap();
    let tree = repo.find_tree(tree_id).unwrap();
    let sig = repo.signature().unwrap();
    repo.commit(Some("HEAD"), &sig, &sig, "e2e test commit", &tree, &[])
        .unwrap();
}

#[tokio::test]
async fn e2e_health_check() {
    let (base_url, handle) = start_daemon(vec![]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/health", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    let json: serde_json::Value = resp.json().await.unwrap();
    assert_eq!(json["data"]["status"], "ok");
    assert_eq!(json["data"]["version"], "0.1.0");
    assert!(json["data"]["uptime_seconds"].as_u64().is_some());
    assert!(json["timestamp"].is_string());

    handle.abort();
}

#[tokio::test]
async fn e2e_workspaces_list() {
    let tmp = TempDir::new().unwrap();
    let proj = tmp.path().join("alpha");
    fs::create_dir_all(&proj).unwrap();

    let (base_url, handle) = start_daemon(vec![proj]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/workspaces", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    let json: serde_json::Value = resp.json().await.unwrap();
    let workspaces = json["data"].as_array().unwrap();
    assert_eq!(workspaces.len(), 1);
    assert_eq!(workspaces[0]["id"], "alpha");

    handle.abort();
}

#[tokio::test]
async fn e2e_git_endpoint() {
    let tmp = TempDir::new().unwrap();
    let proj = tmp.path().join("gitrepo");
    fs::create_dir_all(&proj).unwrap();
    setup_git_repo(&proj);

    let (base_url, handle) = start_daemon(vec![proj]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/workspaces/gitrepo/git", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    let json: serde_json::Value = resp.json().await.unwrap();
    let data = &json["data"];
    assert!(data["branch"].is_string());
    assert!(data["is_clean"].is_boolean());
    assert!(data["recent_commits"].is_array());
    let commits = data["recent_commits"].as_array().unwrap();
    assert!(!commits.is_empty());
    assert_eq!(commits[0]["message"], "e2e test commit");

    handle.abort();
}

#[tokio::test]
async fn e2e_tasks_endpoint() {
    let tmp = TempDir::new().unwrap();
    let proj = tmp.path().join("taskproj");
    fs::create_dir_all(&proj).unwrap();
    let forge_dir = proj.join(".forge-task");
    fs::create_dir_all(&forge_dir).unwrap();
    fs::write(
        forge_dir.join("task-spec.json"),
        r#"{"name": "e2e-task", "status": "done", "started_at": "2026-03-18T10:00:00Z"}"#,
    )
    .unwrap();
    fs::write(
        forge_dir.join("meta.json"),
        r#"{"exit_code": 0, "duration": 60}"#,
    )
    .unwrap();

    let (base_url, handle) = start_daemon(vec![proj]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/workspaces/taskproj/tasks", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    let json: serde_json::Value = resp.json().await.unwrap();
    let tasks = json["data"].as_array().unwrap();
    assert_eq!(tasks.len(), 1);
    assert_eq!(tasks[0]["name"], "e2e-task");
    assert_eq!(tasks[0]["status"], "done");
    assert_eq!(tasks[0]["exit_code"], 0);

    handle.abort();
}

#[tokio::test]
async fn e2e_workspace_not_found() {
    let (base_url, handle) = start_daemon(vec![]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/workspaces/does-not-exist/git", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 404);
    let json: serde_json::Value = resp.json().await.unwrap();
    assert!(json["error"]
        .as_str()
        .unwrap()
        .contains("workspace not found"));

    handle.abort();
}

#[tokio::test]
async fn e2e_env_endpoint() {
    let tmp = TempDir::new().unwrap();
    let proj = tmp.path().join("envproj");
    fs::create_dir_all(&proj).unwrap();

    let (base_url, handle) = start_daemon(vec![proj]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/workspaces/envproj/env", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    let json: serde_json::Value = resp.json().await.unwrap();
    let tools = json["data"]["tools"].as_array().unwrap();
    assert!(!tools.is_empty());

    // git should be installed
    let git_tool = tools.iter().find(|t| t["tool"] == "git").unwrap();
    assert_eq!(git_tool["installed"], true);

    handle.abort();
}

#[tokio::test]
async fn e2e_activity_endpoint() {
    let tmp = TempDir::new().unwrap();
    let proj = tmp.path().join("actproj");
    fs::create_dir_all(&proj).unwrap();
    setup_git_repo(&proj);

    let (base_url, handle) = start_daemon(vec![proj]).await;
    let client = reqwest::Client::new();

    let resp = client
        .get(format!("{}/api/v1/workspaces/actproj/activity", base_url))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    let json: serde_json::Value = resp.json().await.unwrap();
    let events = json["data"].as_array().unwrap();
    assert!(!events.is_empty());
    assert_eq!(events[0]["event_type"], "git_commit");

    handle.abort();
}

#[tokio::test]
async fn e2e_sse_endpoint_connects() {
    let (base_url, handle) = start_daemon(vec![]).await;
    let client = reqwest::Client::new();

    // Just verify SSE endpoint accepts connection (returns 200 with stream)
    let resp = client
        .get(format!("{}/api/v1/events", base_url))
        .timeout(Duration::from_millis(500))
        .send()
        .await
        .unwrap();

    assert_eq!(resp.status(), 200);
    // Content type should be text/event-stream
    let content_type = resp.headers().get("content-type").unwrap().to_str().unwrap();
    assert!(content_type.contains("text/event-stream"));

    handle.abort();
}
