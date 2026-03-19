use axum::Router;
use axum::routing::get;

use super::handlers;
use crate::api::handlers::AppState;

/// Build the API router with all routes.
pub fn build_router(state: AppState) -> Router {
    Router::new()
        .route("/api/v1/health", get(handlers::health))
        .route("/api/v1/workspaces", get(handlers::list_workspaces))
        .route("/api/v1/workspaces/{id}/git", get(handlers::workspace_git))
        .route(
            "/api/v1/workspaces/{id}/tasks",
            get(handlers::workspace_tasks),
        )
        .route(
            "/api/v1/workspaces/{id}/processes",
            get(handlers::workspace_processes),
        )
        .route("/api/v1/workspaces/{id}/env", get(handlers::workspace_env))
        .route(
            "/api/v1/workspaces/{id}/system",
            get(handlers::workspace_system),
        )
        .route(
            "/api/v1/workspaces/{id}/activity",
            get(handlers::workspace_activity),
        )
        .route("/api/v1/events", get(handlers::sse_events))
        .with_state(state)
}
