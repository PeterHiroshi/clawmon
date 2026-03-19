use std::fs;
use std::path::PathBuf;
use std::process::Command;

use crate::error::Result;
use crate::models::{EnvCheck, EnvHealth, HookCheck};

/// List of tools to check for availability.
const TOOLS_TO_CHECK: &[(&str, &[&str])] = &[
    ("claude", &["--version"]),
    ("go", &["version"]),
    ("rustc", &["--version"]),
    ("cargo", &["--version"]),
    ("gh", &["--version"]),
    ("glab", &["--version"]),
    ("git", &["--version"]),
    ("make", &["--version"]),
    ("jq", &["--version"]),
];

/// Collects environment health information.
#[derive(Default)]
pub struct EnvCollector;

impl EnvCollector {
    /// Create a new environment collector.
    pub fn new() -> Self {
        Self
    }

    /// Collect environment health checks.
    pub fn collect(&self) -> Result<EnvHealth> {
        let tools = check_tools();
        let hooks = check_hooks();

        Ok(EnvHealth { tools, hooks })
    }
}

/// Check all configured tools for availability.
fn check_tools() -> Vec<EnvCheck> {
    TOOLS_TO_CHECK
        .iter()
        .map(|(name, version_args)| check_tool(name, version_args))
        .collect()
}

/// Check a single tool's availability and version.
fn check_tool(name: &str, version_args: &[&str]) -> EnvCheck {
    let which_result = which_tool(name);

    match which_result {
        Some(path) => {
            let version = get_tool_version(name, version_args);
            EnvCheck {
                tool: name.to_string(),
                installed: true,
                version,
                path: Some(path),
            }
        }
        None => EnvCheck {
            tool: name.to_string(),
            installed: false,
            version: None,
            path: None,
        },
    }
}

/// Find tool path using `which`.
fn which_tool(name: &str) -> Option<String> {
    Command::new("which")
        .arg(name)
        .output()
        .ok()
        .filter(|o| o.status.success())
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
}

/// Get tool version string.
fn get_tool_version(name: &str, args: &[&str]) -> Option<String> {
    Command::new(name)
        .args(args)
        .output()
        .ok()
        .map(|o| {
            let stdout = String::from_utf8_lossy(&o.stdout);
            let stderr = String::from_utf8_lossy(&o.stderr);
            let output = if stdout.trim().is_empty() {
                stderr.to_string()
            } else {
                stdout.to_string()
            };
            // Take first line, trim
            output.lines().next().unwrap_or("").trim().to_string()
        })
        .filter(|v| !v.is_empty())
}

/// Check hooks and plugin status.
fn check_hooks() -> Vec<HookCheck> {
    vec![check_notify_forge_hook(), check_superpowers_plugin()]
}

/// Check if ~/.claude/hooks/notify-forge.sh exists and is executable.
fn check_notify_forge_hook() -> HookCheck {
    let hook_path = home_dir().map(|h| h.join(".claude/hooks/notify-forge.sh"));

    match hook_path {
        Some(path) if path.exists() => {
            let is_executable = is_executable(&path);
            HookCheck {
                name: "notify-forge-hook".to_string(),
                installed: is_executable,
                details: if is_executable {
                    Some(format!("Found at {}", path.display()))
                } else {
                    Some("Found but not executable".to_string())
                },
            }
        }
        _ => HookCheck {
            name: "notify-forge-hook".to_string(),
            installed: false,
            details: None,
        },
    }
}

/// Check if Superpowers plugin is enabled in ~/.claude/settings.json.
fn check_superpowers_plugin() -> HookCheck {
    let settings_path = home_dir().map(|h| h.join(".claude/settings.json"));

    match settings_path {
        Some(path) if path.exists() => {
            let content = fs::read_to_string(&path).unwrap_or_default();
            let has_superpowers =
                content.contains("superpowers") || content.contains("Superpowers");
            HookCheck {
                name: "superpowers-plugin".to_string(),
                installed: has_superpowers,
                details: if has_superpowers {
                    Some("Enabled in settings.json".to_string())
                } else {
                    Some("Not found in settings.json".to_string())
                },
            }
        }
        _ => HookCheck {
            name: "superpowers-plugin".to_string(),
            installed: false,
            details: Some("settings.json not found".to_string()),
        },
    }
}

/// Check if a file is executable.
fn is_executable(path: &PathBuf) -> bool {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        fs::metadata(path)
            .map(|m| m.permissions().mode() & 0o111 != 0)
            .unwrap_or(false)
    }
    #[cfg(not(unix))]
    {
        let _ = path;
        false
    }
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
    fn test_check_tool_git() {
        let check = check_tool("git", &["--version"]);
        // Git should be installed in CI/dev environments
        assert_eq!(check.tool, "git");
        assert!(check.installed);
        assert!(check.version.is_some());
        assert!(check.path.is_some());
    }

    #[test]
    fn test_check_tool_nonexistent() {
        let check = check_tool("nonexistent_tool_xyz_12345", &["--version"]);
        assert!(!check.installed);
        assert!(check.version.is_none());
        assert!(check.path.is_none());
    }

    #[test]
    fn test_check_tools_returns_all() {
        let tools = check_tools();
        assert_eq!(tools.len(), TOOLS_TO_CHECK.len());
    }

    #[test]
    fn test_env_collector_runs() {
        let collector = EnvCollector::new();
        let health = collector.collect().unwrap();
        assert!(!health.tools.is_empty());
        assert!(!health.hooks.is_empty());
    }

    #[test]
    fn test_is_executable() {
        let tmp = TempDir::new().unwrap();
        let file = tmp.path().join("test.sh");
        fs::write(&file, "#!/bin/sh").unwrap();

        // Not executable initially
        assert!(!is_executable(&file));

        // Make executable
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            fs::set_permissions(&file, fs::Permissions::from_mode(0o755)).unwrap();
            assert!(is_executable(&file));
        }
    }

    #[test]
    fn test_which_tool_git() {
        let result = which_tool("git");
        assert!(result.is_some());
    }

    #[test]
    fn test_which_tool_nonexistent() {
        let result = which_tool("definitely_not_a_real_tool_xyz");
        assert!(result.is_none());
    }
}
