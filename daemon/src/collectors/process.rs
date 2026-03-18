use std::fs;
use std::path::Path;
use std::time::SystemTime;

use crate::config::STALL_THRESHOLD_SECS;
use crate::error::Result;
use crate::models::{ProcessInfo, ProcessState};

/// Collects information about running Claude Code processes.
#[derive(Default)]
pub struct ProcessCollector;

impl ProcessCollector {
    /// Create a new process collector.
    pub fn new() -> Self {
        Self
    }

    /// Collect all running Claude Code processes.
    pub fn collect(&self) -> Result<Vec<ProcessInfo>> {
        let mut processes = Vec::new();

        // Try /proc first (Linux), fall back to ps
        if Path::new("/proc").is_dir() {
            processes = collect_from_proc()?;
        }

        if processes.is_empty() {
            processes = collect_from_ps()?;
        }

        Ok(processes)
    }
}

/// Collect process info from /proc filesystem (Linux).
fn collect_from_proc() -> Result<Vec<ProcessInfo>> {
    let mut processes = Vec::new();

    let entries = match fs::read_dir("/proc") {
        Ok(e) => e,
        Err(_) => return Ok(processes),
    };

    for entry in entries.flatten() {
        let name = entry.file_name();
        let name_str = name.to_string_lossy();

        // Skip non-PID directories
        if !name_str.chars().all(|c| c.is_ascii_digit()) {
            continue;
        }

        let pid: u32 = match name_str.parse() {
            Ok(p) => p,
            Err(_) => continue,
        };

        let proc_path = entry.path();
        if let Some(info) = read_process_info(pid, &proc_path)
            && is_claude_process(&info.command)
        {
            processes.push(info);
        }
    }

    Ok(processes)
}

/// Read process info from /proc/PID/.
fn read_process_info(pid: u32, proc_path: &Path) -> Option<ProcessInfo> {
    let cmdline = read_cmdline(proc_path)?;
    let state = read_state(proc_path);
    let workdir = read_workdir(proc_path);
    let (memory_kb, uptime_seconds) = read_stat_info(proc_path);
    let stalled = check_stalled(&workdir);

    Some(ProcessInfo {
        pid,
        workdir,
        cpu_percent: None, // CPU requires sampling over time
        memory_kb,
        uptime_seconds,
        state,
        stalled,
        command: cmdline,
    })
}

/// Read /proc/PID/cmdline.
fn read_cmdline(proc_path: &Path) -> Option<String> {
    let data = fs::read(proc_path.join("cmdline")).ok()?;
    if data.is_empty() {
        return None;
    }
    // cmdline uses null bytes as separators
    let cmd = data
        .split(|&b| b == 0)
        .filter(|s| !s.is_empty())
        .map(|s| String::from_utf8_lossy(s).to_string())
        .collect::<Vec<_>>()
        .join(" ");
    Some(cmd)
}

/// Read process state from /proc/PID/status.
fn read_state(proc_path: &Path) -> ProcessState {
    let status = match fs::read_to_string(proc_path.join("status")) {
        Ok(s) => s,
        Err(_) => return ProcessState::Unknown,
    };

    for line in status.lines() {
        if let Some(state_str) = line.strip_prefix("State:\t") {
            return match state_str.chars().next() {
                Some('R') => ProcessState::Running,
                Some('S') => ProcessState::Sleeping,
                Some('Z') => ProcessState::Zombie,
                Some('T') => ProcessState::Stopped,
                _ => ProcessState::Unknown,
            };
        }
    }

    ProcessState::Unknown
}

/// Read working directory from /proc/PID/cwd symlink.
fn read_workdir(proc_path: &Path) -> Option<String> {
    fs::read_link(proc_path.join("cwd"))
        .ok()
        .map(|p| p.to_string_lossy().to_string())
}

/// Read memory and uptime from /proc/PID/stat and /proc/PID/status.
fn read_stat_info(proc_path: &Path) -> (Option<u64>, Option<u64>) {
    let memory_kb = read_memory_kb(proc_path);
    let uptime = read_uptime_seconds(proc_path);
    (memory_kb, uptime)
}

/// Read VmRSS from /proc/PID/status.
fn read_memory_kb(proc_path: &Path) -> Option<u64> {
    let status = fs::read_to_string(proc_path.join("status")).ok()?;
    for line in status.lines() {
        if let Some(val) = line.strip_prefix("VmRSS:") {
            let val = val.trim();
            // Format is "12345 kB"
            return val.split_whitespace().next()?.parse().ok();
        }
    }
    None
}

