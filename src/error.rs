use thiserror::Error;

#[derive(Error, Debug)]
pub enum RepoDocsError {
    #[error("Git operation failed: {message}")]
    Git {
        message: String,
        #[source]
        source: git2::Error,
    },

    #[error("IO operation failed: {0}")]
    Io(#[from] std::io::Error),

    #[error("Invalid repository URL: {url}")]
    InvalidUrl { url: String },

    #[error("Repository not found or inaccessible: {url}")]
    RepositoryNotFound { url: String },

    #[error("Authentication failed for repository: {url}")]
    AuthenticationFailed { url: String },

    #[error("Network error occurred")]
    NetworkError { message: String },

    #[error("No documentation files found in repository")]
    NoDocumentationFound { searched_extensions: Vec<String> },

    #[error("Configuration error: {message}")]
    Config { message: String },

    #[error("Permission denied: {path}")]
    Permission { path: String },

    #[error("Operation was cancelled by user")]
    Cancelled,

    #[error("Operation timed out after {seconds} seconds")]
    Timeout { seconds: u64 },

    #[error("File too large: {size} bytes (max: {max_size} bytes)")]
    FileTooLarge { size: u64, max_size: u64 },

    #[error("Path validation failed: {path}")]
    InvalidPath { path: String },

    #[error("Output directory already exists: {path}")]
    OutputDirectoryExists { path: String },
}

pub trait UserFriendlyError {
    fn user_message(&self) -> String;
    fn suggestion(&self) -> Option<String>;
}

impl UserFriendlyError for RepoDocsError {
    fn user_message(&self) -> String {
        match self {
            RepoDocsError::Git { message, .. } => {
                format!("Git operation failed: {}", message)
            }
            RepoDocsError::InvalidUrl { url } => {
                format!("Invalid repository URL: {}", url)
            }
            RepoDocsError::RepositoryNotFound { url } => {
                format!("Repository not found: {}", url)
            }
            RepoDocsError::AuthenticationFailed { url } => {
                format!("Authentication failed for: {}", url)
            }
            RepoDocsError::NetworkError { message } => {
                format!("Network error: {}", message)
            }
            RepoDocsError::NoDocumentationFound { searched_extensions } => {
                format!(
                    "No documentation files found with extensions: {}",
                    searched_extensions.join(", ")
                )
            }
            RepoDocsError::Config { message } => {
                format!("Configuration error: {}", message)
            }
            RepoDocsError::Permission { path } => {
                format!("Permission denied accessing: {}", path)
            }
            RepoDocsError::Cancelled => {
                "Operation was cancelled by user".to_string()
            }
            RepoDocsError::Timeout { seconds } => {
                format!("Operation timed out after {} seconds", seconds)
            }
            RepoDocsError::FileTooLarge { size, max_size } => {
                format!(
                    "File too large: {} bytes (maximum allowed: {} bytes)",
                    format_bytes(*size),
                    format_bytes(*max_size)
                )
            }
            RepoDocsError::InvalidPath { path } => {
                format!("Invalid file path: {}", path)
            }
            RepoDocsError::OutputDirectoryExists { path } => {
                format!("Output directory already exists: {}", path)
            }
            _ => self.to_string(),
        }
    }

    fn suggestion(&self) -> Option<String> {
        match self {
            RepoDocsError::InvalidUrl { .. } => Some(
                "Please check that the URL is a valid GitHub repository URL (e.g., https://github.com/owner/repo)".to_string()
            ),
            RepoDocsError::RepositoryNotFound { .. } => Some(
                "Verify the repository exists and you have access to it. For private repositories, set the GITHUB_TOKEN environment variable.".to_string()
            ),
            RepoDocsError::AuthenticationFailed { .. } => Some(
                "Set the GITHUB_TOKEN environment variable with a valid personal access token for private repositories.".to_string()
            ),
            RepoDocsError::NetworkError { .. } => Some(
                "Check your internet connection and try again. If the problem persists, the repository server might be temporarily unavailable.".to_string()
            ),
            RepoDocsError::NoDocumentationFound { .. } => Some(
                "Try using different file extensions with --formats (e.g., --formats md,rst,txt,adoc) or check if the repository contains documentation files.".to_string()
            ),
            RepoDocsError::Config { .. } => Some(
                "Check your configuration file syntax and ensure all required fields are present.".to_string()
            ),
            RepoDocsError::Permission { .. } => Some(
                "Ensure you have the necessary read/write permissions for the target directory.".to_string()
            ),
            RepoDocsError::Timeout { .. } => Some(
                "The operation took longer than expected. Try again or increase the timeout with --timeout.".to_string()
            ),
            RepoDocsError::FileTooLarge { .. } => Some(
                "Increase the maximum file size limit with --max-size or exclude large files.".to_string()
            ),
            RepoDocsError::OutputDirectoryExists { .. } => Some(
                "Remove the existing directory, choose a different output name with --output, or use --force to overwrite.".to_string()
            ),
            _ => None,
        }
    }
}

impl From<git2::Error> for RepoDocsError {
    fn from(error: git2::Error) -> Self {
        use git2::{ErrorClass, ErrorCode};

        match (error.class(), error.code()) {
            (ErrorClass::Net, ErrorCode::GenericError) => RepoDocsError::NetworkError {
                message: "Network connection failed".to_string(),
            },
            (ErrorClass::Http, ErrorCode::Auth) => RepoDocsError::AuthenticationFailed {
                url: "repository".to_string(),
            },
            (ErrorClass::Http, ErrorCode::NotFound) => RepoDocsError::RepositoryNotFound {
                url: "repository".to_string(),
            },
            _ => RepoDocsError::Git {
                message: error.message().to_string(),
                source: error,
            },
        }
    }
}

impl From<url::ParseError> for RepoDocsError {
    fn from(_: url::ParseError) -> Self {
        RepoDocsError::InvalidUrl {
            url: "invalid URL".to_string(),
        }
    }
}

impl From<toml::de::Error> for RepoDocsError {
    fn from(error: toml::de::Error) -> Self {
        RepoDocsError::Config {
            message: error.to_string(),
        }
    }
}

pub type Result<T> = std::result::Result<T, RepoDocsError>;

fn format_bytes(bytes: u64) -> String {
    const UNITS: &[&str] = &["B", "KB", "MB", "GB"];
    let mut size = bytes as f64;
    let mut unit_index = 0;

    while size >= 1024.0 && unit_index < UNITS.len() - 1 {
        size /= 1024.0;
        unit_index += 1;
    }

    if unit_index == 0 {
        format!("{} {}", bytes, UNITS[unit_index])
    } else {
        format!("{:.1} {}", size, UNITS[unit_index])
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_user_friendly_messages() {
        let error = RepoDocsError::InvalidUrl {
            url: "not-a-url".to_string(),
        };
        assert!(error.user_message().contains("Invalid repository URL"));
        assert!(error.suggestion().is_some());
    }

    #[test]
    fn test_format_bytes() {
        assert_eq!(format_bytes(1024), "1.0 KB");
        assert_eq!(format_bytes(1048576), "1.0 MB");
        assert_eq!(format_bytes(500), "500 B");
    }

    #[test]
    fn test_git_error_conversion() {
        let git_error = git2::Error::from_str("test error");
        let repo_error = RepoDocsError::from(git_error);
        matches!(repo_error, RepoDocsError::Git { .. });
    }
}