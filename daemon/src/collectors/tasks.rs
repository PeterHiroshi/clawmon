use std::fs;
use std::path::{Path, PathBuf};

use chrono::{DateTime, Utc};

use crate::error::Result;
use crate::models::{TaskInfo, TaskStatus};

/// Task spec JSON fields we care about.
#[derive(serde::Deserialize, Default)]
struct TaskSpec {
    #[serde(default)]
    name: Option<String>,
    #[serde(default)]
    status: Option<String>,
    #[serde(default)]
    started_at: Option<String>,
    #[serde(default)]
    completed_at: Option<String>,
    #[serde(default)]
    model: Option<String>,
    #[serde(default)]
    agent_teams: Option<bool>,
}

/// Meta JSON fields.
#[derive(serde::Deserialize, Default)]
struct TaskMeta {
    #[serde(default)]
    exit_code: Option<i32>,
    #[serde(default)]
    effective_model: Option<String>,
    #[serde(default)]
    duration: Option<u64>,
}

/// Collects task information from .forge-task/ directories.
pub struct TaskCollector {
    search_paths: Vec<PathBuf>,
}

impl TaskCollector {
    /// Create a new task collector that searches given paths for .forge-task/ dirs.
    pub fn new(search_paths: &[PathBuf]) -> Self {
        Self {
            search_paths: search_paths.to_vec(),
        }
    }

    /// Collect all tasks from configured search paths.
    pub fn collect(&self) -> Result<Vec<TaskInfo>> {
        let mut tasks = Vec::new();

        for base_path in &self.search_paths {
            // Check for .forge-task/ directly in the path
            let forge_dir = base_path.join(".forge-task");
            if forge_dir.is_dir()
                && let Some(task) = parse_task_dir(&forge_dir)
            {
                tasks.push(task);
            }

            // Also scan subdirectories (project dirs may have nested tasks)
            if let Ok(entries) = fs::read_dir(base_path) {
                for entry in entries.flatten() {
                    let subdir = entry.path().join(".forge-task");
                    if subdir.is_dir()
                        && let Some(task) = parse_task_dir(&subdir)
                    {
                        tasks.push(task);
                    }
                }
            }
        }

        // Sort by started_at descending (most recent first)
        tasks.sort_by(|a, b| b.started_at.cmp(&a.started_at));

        Ok(tasks)
    }
}

/// Parse a single .forge-task/ directory into a TaskInfo.
fn parse_task_dir(dir: &Path) -> Option<TaskInfo> {
    let task_spec = parse_task_spec(dir);
    let meta = parse_meta(dir);
    let (current_step, progress_percent) = parse_progress(dir);

    let name = task_spec.name.unwrap_or_else(|| {
        dir.parent()
            .and_then(|p| p.file_name())
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_else(|| "unknown".to_string())
    });

    let status = match task_spec.status.as_deref() {
        Some("pending") => TaskStatus::Pending,
        Some("in_progress") => TaskStatus::InProgress,
        Some("done") => TaskStatus::Done,
        Some("failed") => TaskStatus::Failed,
        _ => TaskStatus::Unknown,
    };

    Some(TaskInfo {
        name,
        status,
        started_at: task_spec.started_at.and_then(|s| parse_datetime(&s)),
        completed_at: task_spec.completed_at.and_then(|s| parse_datetime(&s)),
        model: task_spec.model,
        agent_teams: task_spec.agent_teams,
        current_step,
        progress_percent,
        exit_code: meta.exit_code,
        effective_model: meta.effective_model,
        duration_seconds: meta.duration,
        task_dir: dir.to_string_lossy().to_string(),
    })
}

/// Parse task-spec.json from a .forge-task/ directory.
fn parse_task_spec(dir: &Path) -> TaskSpec {
    let spec_path = dir.join("task-spec.json");
    fs::read_to_string(&spec_path)
        .ok()
        .and_then(|content| serde_json::from_str(&content).ok())
        .unwrap_or_default()
}

/// Parse meta.json from a .forge-task/ directory.
fn parse_meta(dir: &Path) -> TaskMeta {
    let meta_path = dir.join("meta.json");
    fs::read_to_string(&meta_path)
        .ok()
        .and_then(|content| serde_json::from_str(&content).ok())
        .unwrap_or_default()
}

/// Parse progress.md for current step and percentage.
fn parse_progress(dir: &Path) -> (Option<String>, Option<u8>) {
    let progress_path = dir.join("progress.md");
    let content = match fs::read_to_string(&progress_path) {
        Ok(c) => c,
        Err(_) => return (None, None),
    };

    let mut current_step = None;
    let mut progress_percent = None;

    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with("## In Progress") {
            // Next non-empty line after this header is the current step
            continue;
        }
        if current_step.is_none() && trimmed.starts_with("- ") {
            // Extract step after "## In Progress" section
            if content.contains("## In Progress") {
                let in_progress_section = content.split("## In Progress").nth(1);
                if let Some(section) = in_progress_section {
                    for section_line in section.lines() {
                        let sl = section_line.trim();
                        if sl.starts_with("- ") {
                            current_step = Some(sl.trim_start_matches("- ").to_string());
                            break;
                        }
                        if sl.starts_with("## ") && sl != "## In Progress" {
                            break;
                        }
                    }
                }
            }
        }
        // Look for percentage patterns like "50%" or "Progress: 50%"
        if let Some(pct) = extract_percentage(trimmed) {
            progress_percent = Some(pct);
        }
    }

    (current_step, progress_percent)
}

