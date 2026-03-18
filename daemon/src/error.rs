/// Errors that can occur in the clawmon daemon.
#[derive(Debug, thiserror::Error)]
pub enum DaemonError {
    /// Git operation failed.
    #[error("git error: {0}")]
    Git(#[from] git2::Error),

    /// IO operation failed.
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),

    /// JSON parsing failed.
    #[error("json error: {0}")]
    Json(#[from] serde_json::Error),

    /// File watcher error.
    #[error("watcher error: {0}")]
    Watcher(#[from] notify::Error),

    /// Workspace not found.
    #[error("workspace not found: {0}")]
    WorkspaceNotFound(String),

    /// Configuration error.
    #[error("config error: {0}")]
    Config(String),

    /// Collector failed.
    #[error("collector error: {0}")]
    Collector(String),
}

pub type Result<T> = std::result::Result<T, DaemonError>;