/// Estimate process uptime from /proc/PID/stat starttime.
fn read_uptime_seconds(proc_path: &Path) -> Option<u64> {
    let stat = fs::read_to_string(proc_path.join("stat")).ok()?;
    // Field 22 (0-indexed: 21) is starttime in clock ticks
    let fields: Vec<&str> = stat.split_whitespace().collect();
    if fields.len() < 22 {
        return None;
    }
    let starttime_ticks: u64 = fields[21].parse().ok()?;
    let ticks_per_sec: u64 = 100; // sysconf(_SC_CLK_TCK), usually 100
    let uptime_str = fs::read_to_string("/proc/uptime").ok()?;
    let system_uptime_secs: f64 = uptime_str.split_whitespace().next()?.parse().ok()?;
    let process_start_secs = starttime_ticks / ticks_per_sec;
    let uptime = system_uptime_secs as u64 - process_start_secs;
    Some(uptime)
}

/// Check if a process is stalled by checking last modification time in workdir.
fn check_stalled(workdir: &Option<String>) -> bool {
    let workdir = match workdir {
        Some(w) => w,
        None => return false,
    };

    let path = Path::new(workdir);
    if !path.is_dir() {
        return false;
    }

    // Check the most recent modification in the directory
    let latest_mod = most_recent_modification(path);
    match latest_mod {
        Some(time) => {
            let elapsed = SystemTime::now()
                .duration_since(time)
                .unwrap_or_default()
                .as_secs();
            elapsed > STALL_THRESHOLD_SECS
        }
        None => false,
    }
}

/// Find the most recent file modification time in a directory (non-recursive, top level only).
fn most_recent_modification(dir: &Path) -> Option<SystemTime> {
    let mut latest = None;

    let entries = fs::read_dir(dir).ok()?;
    for entry in entries.flatten() {
        if let Ok(metadata) = entry.metadata()
            && let Ok(modified) = metadata.modified()
        {
            latest = Some(match latest {
                Some(current) if modified > current => modified,
                Some(current) => current,
                None => modified,
            });
        }
    }

    latest
}

/// Check if a command line belongs to a Claude Code process.
fn is_claude_process(cmdline: &str) -> bool {
    let lower = cmdline.to_lowercase();
    lower.contains("claude") || lower.contains("claude-code")
}

/// Fallback: collect from `ps aux` output.
fn collect_from_ps() -> Result<Vec<ProcessInfo>> {
    let output = match std::process::Command::new("ps").args(["aux"]).output() {
        Ok(o) => o,
        Err(_) => return Ok(vec![]),
    };

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut processes = Vec::new();

    for line in stdout.lines().skip(1) {
        // Skip header
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() < 11 {
            continue;
        }

        let command = fields[10..].join(" ");
        if !is_claude_process(&command) {
            continue;
        }

        let pid: u32 = match fields[1].parse() {
            Ok(p) => p,
            Err(_) => continue,
        };

        let cpu_percent: Option<f64> = fields[2].parse().ok();
        let memory_kb: Option<u64> = fields[5].parse().ok();

        processes.push(ProcessInfo {
            pid,
            workdir: None,
            cpu_percent,
            memory_kb,
            uptime_seconds: None,
            state: ProcessState::Unknown,
            stalled: false,
            command,
        });
    }

    Ok(processes)
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    #[test]
    fn test_is_claude_process() {
        assert!(is_claude_process("/usr/bin/claude --no-tty"));
        assert!(is_claude_process("node /home/user/.claude/claude-code"));
        assert!(is_claude_process("Claude something"));
        assert!(!is_claude_process("vim file.rs"));
        assert!(!is_claude_process("cargo test"));
    }

    #[test]
    fn test_check_stalled_no_workdir() {
        assert!(!check_stalled(&None));
    }

    #[test]
    fn test_check_stalled_nonexistent_dir() {
        assert!(!check_stalled(&Some("/nonexistent/dir/12345".to_string())));
    }

    #[test]
    fn test_check_stalled_recent_activity() {
        let tmp = TempDir::new().unwrap();
        fs::write(tmp.path().join("recent_file.txt"), "data").unwrap();
        assert!(!check_stalled(&Some(
            tmp.path().to_string_lossy().to_string()
        )));
    }

    #[test]
    fn test_most_recent_modification() {
        let tmp = TempDir::new().unwrap();
        fs::write(tmp.path().join("file1.txt"), "a").unwrap();
        fs::write(tmp.path().join("file2.txt"), "b").unwrap();

        let result = most_recent_modification(tmp.path());
        assert!(result.is_some());
    }

    #[test]
    fn test_most_recent_modification_empty_dir() {
        let tmp = TempDir::new().unwrap();
        let result = most_recent_modification(tmp.path());
        assert!(result.is_none());
    }

    #[test]
    fn test_process_collector_runs() {
        let collector = ProcessCollector::new();
        // Should not panic, may return empty list
        let result = collector.collect();
        assert!(result.is_ok());
    }
}
