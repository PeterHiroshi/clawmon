use std::path::{Path, PathBuf};
use std::sync::Arc;

use notify::{Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use tokio::sync::mpsc;

use crate::error::Result;

/// File change event with filtered path info.
#[derive(Debug, Clone)]
pub struct FileChangeEvent {
    /// Path that changed.
    pub path: PathBuf,
    /// Kind of change.
    pub kind: ChangeKind,
}

/// Simplified change kind.
#[derive(Debug, Clone, PartialEq)]
pub enum ChangeKind {
    /// File or dir created/modified.
    Modified,
    /// File or dir removed.
    Removed,
}

/// Filesystem watcher that monitors workspace paths for changes.
pub struct FileWatcher {
    _watcher: RecommendedWatcher,
}

impl FileWatcher {
    /// Create a new file watcher monitoring the given paths.
    /// Returns the watcher and a channel receiver for change events.
    pub fn new(paths: &[PathBuf]) -> Result<(Self, mpsc::Receiver<FileChangeEvent>)> {
        let (tx, rx) = mpsc::channel(512);
        let tx = Arc::new(tx);

        let tx_clone = tx.clone();
        let mut watcher =
            notify::recommended_watcher(move |res: std::result::Result<Event, notify::Error>| {
                if let Ok(event) = res
                    && let Some(change) = filter_event(&event)
                {
                    let _ = tx_clone.blocking_send(change);
                }
            })?;

        for path in paths {
            if path.is_dir() {
                watcher.watch(path, RecursiveMode::Recursive)?;
            }
        }

        Ok((Self { _watcher: watcher }, rx))
    }
}

/// Filter and convert notify events to our change events.
fn filter_event(event: &Event) -> Option<FileChangeEvent> {
    let path = event.paths.first()?.clone();

    // Skip files in target/ and .git/objects (too noisy)
    let path_str = path.to_string_lossy();
    if path_str.contains("/target/") || path_str.contains("/.git/objects/") {
        return None;
    }

    let kind = match event.kind {
        EventKind::Create(_) | EventKind::Modify(_) => ChangeKind::Modified,
        EventKind::Remove(_) => ChangeKind::Removed,
        _ => return None,
    };

    Some(FileChangeEvent { path, kind })
}

/// Check if a path is relevant for triggering collector refreshes.
pub fn is_relevant_change(path: &Path) -> bool {
    let path_str = path.to_string_lossy();

    // Relevant: .forge-task/ changes, source files, git refs
    path_str.contains(".forge-task")
        || path_str.contains(".git/refs")
        || path_str.contains(".git/HEAD")
        || path_str.ends_with(".rs")
        || path_str.ends_with(".go")
        || path_str.ends_with(".json")
        || path_str.ends_with(".toml")
        || path_str.ends_with(".md")
}

#[cfg(test)]
mod tests {
    use super::*;
    use notify::event::{CreateKind, ModifyKind, RemoveKind};

    #[test]
    fn test_filter_event_create() {
        let event = Event {
            kind: EventKind::Create(CreateKind::File),
            paths: vec![PathBuf::from("/workspace/src/main.rs")],
            attrs: Default::default(),
        };
        let result = filter_event(&event);
        assert!(result.is_some());
        assert_eq!(result.unwrap().kind, ChangeKind::Modified);
    }

    #[test]
    fn test_filter_event_modify() {
        let event = Event {
            kind: EventKind::Modify(ModifyKind::Data(notify::event::DataChange::Content)),
            paths: vec![PathBuf::from("/workspace/file.json")],
            attrs: Default::default(),
        };
        let result = filter_event(&event);
        assert!(result.is_some());
    }

    #[test]
    fn test_filter_event_remove() {
        let event = Event {
            kind: EventKind::Remove(RemoveKind::File),
            paths: vec![PathBuf::from("/workspace/old.rs")],
            attrs: Default::default(),
        };
        let result = filter_event(&event);
        assert!(result.is_some());
        assert_eq!(result.unwrap().kind, ChangeKind::Removed);
    }

    #[test]
    fn test_filter_event_skips_target() {
        let event = Event {
            kind: EventKind::Create(CreateKind::File),
            paths: vec![PathBuf::from("/workspace/target/debug/something")],
            attrs: Default::default(),
        };
        assert!(filter_event(&event).is_none());
    }

    #[test]
    fn test_filter_event_skips_git_objects() {
        let event = Event {
            kind: EventKind::Create(CreateKind::File),
            paths: vec![PathBuf::from("/workspace/.git/objects/ab/cdef12345")],
            attrs: Default::default(),
        };
        assert!(filter_event(&event).is_none());
    }

    #[test]
    fn test_filter_event_empty_paths() {
        let event = Event {
            kind: EventKind::Create(CreateKind::File),
            paths: vec![],
            attrs: Default::default(),
        };
        assert!(filter_event(&event).is_none());
    }

    #[test]
    fn test_is_relevant_change() {
        assert!(is_relevant_change(&PathBuf::from(
            "/ws/.forge-task/progress.md"
        )));
        assert!(is_relevant_change(&PathBuf::from(
            "/ws/.git/refs/heads/main"
        )));
        assert!(is_relevant_change(&PathBuf::from("/ws/src/main.rs")));
        assert!(is_relevant_change(&PathBuf::from("/ws/config.json")));
        assert!(!is_relevant_change(&PathBuf::from("/ws/image.png")));
        assert!(!is_relevant_change(&PathBuf::from("/ws/binary.exe")));
    }
}
