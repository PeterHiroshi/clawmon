use std::path::{Path, PathBuf};

use chrono::{DateTime, TimeZone, Utc};
use git2::Repository;

use crate::config::MAX_RECENT_COMMITS;
use crate::error::Result;
use crate::models::{CommitInfo, GitStatus};

/// Collects git repository status information.
pub struct GitCollector {
    repo_path: PathBuf,
}

impl GitCollector {
    /// Create a new git collector for the given repository path.
    pub fn new(repo_path: &Path) -> Self {
        Self {
            repo_path: repo_path.to_path_buf(),
        }
    }

    /// Collect current git status for the repository.
    pub fn collect(&self) -> Result<GitStatus> {
        let repo = Repository::open(&self.repo_path)?;

        let branch = current_branch(&repo);
        let (is_clean, uncommitted_files) = repo_status(&repo)?;
        let (ahead, behind) = ahead_behind(&repo)?;
        let recent_commits = recent_commits(&repo, MAX_RECENT_COMMITS)?;
        let last_commit = recent_commits.first().cloned();
        let last_push_time = last_push_time(&repo);

        Ok(GitStatus {
            branch,
            is_clean,
            uncommitted_files,
            ahead,
            behind,
            last_commit,
            last_push_time,
            recent_commits,
        })
    }
}

/// Get the current branch name.
fn current_branch(repo: &Repository) -> String {
    repo.head()
        .ok()
        .and_then(|head| head.shorthand().map(String::from))
        .unwrap_or_else(|| "HEAD (detached)".to_string())
}

/// Check repo status: clean/dirty and list uncommitted files.
fn repo_status(repo: &Repository) -> Result<(bool, Vec<String>)> {
    let statuses = repo.statuses(Some(
        git2::StatusOptions::new()
            .include_untracked(true)
            .recurse_untracked_dirs(true),
    ))?;

    let mut files = Vec::new();
    for entry in statuses.iter() {
        if let Some(path) = entry.path() {
            files.push(path.to_string());
        }
    }

    Ok((files.is_empty(), files))
}

/// Calculate ahead/behind counts relative to tracking branch.
fn ahead_behind(repo: &Repository) -> Result<(usize, usize)> {
    let head = match repo.head() {
        Ok(h) => h,
        Err(_) => return Ok((0, 0)),
    };

    let local_oid = match head.target() {
        Some(oid) => oid,
        None => return Ok((0, 0)),
    };

    let branch_name = match head.shorthand() {
        Some(name) => name.to_string(),
        None => return Ok((0, 0)),
    };

    let upstream_ref = format!("refs/remotes/origin/{}", branch_name);
    let upstream_oid = match repo.refname_to_id(&upstream_ref) {
        Ok(oid) => oid,
        Err(_) => return Ok((0, 0)),
    };

    let (ahead, behind) = repo.graph_ahead_behind(local_oid, upstream_oid)?;
    Ok((ahead, behind))
}

/// Get recent commits from HEAD.
fn recent_commits(repo: &Repository, max: usize) -> Result<Vec<CommitInfo>> {
    let head = match repo.head() {
        Ok(h) => h,
        Err(_) => return Ok(vec![]),
    };

    let oid = match head.target() {
        Some(oid) => oid,
        None => return Ok(vec![]),
    };

    let mut revwalk = repo.revwalk()?;
    revwalk.push(oid)?;
    revwalk.set_sorting(git2::Sort::TIME)?;

    let mut commits = Vec::new();
    for oid_result in revwalk.take(max) {
        let oid = oid_result?;
        let commit = repo.find_commit(oid)?;
        commits.push(commit_to_info(&commit));
    }

    Ok(commits)
}

/// Convert a git2 Commit to our CommitInfo model.
fn commit_to_info(commit: &git2::Commit<'_>) -> CommitInfo {
    let hash = commit.id().to_string();
    let short_hash = hash[..7.min(hash.len())].to_string();
    let message = commit
        .message()
        .unwrap_or("")
        .lines()
        .next()
        .unwrap_or("")
        .to_string();
    let author = commit.author().name().unwrap_or("Unknown").to_string();
    let time = Utc
        .timestamp_opt(commit.time().seconds(), 0)
        .single()
        .unwrap_or_else(Utc::now);

    CommitInfo {
        hash,
        short_hash,
        message,
        author,
        time,
    }
}

