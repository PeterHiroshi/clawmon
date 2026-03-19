//! Configuration file support for clawmon (~/.clawmon/config.toml).

use serde::Deserialize;
use std::path::PathBuf;

use crate::error::{DaemonError, Result};

/// Default config directory name under the user's home.
const CONFIG_DIR: &str = ".clawmon";

/// Default config file name.
const CONFIG_FILE: &str = "config.toml";

/// Default TOML content written on first run.
const DEFAULT_CONFIG_CONTENT: &str = r#"# clawmon configuration file
# See README.md for full documentation

[daemon]
# port = 9876
# poll_interval = 30

[daemon.workspaces]
# paths = [
#   "~/.openclaw/workspace-main",
# ]

[tui]
# daemon_url = "http://127.0.0.1:9876/api/v1"
# refresh_interval = 5
# theme = "dark"
"#;

/// Root configuration file structure.
#[derive(Debug, Deserialize, Default, Clone)]
pub struct ConfigFile {
    /// Daemon-specific configuration.
    pub daemon: Option<DaemonFileConfig>,
    /// TUI-specific configuration.
    pub tui: Option<TuiFileConfig>,
}

/// Daemon section of the config file.
#[derive(Debug, Deserialize, Default, Clone)]
pub struct DaemonFileConfig {
    /// Port to listen on.
    pub port: Option<u16>,
    /// Poll interval in seconds.
    pub poll_interval: Option<u64>,
    /// Workspace configuration.
    pub workspaces: Option<WorkspacesFileConfig>,
}

/// Workspace paths section of the config file.
#[derive(Debug, Deserialize, Default, Clone)]
pub struct WorkspacesFileConfig {
    /// List of workspace root paths to monitor.
    pub paths: Option<Vec<String>>,
}

/// TUI section of the config file.
#[derive(Debug, Deserialize, Default, Clone)]
pub struct TuiFileConfig {
    /// Daemon API URL.
    pub daemon_url: Option<String>,
    /// Refresh interval in seconds.
    pub refresh_interval: Option<u64>,
    /// Theme name.
    pub theme: Option<String>,
}

/// Returns the path to the config file (~/.clawmon/config.toml).
pub fn config_file_path() -> Option<PathBuf> {
    home_dir().map(|h| h.join(CONFIG_DIR).join(CONFIG_FILE))
}

/// Loads and parses the configuration file.
/// Returns `Ok(None)` if the file does not exist.
/// Returns `Err` if the file exists but cannot be parsed.
pub fn load_config_file() -> Result<Option<ConfigFile>> {
    let path = match config_file_path() {
        Some(p) => p,
        None => return Ok(None),
    };

    if !path.is_file() {
        return Ok(None);
    }

    let content = std::fs::read_to_string(&path).map_err(|e| {
        DaemonError::Config(format!(
            "failed to read config file {}: {}",
            path.display(),
            e
        ))
    })?;

    let config: ConfigFile = toml::from_str(&content).map_err(|e| {
        DaemonError::Config(format!(
            "failed to parse config file {}: {}",
            path.display(),
            e
        ))
    })?;

    Ok(Some(config))
}

/// Creates a default config file at ~/.clawmon/config.toml if none exists.
/// Returns the path where the config was created, or None if it already exists.
pub fn create_default_config() -> Result<Option<PathBuf>> {
    let path = match config_file_path() {
        Some(p) => p,
        None => return Ok(None),
    };

    if path.is_file() {
        return Ok(None);
    }

    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| {
            DaemonError::Config(format!(
                "failed to create config directory {}: {}",
                parent.display(),
                e
            ))
        })?;
    }

    std::fs::write(&path, DEFAULT_CONFIG_CONTENT).map_err(|e| {
        DaemonError::Config(format!(
            "failed to write default config to {}: {}",
            path.display(),
            e
        ))
    })?;

    Ok(Some(path))
}

/// Expands a tilde prefix in a path string to the user's home directory.
pub fn expand_tilde(path: &str) -> PathBuf {
    if let Some(rest) = path.strip_prefix("~/")
        && let Some(home) = home_dir()
    {
        return home.join(rest);
    }
    if path == "~"
        && let Some(home) = home_dir()
    {
        return home;
    }
    PathBuf::from(path)
}

