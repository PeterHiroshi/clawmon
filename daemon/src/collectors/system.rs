use std::process::Command;

use sysinfo::{Disks, System};

use crate::error::Result;
use crate::models::{CpuInfo, DiskInfo, GpuInfo, MemoryInfo, SwapInfo, SystemResources};

/// Mount points to skip when collecting disk info.
const SKIP_MOUNT_PREFIXES: &[&str] = &["/proc", "/sys", "/dev", "/run", "/snap"];

/// Collects host system resource metrics.
#[derive(Default)]
pub struct SystemCollector {
    sys: System,
}

impl SystemCollector {
    /// Create a new system collector with initial data.
    pub fn new() -> Self {
        let mut sys = System::new_all();
        // Initial refresh for CPU usage baseline
        sys.refresh_all();
        Self { sys }
    }

    /// Collect current system resource metrics.
    pub fn collect(&mut self) -> Result<SystemResources> {
        self.sys.refresh_all();

        let cpu = collect_cpu(&self.sys);
        let memory = collect_memory(&self.sys);
        let swap = collect_swap(&self.sys);
        let disks = collect_disks();
        let gpus = collect_gpus();
        let uptime_seconds = System::uptime();
        let collected_at = chrono::Utc::now().to_rfc3339();

        Ok(SystemResources {
            cpu,
            memory,
            swap,
            disks,
            gpus,
            uptime_seconds,
            collected_at,
        })
    }
}

/// Collect CPU information.
fn collect_cpu(sys: &System) -> CpuInfo {
    let cpus = sys.cpus();
    let model = cpus
        .first()
        .map(|c| c.brand().to_string())
        .unwrap_or_else(|| "Unknown".to_string());

    let core_count = cpus.len();
    let per_core_usage: Vec<f32> = cpus.iter().map(|c| c.cpu_usage()).collect();

    let usage_percent = if core_count > 0 {
        per_core_usage.iter().sum::<f32>() / core_count as f32
    } else {
        0.0
    };

    let load_avg = System::load_average();

    CpuInfo {
        model,
        core_count,
        usage_percent,
        per_core_usage,
        load_avg_1m: load_avg.one,
        load_avg_5m: load_avg.five,
        load_avg_15m: load_avg.fifteen,
    }
}

/// Collect memory usage.
fn collect_memory(sys: &System) -> MemoryInfo {
    let total = sys.total_memory();
    let used = sys.used_memory();
    let available = sys.available_memory();

    let usage_percent = if total > 0 {
        (used as f32 / total as f32) * 100.0
    } else {
        0.0
    };

    MemoryInfo {
        total_bytes: total,
        used_bytes: used,
        available_bytes: available,
        usage_percent,
    }
}

/// Collect swap usage.
fn collect_swap(sys: &System) -> SwapInfo {
    let total = sys.total_swap();
    let used = sys.used_swap();

    let usage_percent = if total > 0 {
        (used as f32 / total as f32) * 100.0
    } else {
        0.0
    };

    SwapInfo {
        total_bytes: total,
        used_bytes: used,
        usage_percent,
    }
}

/// Collect disk usage, filtering out virtual filesystems.
fn collect_disks() -> Vec<DiskInfo> {
    let disks = Disks::new_with_refreshed_list();
    disks
        .list()
        .iter()
        .filter(|d| {
            let mount = d.mount_point().to_string_lossy();
            !SKIP_MOUNT_PREFIXES
                .iter()
                .any(|prefix| mount.starts_with(prefix))
        })
        .map(|d| {
            let total = d.total_space();
            let available = d.available_space();
            let used = total.saturating_sub(available);
            let usage_percent = if total > 0 {
                (used as f32 / total as f32) * 100.0
            } else {
                0.0
            };

            DiskInfo {
                mount_point: d.mount_point().to_string_lossy().to_string(),
                filesystem: d.file_system().to_string_lossy().to_string(),
                total_bytes: total,
                used_bytes: used,
                available_bytes: available,
                usage_percent,
            }
        })
        .collect()
}

/// Collect GPU info via nvidia-smi (graceful fallback if unavailable).
fn collect_gpus() -> Vec<GpuInfo> {
    parse_nvidia_smi().unwrap_or_default()
}

