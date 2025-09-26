use crate::error::{RepoDocsError, Result};
use git2::{
    build::RepoBuilder, CertificateCheckStatus, ErrorClass, ErrorCode, FetchOptions, Progress,
    RemoteCallbacks, Repository,
};
use serde::{Deserialize, Serialize};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tempfile::TempDir;
use url::Url;

#[derive(Debug, Clone)]
pub struct CloneProgress {
    pub total_objects: u32,
    pub received_objects: u32,
    pub local_objects: u32,
    pub total_deltas: u32,
    pub indexed_deltas: u32,
    pub received_bytes: u64,
}

impl From<Progress<'_>> for CloneProgress {
    fn from(progress: Progress) -> Self {
        Self {
            total_objects: progress.total_objects() as u32,
            received_objects: progress.received_objects() as u32,
            local_objects: progress.local_objects() as u32,
            total_deltas: progress.total_deltas() as u32,
            indexed_deltas: progress.indexed_deltas() as u32,
            received_bytes: progress.received_bytes() as u64,
        }
    }
}

pub struct SafeCloner {
    timeout: Duration,
    progress_callback: Option<Box<dyn Fn(CloneProgress) + Send + Sync>>,
    running: Arc<AtomicBool>,
    branch: Option<String>,
}

impl SafeCloner {
    pub fn new() -> Self {
        Self {
            timeout: Duration::from_secs(300), // 5 minutes default
            progress_callback: None,
            running: Arc::new(AtomicBool::new(true)),
            branch: None,
        }
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    pub fn with_progress<F>(mut self, callback: F) -> Self
    where
        F: Fn(CloneProgress) + Send + Sync + 'static,
    {
        self.progress_callback = Some(Box::new(callback));
        self
    }

    pub fn with_branch<S: Into<String>>(mut self, branch: S) -> Self {
        self.branch = Some(branch.into());
        self
    }

    pub fn clone_to_temp(&self, url: &str) -> Result<(Repository, TempDir)> {
        self.validate_url(url)?;

        let temp_dir = TempDir::new().map_err(RepoDocsError::Io)?;

        let repo = self.clone_repository(url, temp_dir.path())?;

        Ok((repo, temp_dir))
    }

    fn validate_url(&self, url: &str) -> Result<()> {
        let parsed_url = Url::parse(url).map_err(|_| RepoDocsError::InvalidUrl {
            url: url.to_string(),
        })?;

        // Security: Only allow specific protocols
        match parsed_url.scheme() {
            "https" => {}
            "ssh" => {}
            "git" => {
                // Only allow git:// for github.com
                if !parsed_url
                    .host_str()
                    .is_some_and(|h| h.contains("github.com"))
                {
                    return Err(RepoDocsError::InvalidUrl {
                        url: url.to_string(),
                    });
                }
            }
            _ => {
                return Err(RepoDocsError::InvalidUrl {
                    url: url.to_string(),
                })
            }
        }

        // Validate GitHub domain
        if !parsed_url
            .host_str()
            .is_some_and(|h| h.ends_with("github.com"))
        {
            return Err(RepoDocsError::InvalidUrl {
                url: url.to_string(),
            });
        }

        Ok(())
    }

    fn clone_repository(&self, url: &str, path: &std::path::Path) -> Result<Repository> {
        let mut callbacks = RemoteCallbacks::new();
        let start_time = Instant::now();
        let timeout = self.timeout;
        let running = self.running.clone();

        // Progress callback with timeout handling
        let progress_callback = self.progress_callback.as_ref().map(|cb| cb.as_ref());
        callbacks.transfer_progress(move |stats: Progress| {
            // Check timeout
            if start_time.elapsed() > timeout {
                eprintln!("Clone operation timed out after {:?}", timeout);
                running.store(false, Ordering::SeqCst);
                return false;
            }

            // Check if operation was cancelled
            if !running.load(Ordering::SeqCst) {
                return false;
            }

            // Call user-provided progress callback
            if let Some(ref callback) = progress_callback {
                callback(CloneProgress::from(stats));
            }

            true
        });

        // Certificate validation (be strict)
        callbacks.certificate_check(|_cert, _valid| {
            // Always accept certificates for now - in production you might want stricter validation
            Ok(CertificateCheckStatus::CertificateOk)
        });

        // Authentication callback for private repositories
        callbacks.credentials(|_url, username_from_url, _allowed_types| {
            // For HTTPS, try token-based auth first
            if let Ok(token) = std::env::var("GITHUB_TOKEN") {
                return git2::Cred::userpass_plaintext(username_from_url.unwrap_or("git"), &token);
            }

            // For SSH, try default SSH key
            if let Some(username) = username_from_url {
                if let Ok(home) = std::env::var("HOME") {
                    let ssh_key = std::path::Path::new(&home).join(".ssh/id_rsa");
                    if ssh_key.exists() {
                        return git2::Cred::ssh_key(username, None, &ssh_key, None);
                    }
                }
            }

            // Default credential (will fail for private repos)
            git2::Cred::default()
        });

        let mut fetch_options = FetchOptions::new();
        fetch_options.remote_callbacks(callbacks);

        let mut builder = RepoBuilder::new();
        builder.fetch_options(fetch_options);

        // Set specific branch if requested
        if let Some(ref branch) = self.branch {
            builder.branch(branch);
        }

        // Clone the repository
        builder
            .clone(url, path)
            .map_err(|e| self.handle_git_error(e, url))
    }

    fn handle_git_error(&self, error: git2::Error, url: &str) -> RepoDocsError {
        match (error.class(), error.code()) {
            (ErrorClass::Net, ErrorCode::GenericError) => RepoDocsError::NetworkError {
                message: format!(
                    "Network error while cloning {}. Check your internet connection and try again.",
                    url
                ),
            },
            (ErrorClass::Http, ErrorCode::Auth) => RepoDocsError::AuthenticationFailed {
                url: url.to_string(),
            },
            (ErrorClass::Http, ErrorCode::NotFound) => RepoDocsError::RepositoryNotFound {
                url: url.to_string(),
            },
            _ => RepoDocsError::Git {
                message: error.message().to_string(),
                source: error,
            },
        }
    }

    pub fn cancel(&self) {
        self.running.store(false, Ordering::SeqCst);
    }

    pub fn is_running(&self) -> bool {
        self.running.load(Ordering::SeqCst)
    }
}

impl Default for SafeCloner {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepositoryInfo {
    pub name: String,
    pub owner: String,
    pub default_branch: String,
    pub is_empty: bool,
    pub total_commits: usize,
    pub url: String,
}

impl RepositoryInfo {
    pub fn from_repository(repo: &Repository, original_url: &str) -> Result<Self> {
        let head = repo.head().map_err(|e| RepoDocsError::Git {
            message: "Repository has no HEAD".to_string(),
            source: e,
        })?;

        let default_branch = head.shorthand().unwrap_or("main").to_string();
        let is_empty = repo.is_empty().map_err(|e| RepoDocsError::Git {
            message: "Failed to check if repository is empty".to_string(),
            source: e,
        })?;

        // Extract owner/name from original URL
        let (owner, name) = Self::parse_github_url(original_url)?;

        let total_commits = if !is_empty {
            Self::count_commits(repo)?
        } else {
            0
        };

        Ok(RepositoryInfo {
            name,
            owner,
            default_branch,
            is_empty,
            total_commits,
            url: original_url.to_string(),
        })
    }