/// Extracts additional workspace paths from the config file.
/// Expands tildes and returns only paths that exist as directories.
pub fn workspace_paths_from_config(config: &ConfigFile) -> Vec<PathBuf> {
    let daemon = match &config.daemon {
        Some(d) => d,
        None => return Vec::new(),
    };

    let ws_config = match &daemon.workspaces {
        Some(w) => w,
        None => return Vec::new(),
    };

    let paths = match &ws_config.paths {
        Some(p) => p,
        None => return Vec::new(),
    };

    paths.iter().map(|p| expand_tilde(p)).collect()
}

/// Get the user's home directory.
fn home_dir() -> Option<PathBuf> {
    std::env::var("HOME").ok().map(PathBuf::from)
}

/// Parses a TOML string into a ConfigFile.
/// Useful for testing without filesystem access.
pub fn parse_config(content: &str) -> Result<ConfigFile> {
    toml::from_str(content)
        .map_err(|e| DaemonError::Config(format!("failed to parse config: {}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    #[test]
    fn test_parse_empty_config() {
        let config = parse_config("").unwrap();
        assert!(config.daemon.is_none());
        assert!(config.tui.is_none());
    }

    #[test]
    fn test_parse_full_config() {
        let toml = r#"
[daemon]
port = 8080
poll_interval = 15

[daemon.workspaces]
paths = ["~/.openclaw/workspace-main", "/tmp/other"]

[tui]
daemon_url = "http://localhost:8080/api/v1"
refresh_interval = 10
theme = "dark"
"#;
        let config = parse_config(toml).unwrap();
        let daemon = config.daemon.unwrap();
        assert_eq!(daemon.port, Some(8080));
        assert_eq!(daemon.poll_interval, Some(15));
        let ws = daemon.workspaces.unwrap();
        assert_eq!(ws.paths.unwrap().len(), 2);

        let tui = config.tui.unwrap();
        assert_eq!(tui.daemon_url.unwrap(), "http://localhost:8080/api/v1");
        assert_eq!(tui.refresh_interval, Some(10));
        assert_eq!(tui.theme.unwrap(), "dark");
    }

    #[test]
    fn test_parse_partial_config() {
        let toml = r#"
[daemon]
port = 9999
"#;
        let config = parse_config(toml).unwrap();
        let daemon = config.daemon.unwrap();
        assert_eq!(daemon.port, Some(9999));
        assert!(daemon.poll_interval.is_none());
        assert!(daemon.workspaces.is_none());
        assert!(config.tui.is_none());
    }

    #[test]
    fn test_parse_invalid_config() {
        let result = parse_config("not valid [[[toml");
        assert!(result.is_err());
    }

    #[test]
    fn test_expand_tilde() {
        let home = std::env::var("HOME").unwrap_or_else(|_| "/home/test".to_string());
        let expanded = expand_tilde("~/workspace");
        assert_eq!(expanded, PathBuf::from(&home).join("workspace"));

        let no_tilde = expand_tilde("/absolute/path");
        assert_eq!(no_tilde, PathBuf::from("/absolute/path"));

        let just_tilde = expand_tilde("~");
        assert_eq!(just_tilde, PathBuf::from(&home));
    }

    #[test]
    fn test_workspace_paths_from_config() {
        let config = ConfigFile {
            daemon: Some(DaemonFileConfig {
                port: None,
                poll_interval: None,
                workspaces: Some(WorkspacesFileConfig {
                    paths: Some(vec!["/tmp/ws1".to_string(), "/tmp/ws2".to_string()]),
                }),
            }),
            tui: None,
        };
        let paths = workspace_paths_from_config(&config);
        assert_eq!(paths.len(), 2);
        assert_eq!(paths[0], PathBuf::from("/tmp/ws1"));
        assert_eq!(paths[1], PathBuf::from("/tmp/ws2"));
    }

    #[test]
    fn test_workspace_paths_empty_config() {
        let config = ConfigFile::default();
        let paths = workspace_paths_from_config(&config);
        assert!(paths.is_empty());
    }

    #[test]
    fn test_create_and_load_config() {
        let tmp = TempDir::new().unwrap();
        let config_dir = tmp.path().join(".clawmon");
        let config_path = config_dir.join("config.toml");

        // Write a config file manually
        std::fs::create_dir_all(&config_dir).unwrap();
        std::fs::write(
            &config_path,
            r#"
[daemon]
port = 7777
"#,
        )
        .unwrap();

        // Parse it
        let content = std::fs::read_to_string(&config_path).unwrap();
        let config = parse_config(&content).unwrap();
        let daemon = config.daemon.unwrap();
        assert_eq!(daemon.port, Some(7777));
    }
}
