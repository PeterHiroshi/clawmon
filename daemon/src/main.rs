use std::net::SocketAddr;
use std::path::PathBuf;

use clap::Parser;
use tokio::net::TcpListener;
use tracing::{error, info};

use clawmon_daemon::api::handlers::AppState;
use clawmon_daemon::api::routes::build_router;
use clawmon_daemon::config::{CliArgs, Config, VERSION};
use clawmon_daemon::models::SseEvent;
use clawmon_daemon::watcher::{self, FileWatcher};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "clawmon_daemon=info".into()),
        )
        .with_target(false)
        .init();

    let args = CliArgs::parse();
    let config = Config::from_args(&args)?;

    info!("clawmon-daemon v{} starting", VERSION);
    info!("workspace: {}", config.workspace_path.display());
    info!("monitoring {} project(s)", config.project_dirs.len());

    let port = config.port;
    let state = AppState::new(config.clone());

    // Start file watcher
    let watch_paths: Vec<PathBuf> = config.project_dirs.clone();
    let event_bus = state.event_bus.clone();
    if !watch_paths.is_empty() {
        match FileWatcher::new(&watch_paths) {
            Ok((_watcher, mut rx)) => {
                let bus = event_bus.clone();
                tokio::spawn(async move {
                    while let Some(change) = rx.recv().await {
                        if watcher::is_relevant_change(&change.path) {
                            bus.publish(SseEvent {
                                event_type: "file_changed".to_string(),
                                workspace_id: None,
                                data: serde_json::json!({
                                    "path": change.path.to_string_lossy(),
                                    "kind": format!("{:?}", change.kind),
                                }),
                            });
                        }
                    }
                });
                info!("file watcher started");
            }
            Err(err) => {
                error!("failed to start file watcher: {}", err);
            }
        }
    }

    // Start periodic poll
    let poll_state = state.clone();
    let poll_interval = config.poll_interval;
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(poll_interval);
        loop {
            interval.tick().await;
            // Publish a poll event to signal TUI to refresh
            poll_state.event_bus.publish(SseEvent {
                event_type: "poll".to_string(),
                workspace_id: None,
                data: serde_json::json!({"message": "periodic refresh"}),
            });
        }
    });

    let router = build_router(state);
    let addr = SocketAddr::from(([127, 0, 0, 1], port));
    let listener = TcpListener::bind(addr).await?;
    info!("listening on http://{}", addr);

    axum::serve(listener, router)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    info!("daemon shutdown complete");
    Ok(())
}

async fn shutdown_signal() {
    tokio::signal::ctrl_c()
        .await
        .expect("failed to install CTRL+C signal handler");
    info!("shutdown signal received");
}