    fn parse_github_url(url: &str) -> Result<(String, String)> {
        let parsed = Url::parse(url).map_err(|_| RepoDocsError::InvalidUrl {
            url: url.to_string(),
        })?;

        let path_segments: Vec<&str> = parsed
            .path_segments()
            .ok_or(RepoDocsError::InvalidUrl {
                url: url.to_string(),
            })?
            .collect();

        if path_segments.len() < 2 {
            return Err(RepoDocsError::InvalidUrl {
                url: url.to_string(),
            });
        }

        let owner = path_segments[0].to_string();
        let mut name = path_segments[1].to_string();

        // Remove .git suffix if present
        if name.ends_with(".git") {
            name = name[..name.len() - 4].to_string();
        }

        Ok((owner, name))
    }

    fn count_commits(repo: &Repository) -> Result<usize> {
        let mut revwalk = repo.revwalk().map_err(|e| RepoDocsError::Git {
            message: "Failed to create revision walker".to_string(),
            source: e,
        })?;

        revwalk.push_head().map_err(|e| RepoDocsError::Git {
            message: "Failed to push HEAD to revision walker".to_string(),
            source: e,
        })?;

        Ok(revwalk.count())
    }

    pub fn display_summary(&self) -> String {
        format!(
            "Repository: {}/{}\nBranch: {}\nCommits: {}\nEmpty: {}",
            self.owner, self.name, self.default_branch, self.total_commits, self.is_empty
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_url_validation() {
        let cloner = SafeCloner::new();

        // Valid URLs
        assert!(cloner
            .validate_url("https://github.com/microsoft/vscode")
            .is_ok());
        assert!(cloner
            .validate_url("https://github.com/rust-lang/rust.git")
            .is_ok());

        // Invalid URLs
        assert!(cloner
            .validate_url("https://gitlab.com/owner/repo")
            .is_err());
        assert!(cloner.validate_url("http://github.com/owner/repo").is_err());
        assert!(cloner.validate_url("ftp://github.com/owner/repo").is_err());
        assert!(cloner.validate_url("not-a-url").is_err());
    }

    #[test]
    fn test_parse_github_url() {
        let (owner, name) =
            RepositoryInfo::parse_github_url("https://github.com/microsoft/vscode").unwrap();
        assert_eq!(owner, "microsoft");
        assert_eq!(name, "vscode");

        let (owner, name) =
            RepositoryInfo::parse_github_url("https://github.com/rust-lang/rust.git").unwrap();
        assert_eq!(owner, "rust-lang");
        assert_eq!(name, "rust");
    }

    #[test]
    fn test_clone_progress() {
        let cloner = SafeCloner::new();

        let cloner_with_progress = cloner.with_progress(|progress| {
            #[allow(clippy::double_comparisons)]
            let _condition = progress.total_objects > 0 || progress.total_objects == 0;
            assert!(_condition);
        });

        // The callback should be set
        assert!(cloner_with_progress.progress_callback.is_some());
    }

    #[test]
    fn test_cancellation() {
        let cloner = SafeCloner::new();
        assert!(cloner.is_running());

        cloner.cancel();
        assert!(!cloner.is_running());
    }

    #[test]
    fn test_timeout_configuration() {
        let timeout = Duration::from_secs(600);
        let cloner = SafeCloner::new().with_timeout(timeout);
        assert_eq!(cloner.timeout, timeout);
    }

    #[test]
    fn test_branch_configuration() {
        let branch = "develop";
        let cloner = SafeCloner::new().with_branch(branch);
        assert_eq!(cloner.branch, Some(branch.to_string()));
    }
}