/// Parse nvidia-smi output for GPU metrics.
fn parse_nvidia_smi() -> Option<Vec<GpuInfo>> {
    let output = Command::new("nvidia-smi")
        .args([
            "--query-gpu=name,memory.used,memory.total,utilization.gpu,temperature.gpu",
            "--format=csv,noheader,nounits",
        ])
        .output()
        .ok()?;

    if !output.status.success() {
        return None;
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let gpus = stdout
        .lines()
        .filter(|line| !line.trim().is_empty())
        .filter_map(parse_nvidia_smi_line)
        .collect::<Vec<_>>();

    Some(gpus)
}

/// Parse a single line of nvidia-smi CSV output.
fn parse_nvidia_smi_line(line: &str) -> Option<GpuInfo> {
    let parts: Vec<&str> = line.split(',').map(|s| s.trim()).collect();
    if parts.len() < 5 {
        return None;
    }

    let name = parts[0].to_string();
    let memory_used_mb = parts[1].parse::<u64>().ok()?;
    let memory_total_mb = parts[2].parse::<u64>().ok()?;
    let utilization_percent = parts[3].parse::<f32>().ok()?;
    let temperature_celsius = parts[4].parse::<f32>().ok()?;

    let memory_usage_percent = if memory_total_mb > 0 {
        (memory_used_mb as f32 / memory_total_mb as f32) * 100.0
    } else {
        0.0
    };

    Some(GpuInfo {
        name,
        memory_used_mb,
        memory_total_mb,
        memory_usage_percent,
        utilization_percent,
        temperature_celsius,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_system_collector_runs() {
        let mut collector = SystemCollector::new();
        let resources = collector.collect().unwrap();

        // Basic sanity checks
        assert!(resources.cpu.core_count > 0);
        assert!(!resources.cpu.model.is_empty());
        assert!(resources.memory.total_bytes > 0);
        assert!(!resources.collected_at.is_empty());
        assert!(resources.uptime_seconds > 0);
    }

    #[test]
    fn test_collect_cpu() {
        let mut sys = System::new_all();
        sys.refresh_all();
        let cpu = collect_cpu(&sys);

        assert!(cpu.core_count > 0);
        assert_eq!(cpu.per_core_usage.len(), cpu.core_count);
        assert!(cpu.usage_percent >= 0.0);
        assert!(cpu.usage_percent <= 100.0);
    }

    #[test]
    fn test_collect_memory() {
        let mut sys = System::new_all();
        sys.refresh_all();
        let memory = collect_memory(&sys);

        assert!(memory.total_bytes > 0);
        assert!(memory.used_bytes <= memory.total_bytes);
        assert!(memory.usage_percent >= 0.0);
        assert!(memory.usage_percent <= 100.0);
    }

    #[test]
    fn test_collect_swap() {
        let mut sys = System::new_all();
        sys.refresh_all();
        let swap = collect_swap(&sys);

        // Swap may be 0 on some systems
        assert!(swap.usage_percent >= 0.0);
        assert!(swap.usage_percent <= 100.0);
    }

    #[test]
    fn test_collect_disks() {
        let disks = collect_disks();
        // Should have at least one disk (root)
        assert!(!disks.is_empty());

        for disk in &disks {
            assert!(!disk.mount_point.is_empty());
            assert!(disk.usage_percent >= 0.0);
            assert!(disk.usage_percent <= 100.0);
            // Should not contain filtered paths
            for prefix in SKIP_MOUNT_PREFIXES {
                assert!(
                    !disk.mount_point.starts_with(prefix),
                    "disk {} should be filtered",
                    disk.mount_point
                );
            }
        }
    }

    #[test]
    fn test_collect_gpus_graceful() {
        // Should not error even if nvidia-smi is not available
        let gpus = collect_gpus();
        // Just verify it returns without panicking
        assert!(gpus.len() < 100); // sanity
    }

    #[test]
    fn test_parse_nvidia_smi_line_valid() {
        let line = "NVIDIA RTX 4090, 21900, 24000, 82, 72";
        let gpu = parse_nvidia_smi_line(line).unwrap();
        assert_eq!(gpu.name, "NVIDIA RTX 4090");
        assert_eq!(gpu.memory_used_mb, 21900);
        assert_eq!(gpu.memory_total_mb, 24000);
        assert_eq!(gpu.utilization_percent, 82.0);
        assert_eq!(gpu.temperature_celsius, 72.0);
        assert!((gpu.memory_usage_percent - 91.25).abs() < 0.1);
    }

    #[test]
    fn test_parse_nvidia_smi_line_invalid() {
        assert!(parse_nvidia_smi_line("").is_none());
        assert!(parse_nvidia_smi_line("only,two,fields").is_none());
        assert!(parse_nvidia_smi_line("name, not_a_number, 100, 50, 60").is_none());
    }

    #[test]
    fn test_memory_zero_total() {
        // Edge case: zero total should not divide by zero
        let mut sys = System::new();
        // We can't easily set memory to 0, but we can test the formula directly
        let total: u64 = 0;
        let used: u64 = 0;
        let percent = if total > 0 {
            (used as f32 / total as f32) * 100.0
        } else {
            0.0
        };
        assert_eq!(percent, 0.0);
    }
}