/// Extract a percentage value from a string like "50%" or "Progress: 75%".
fn extract_percentage(text: &str) -> Option<u8> {
    for word in text.split_whitespace() {
        if let Some(num_str) = word.strip_suffix('%')
            && let Ok(num) = num_str.parse::<u8>()
            && num <= 100
        {
            return Some(num);
        }
    }
    None
}

/// Parse a datetime string, trying ISO 8601 formats.
fn parse_datetime(s: &str) -> Option<DateTime<Utc>> {
    s.parse::<DateTime<Utc>>().ok()
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    fn create_forge_task(dir: &Path, spec: &str, meta: Option<&str>, progress: Option<&str>) {
        let forge_dir = dir.join(".forge-task");
        fs::create_dir_all(&forge_dir).unwrap();
        fs::write(forge_dir.join("task-spec.json"), spec).unwrap();
        if let Some(m) = meta {
            fs::write(forge_dir.join("meta.json"), m).unwrap();
        }
        if let Some(p) = progress {
            fs::write(forge_dir.join("progress.md"), p).unwrap();
        }
    }

    #[test]
    fn test_parse_task_spec_basic() {
        let tmp = TempDir::new().unwrap();
        create_forge_task(
            tmp.path(),
            r#"{"name": "test-task", "status": "in_progress", "model": "opus"}"#,
            None,
            None,
        );

        let collector = TaskCollector::new(&[tmp.path().to_path_buf()]);
        let tasks = collector.collect().unwrap();
        assert_eq!(tasks.len(), 1);
        assert_eq!(tasks[0].name, "test-task");
        assert_eq!(tasks[0].status, TaskStatus::InProgress);
        assert_eq!(tasks[0].model, Some("opus".to_string()));
    }

    #[test]
    fn test_parse_task_with_meta() {
        let tmp = TempDir::new().unwrap();
        create_forge_task(
            tmp.path(),
            r#"{"name": "done-task", "status": "done"}"#,
            Some(r#"{"exit_code": 0, "effective_model": "sonnet", "duration": 120}"#),
            None,
        );

        let collector = TaskCollector::new(&[tmp.path().to_path_buf()]);
        let tasks = collector.collect().unwrap();
        assert_eq!(tasks[0].exit_code, Some(0));
        assert_eq!(tasks[0].effective_model, Some("sonnet".to_string()));
        assert_eq!(tasks[0].duration_seconds, Some(120));
    }

    #[test]
    fn test_parse_progress_md() {
        let tmp = TempDir::new().unwrap();
        create_forge_task(
            tmp.path(),
            r#"{"name": "prog-task", "status": "in_progress"}"#,
            None,
            Some(
                "# Progress\n## In Progress\n- Building API handlers\n## Remaining\n- Tests\nProgress: 60%",
            ),
        );

        let collector = TaskCollector::new(&[tmp.path().to_path_buf()]);
        let tasks = collector.collect().unwrap();
        assert_eq!(
            tasks[0].current_step,
            Some("Building API handlers".to_string())
        );
        assert_eq!(tasks[0].progress_percent, Some(60));
    }

    #[test]
    fn test_no_forge_task_dir() {
        let tmp = TempDir::new().unwrap();
        let collector = TaskCollector::new(&[tmp.path().to_path_buf()]);
        let tasks = collector.collect().unwrap();
        assert!(tasks.is_empty());
    }

    #[test]
    fn test_multiple_tasks_sorted() {
        let tmp = TempDir::new().unwrap();
        let sub1 = tmp.path().join("project-a");
        let sub2 = tmp.path().join("project-b");
        fs::create_dir_all(&sub1).unwrap();
        fs::create_dir_all(&sub2).unwrap();

        create_forge_task(
            &sub1,
            r#"{"name": "older-task", "status": "done", "started_at": "2026-03-18T10:00:00Z"}"#,
            None,
            None,
        );
        create_forge_task(
            &sub2,
            r#"{"name": "newer-task", "status": "in_progress", "started_at": "2026-03-18T12:00:00Z"}"#,
            None,
            None,
        );

        let collector = TaskCollector::new(&[tmp.path().to_path_buf()]);
        let tasks = collector.collect().unwrap();
        assert_eq!(tasks.len(), 2);
        assert_eq!(tasks[0].name, "newer-task"); // Most recent first
    }

    #[test]
    fn test_extract_percentage() {
        assert_eq!(extract_percentage("Progress: 50%"), Some(50));
        assert_eq!(extract_percentage("75%"), Some(75));
        assert_eq!(extract_percentage("no percent here"), None);
        assert_eq!(extract_percentage("100%"), Some(100));
    }

    #[test]
    fn test_task_status_parsing() {
        let cases = vec![
            ("pending", TaskStatus::Pending),
            ("in_progress", TaskStatus::InProgress),
            ("done", TaskStatus::Done),
            ("failed", TaskStatus::Failed),
            ("unknown_val", TaskStatus::Unknown),
        ];

        for (input, expected) in cases {
            let tmp = TempDir::new().unwrap();
            create_forge_task(
                tmp.path(),
                &format!(r#"{{"name": "t", "status": "{}"}}"#, input),
                None,
                None,
            );
            let collector = TaskCollector::new(&[tmp.path().to_path_buf()]);
            let tasks = collector.collect().unwrap();
            assert_eq!(tasks[0].status, expected, "Failed for input: {}", input);
        }
    }
}