/// Try to get the last push time from reflog.
fn last_push_time(repo: &Repository) -> Option<DateTime<Utc>> {
    let head = repo.head().ok()?;
    let branch_name = head.shorthand()?;
    let remote_ref = format!("refs/remotes/origin/{}", branch_name);

    let reflog = repo.reflog(&remote_ref).ok()?;

    reflog.iter().next().and_then(|entry| {
        Utc.timestamp_opt(entry.committer().when().seconds(), 0)
            .single()
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::TempDir;

    fn init_test_repo(dir: &Path) -> Repository {
        let repo = Repository::init(dir).unwrap();

        // Configure user for commits
        let mut config = repo.config().unwrap();
        config.set_str("user.name", "Test User").unwrap();
        config.set_str("user.email", "test@example.com").unwrap();

        repo
    }

    fn make_commit(repo: &Repository, dir: &Path, filename: &str, message: &str) {
        let file_path = dir.join(filename);
        fs::write(&file_path, "content").unwrap();

        let mut index = repo.index().unwrap();
        index.add_path(Path::new(filename)).unwrap();
        index.write().unwrap();
        let tree_id = index.write_tree().unwrap();
        let tree = repo.find_tree(tree_id).unwrap();
        let sig = repo.signature().unwrap();

        let parent = repo.head().ok().and_then(|h| h.peel_to_commit().ok());
        let parents: Vec<&git2::Commit<'_>> = parent.iter().collect();

        repo.commit(Some("HEAD"), &sig, &sig, message, &tree, &parents)
            .unwrap();
    }

    #[test]
    fn test_collect_empty_repo() {
        let tmp = TempDir::new().unwrap();
        let _repo = init_test_repo(tmp.path());
        let collector = GitCollector::new(tmp.path());
        let status = collector.collect().unwrap();
        assert!(status.recent_commits.is_empty());
    }

    #[test]
    fn test_collect_repo_with_commits() {
        let tmp = TempDir::new().unwrap();
        let repo = init_test_repo(tmp.path());
        make_commit(&repo, tmp.path(), "file1.txt", "first commit");
        make_commit(&repo, tmp.path(), "file2.txt", "second commit");

        let collector = GitCollector::new(tmp.path());
        let status = collector.collect().unwrap();

        assert_eq!(status.recent_commits.len(), 2);
        assert_eq!(status.recent_commits[0].message, "second commit");
        assert_eq!(status.recent_commits[1].message, "first commit");
        assert!(status.last_commit.is_some());
        assert_eq!(status.last_commit.unwrap().message, "second commit");
    }

    #[test]
    fn test_clean_vs_dirty_repo() {
        let tmp = TempDir::new().unwrap();
        let repo = init_test_repo(tmp.path());
        make_commit(&repo, tmp.path(), "file1.txt", "initial");

        let collector = GitCollector::new(tmp.path());

        // Should be clean after commit
        let status = collector.collect().unwrap();
        assert!(status.is_clean);
        assert!(status.uncommitted_files.is_empty());

        // Make it dirty
        fs::write(tmp.path().join("untracked.txt"), "new file").unwrap();
        let status = collector.collect().unwrap();
        assert!(!status.is_clean);
        assert!(
            status
                .uncommitted_files
                .contains(&"untracked.txt".to_string())
        );
    }

    #[test]
    fn test_branch_name() {
        let tmp = TempDir::new().unwrap();
        let repo = init_test_repo(tmp.path());
        make_commit(&repo, tmp.path(), "file.txt", "init");

        let branch = current_branch(&repo);
        // Default branch is usually "master" or "main" depending on git config
        assert!(!branch.is_empty());
    }

    #[test]
    fn test_ahead_behind_no_remote() {
        let tmp = TempDir::new().unwrap();
        let repo = init_test_repo(tmp.path());
        make_commit(&repo, tmp.path(), "file.txt", "init");

        let (ahead, behind) = ahead_behind(&repo).unwrap();
        assert_eq!(ahead, 0);
        assert_eq!(behind, 0);
    }

    #[test]
    fn test_commit_info_fields() {
        let tmp = TempDir::new().unwrap();
        let repo = init_test_repo(tmp.path());
        make_commit(&repo, tmp.path(), "file.txt", "test message");

        let commits = recent_commits(&repo, 10).unwrap();
        assert_eq!(commits.len(), 1);
        assert_eq!(commits[0].message, "test message");
        assert_eq!(commits[0].author, "Test User");
        assert_eq!(commits[0].short_hash.len(), 7);
        assert!(commits[0].hash.len() >= 40);
    }
}
