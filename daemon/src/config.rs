use clap::Parser;
use std::path::{Path, PathBuf};
use std::time::Duration;

use crate::error::{DaemonError, Result};

/// Default daemon port.
pub const DEFAULT_PORT: u16 = 9876;

/// Default poll interval in seconds.
pub const DEFAULT_POLL_INTERVAL_SECS: u64 = 30;

/// Default workspace base path.
pub const DEFAULT_WORKSPACE_SUBPATH: &str = ".openclaw/workspace-main";

/// Maximum number of recent commits to fetch.
pub const MAX_RECENT_COMMITS: usize = 20;

/// Stall detection threshold in seconds (15 minutes).
pub const STALL_THRESHOLD_SECS: u64 = 900;

/// Daemon version.
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// CLI arguments for the clawmon daemon.
#[derive(Parser, Debug, Clone)]
#[command(
    name = "clawmon-daemon",
    about = "Monitoring daemon for OpenClaw AI agents"
)]
pub struct CliArgs {
    /// Port to listen on.
    #[arg(long, default_value_t = DEFAULT_PORT)]
    pub port: u16,

    /// Workspace root path. Auto-detected from OPENCLAW_WORKSPACE env or ~/.openclaw/workspace-main.
    #[arg(long)]
    pub workspace: Option<String>,

    /// Poll interval in seconds for data collection.
    #[arg(long, default_value_t = DEFAULT_POLL_INTERVAL_SECS)]
    pub poll_interval: u64,

    /// Additional project directories to monitor (comma-separated).
    #[arg(long, value_delimiter = ',')]
    pub project_dirs: Vec<String>,
}

/// Resolved daemon configuration.
#[derive(Debug, Clone)]
pub struct Config {
    /// Port to bind HTTP server.
    pub port: u16,
    /// Root workspace path.
    pub workspace_path: PathBuf,
    /// Poll interval for data collection.
    pub poll_interval: Duration,
    /// All project directories to monitor.
    pub project_dirs: Vec<PathBuf>,
}

impl Config {
    /// Build configuration from CLI args, resolving workspace path and scanning for projects.
    pub fn from_args(args: &CliArgs) -> Result<Self> {
        let workspace_path = resolve_workspace_path(args.workspace.as_deref())?;
        let mut project_dirs = scan_project_dirs(&workspace_path);

        // Add explicitly specified project dirs
        for dir in &args.project_dirs {
            let path = PathBuf::from(dir);
            if path.is_dir() && !project_dirs.contains(&path) {
                project_dirs.push(path);
            }
        }

        Ok(Self {
            port: args.port,
            workspace_path,
            poll_interval: Duration::from_secs(args.poll_interval),
            project_dirs,
        })
    }
}

/// Resolve the workspace path from explicit arg, env var, or default location.
fn resolve_workspace_path(explicit: Option<&str>) -> Result<PathBuf> {
    if let Some(path) = explicit {
        let p = PathBuf::from(path);
        if p.is_dir() {
            return Ok(p);
        }
        return Err(DaemonError::Config(format!(
            "specified workspace path does not exist: {}",
            path
        )));
    }

    if let Ok(env_path) = std::env::var("OPENCLAW_WORKSPACE") {
        let p = PathBuf::from(&env_path);
        if p.is_dir() {
            return Ok(p);
        }
    }

    if let Some(home) = home_dir() {
        let default_path = home.join(DEFAULT_WORKSPACE_SUBPATH);
        if default_path.is_dir() {
            return Ok(default_path);
        }
    }

    Err(DaemonError::Config(
        "could not detect workspace path; set --workspace or OPENCLAW_WORKSPACE".to_string(),
    ))
}

/// Scan workspace/projects/ for subdirectories containing .git/.
fn scan_project_dirs(workspace: &Path) -> Vec<PathBuf> {
    let projects_dir = workspace.join("projects");
    let mut dirs = Vec::new();

    if let Ok(entries) = std::fs::read_dir(&projects_dir) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() && path.join(".git").is_dir() {
                dirs.push(path);
            }
        }
    }

    dirs.sort();
    dirs
}

/// Get the user's home directory.
fn home_dir() -> Option<PathBuf> {
    std::env::var("HOME").ok().map(PathBuf::from)
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    #[test]
    fn test_default_config_values() {
        assert_eq!(DEFAULT_PORT, 9876);
        assert_eq!(DEFAULT_POLL_INTERVAL_SECS, 30);
    }

    #[test]
    fn test_resolve_workspace_explicit_path() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().to_str().unwrap();
        let result = resolve_workspace_path(Some(path));
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), tmp.path());
    }

    #[test]
    fn test_resolve_workspace_explicit_nonexistent() {
        let result = resolve_workspace_path(Some("/nonexistent/path/12345"));
        assert!(result.is_err());
    }

    #[test]
    fn test_scan_project_dirs_finds_git_repos() {
        let tmp = TempDir::new().unwrap();
        let projects = tmp.path().join("projects");
        std::fs::create_dir_all(projects.join("repo-a/.git")).unwrap();
        std::fs::create_dir_all(projects.join("repo-b/.git")).unwrap();
        std::fs::create_dir_all(projects.join("not-a-repo")).unwrap();

        let dirs = scan_project_dirs(tmp.path());
        assert_eq!(dirs.len(), 2);
        assert!(dirs.iter().any(|d| d.ends_with("repo-a")));
        assert!(dirs.iter().any(|d| d.ends_with("repo-b")));
    }

    #[test]
    fn test_scan_project_dirs_empty_workspace() {
        let tmp = TempDir::new().unwrap();
        let dirs = scan_project_dirs(tmp.path());
        assert!(dirs.is_empty());
    }

    #[test]
    fn test_config_from_args() {
        let tmp = TempDir::new().unwrap();
        let projects = tmp.path().join("projects");
        std::fs::create_dir_all(projects.join("myproject/.git")).unwrap();

        let args = CliArgs {
            port: 8080,
            workspace: Some(tmp.path().to_str().unwrap().to_string()),
            poll_interval: 15,
            project_dirs: vec![],
        };

        let config = Config::from_args(&args).unwrap();
        assert_eq!(config.port, 8080);
        assert_eq!(config.poll_interval, Duration::from_secs(15));
        assert_eq!(config.project_dirs.len(), 1);
    }
}
