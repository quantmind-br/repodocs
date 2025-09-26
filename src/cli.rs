use crate::config::{CliOverrides, Config};
use crate::error::{RepoDocsError, Result};
use clap::{Parser, ValueEnum};
use std::path::PathBuf;
use url::Url;

#[derive(Parser, Debug)]
#[command(name = "repodocs")]
#[command(version = env!("CARGO_PKG_VERSION"))]
#[command(about = "Extract documentation from GitHub repositories")]
#[command(
    long_about = "RepoDocs clones a GitHub repository and extracts all documentation files \
                       into a local directory for offline browsing and analysis."
)]
#[command(before_help = "🚀 RepoDocs - Documentation Extraction Tool")]
#[command(after_help = "EXAMPLES:\n  \
    repodocs https://github.com/microsoft/vscode\n  \
    repodocs https://github.com/rust-lang/rust --output rust-docs --verbose\n  \
    repodocs https://github.com/facebook/react --formats md,rst --exclude tests,examples\n  \
    repodocs https://github.com/torvalds/linux --config my-config.toml\n\n\
    For more information, visit: https://github.com/user/repodocs")]
#[command(arg_required_else_help = true)]
pub struct Cli {
    /// GitHub repository URL
    #[arg(value_parser = validate_github_url)]
    pub repository_url: String,

    /// Output directory name (defaults to docs_{repo_name})
    #[arg(short, long)]
    pub output: Option<String>,

    /// File formats to extract (comma-separated)
    #[arg(
        short,
        long,
        help = "File extensions to extract (e.g., md,rst,txt,adoc)"
    )]
    pub formats: Option<String>,

    /// Directories to exclude from extraction
    #[arg(short, long, value_delimiter = ',')]
    pub exclude: Option<Vec<String>>,

    /// Maximum file size in MB
    #[arg(long, help = "Maximum file size to process (in MB)")]
    pub max_size: Option<u64>,

    /// Configuration file path
    #[arg(short, long, help = "Path to TOML configuration file")]
    pub config: Option<PathBuf>,

    /// Output format for results
    #[arg(long, value_enum, default_value_t = OutputFormat::Human)]
    pub output_format: OutputFormat,

    /// Preserve directory structure in output
    #[arg(long, help = "Preserve original directory structure")]
    pub preserve_structure: Option<bool>,

    /// Git clone timeout in seconds
    #[arg(long, help = "Timeout for git clone operation (seconds)")]
    pub timeout: Option<u64>,

    /// Specific git branch to clone
    #[arg(
        short,
        long,
        help = "Specific branch to clone (default: repository default)"
    )]
    pub branch: Option<String>,

    /// Verbose output level (-v, -vv, -vvv)
    #[arg(short, long, action = clap::ArgAction::Count)]
    pub verbose: u8,

    /// Quiet mode (suppress non-essential output)
    #[arg(short, long, conflicts_with = "verbose")]
    pub quiet: bool,

    /// Force overwrite of existing output directory
    #[arg(long, help = "Overwrite existing output directory")]
    pub force: bool,

    /// Dry run (show what would be done without executing)
    #[arg(long, help = "Show what would be extracted without actually doing it")]
    pub dry_run: bool,

    /// Generate sample configuration file
    #[arg(long, help = "Generate a sample configuration file")]
    pub generate_config: bool,
}

#[derive(Debug, Clone, ValueEnum)]
pub enum OutputFormat {
    /// Human-readable colored output
    Human,
    /// JSON formatted output
    Json,
    /// Plain text output
    Plain,
}

impl Cli {
    pub fn load_config(&self) -> Result<Config> {
        let mut config = Config::load_with_defaults(self.config.as_ref())?;

        let overrides = self.create_cli_overrides();
        config.merge_with_cli_args(&overrides);
        config.validate()?;

        Ok(config)
    }

    pub fn create_cli_overrides(&self) -> CliOverrides {
        let output_dir = self.output.as_ref().map(|o| {
            if o.contains('/') || o.contains('\\') {
                PathBuf::from(o)
            } else {
                std::env::current_dir().unwrap_or_default().join(o)
            }
        });

        let max_file_size = self.max_size.map(|size| size * 1024 * 1024); // Convert MB to bytes

        CliOverrides::new()
            .with_formats(self.formats.clone())
            .with_exclude(self.exclude.clone())
            .with_max_file_size(max_file_size)
            .with_output_dir(output_dir)
            .with_preserve_structure(self.preserve_structure)
            .with_timeout(self.timeout)
            .with_branch(self.branch.clone())
    }

    pub fn extract_repo_info(&self) -> Result<(String, String)> {
        let url = Url::parse(&self.repository_url)?;
        let path_segments: Vec<&str> = url
            .path_segments()
            .ok_or(RepoDocsError::InvalidUrl {
                url: self.repository_url.clone(),
            })?
            .collect();

        if path_segments.len() < 2 {
            return Err(RepoDocsError::InvalidUrl {
                url: self.repository_url.clone(),
            });
        }

        let owner = path_segments[0].to_string();
        let mut repo_name = path_segments[1].to_string();

        // Remove .git suffix if present
        if repo_name.ends_with(".git") {
            repo_name = repo_name[..repo_name.len() - 4].to_string();
        }

        Ok((owner, repo_name))
    }

    pub fn get_output_directory_name(&self) -> Result<String> {
        if let Some(ref output) = self.output {
            Ok(output.clone())
        } else {
            let (_, repo_name) = self.extract_repo_info()?;
            Ok(format!("docs_{}", repo_name))
        }
    }

    pub fn should_use_colors(&self) -> bool {
        !self.quiet && console::Term::stdout().features().colors_supported()
    }

    pub fn is_verbose(&self) -> bool {
        self.verbose > 0 && !self.quiet
    }

    pub fn verbosity_level(&self) -> u8 {
        if self.quiet {
            0
        } else {
            self.verbose
        }
    }
}

pub fn validate_github_url(s: &str) -> std::result::Result<String, String> {
    // Parse URL
    let url =
        Url::parse(s).map_err(|_| "Invalid URL format. Please provide a valid URL.".to_string())?;

    // Security: Only allow specific schemes
    match url.scheme() {
        "https" => {}
        "ssh" => {}
        "git" => {
            // Only allow git:// for github.com (public repos)
            if !url.host_str().is_some_and(|h| h.contains("github.com")) {
                return Err("git:// protocol only allowed for github.com".to_string());
            }
        }
        _ => {
            return Err(
                "Only HTTPS, SSH, and git:// protocols are supported for security reasons"
                    .to_string(),
            )
        }
    }

    // Security: Only allow GitHub URLs
    let host = url
        .host_str()
        .ok_or("URL must include a valid hostname".to_string())?;

    if !host.ends_with("github.com") {
        return Err(
            "Only GitHub URLs are supported (e.g., github.com, api.github.com)".to_string(),
        );
    }

    // Validate path structure
    let path_segments: Vec<&str> = url
        .path_segments()
        .ok_or("Invalid repository path".to_string())?
        .collect();

    if path_segments.len() < 2 {
        return Err(
            "URL must include owner/repository (e.g., https://github.com/owner/repo)".to_string(),
        );
    }

    // Validate owner and repo names
    let owner = path_segments[0];
    let repo = path_segments[1];

    if owner.is_empty() || repo.is_empty() {
        return Err("Both owner and repository names must be non-empty".to_string());
    }

    // Security: Validate characters in owner and repo names
    let valid_chars = |s: &str| {
        s.chars()
            .all(|c| c.is_alphanumeric() || c == '-' || c == '_' || c == '.')
    };

    if !valid_chars(owner) {
        return Err("Owner name contains invalid characters. Only alphanumeric, hyphens, underscores, and dots are allowed.".to_string());
    }

    let repo_name = repo.strip_suffix(".git").unwrap_or(repo);

    if !valid_chars(repo_name) {
        return Err("Repository name contains invalid characters. Only alphanumeric, hyphens, underscores, and dots are allowed.".to_string());
    }

    // Prevent common attack patterns
    if owner.starts_with('.') || repo_name.starts_with('.') {
        return Err("Owner and repository names cannot start with a dot".to_string());
    }

    // Length validation
    if owner.len() > 100 || repo_name.len() > 100 {
        return Err("Owner and repository names must be 100 characters or less".to_string());
    }

    Ok(s.to_string())
}

pub fn parse_size_string(s: &str) -> std::result::Result<u64, String> {
    let s = s.trim().to_lowercase();

    let (number_str, multiplier) = if s.ends_with("kb") || s.ends_with("k") {
        (s.trim_end_matches("kb").trim_end_matches("k"), 1024)
    } else if s.ends_with("mb") || s.ends_with("m") {
        (s.trim_end_matches("mb").trim_end_matches("m"), 1024 * 1024)
    } else if s.ends_with("gb") || s.ends_with("g") {
        (
            s.trim_end_matches("gb").trim_end_matches("g"),
            1024 * 1024 * 1024,
        )
    } else if s.ends_with("b") {
        (s.trim_end_matches("b"), 1)
    } else {
        (s.as_str(), 1)
    };

    let number: f64 = number_str
        .parse()
        .map_err(|_| format!("Invalid number format: {}", number_str))?;

    if number < 0.0 {
        return Err("Size cannot be negative".to_string());
    }

    Ok((number * multiplier as f64) as u64)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_github_urls() {
        let valid_urls = [
            "https://github.com/microsoft/vscode",
            "https://github.com/rust-lang/rust.git",
            "https://github.com/facebook/react",
        ];

        for url in &valid_urls {
            assert!(validate_github_url(url).is_ok(), "Should accept: {}", url);
        }
    }

    #[test]
    fn test_invalid_github_urls() {
        let invalid_urls = [
            "https://gitlab.com/owner/repo",
            "http://github.com/owner/repo", // http not allowed
            "https://github.com/owner",     // missing repo
            "https://github.com/",          // missing owner and repo
            "not-a-url",
            "ftp://github.com/owner/repo", // wrong protocol
        ];

        for url in &invalid_urls {
            assert!(validate_github_url(url).is_err(), "Should reject: {}", url);
        }
    }

    #[test]
    fn test_extract_repo_info() {
        let cli = Cli {
            repository_url: "https://github.com/microsoft/vscode".to_string(),
            output: None,
            formats: None,
            exclude: None,
            max_size: None,
            config: None,
            output_format: OutputFormat::Human,
            preserve_structure: None,
            timeout: None,
            branch: None,
            verbose: 0,
            quiet: false,
            force: false,
            dry_run: false,
            generate_config: false,
        };

        let (owner, repo) = cli.extract_repo_info().unwrap();
        assert_eq!(owner, "microsoft");
        assert_eq!(repo, "vscode");
    }

    #[test]
    fn test_parse_size_string() {
        assert_eq!(parse_size_string("10").unwrap(), 10);
        assert_eq!(parse_size_string("10KB").unwrap(), 10 * 1024);
        assert_eq!(parse_size_string("5MB").unwrap(), 5 * 1024 * 1024);
        assert_eq!(parse_size_string("1GB").unwrap(), 1024 * 1024 * 1024);

        assert!(parse_size_string("invalid").is_err());
        assert!(parse_size_string("-5MB").is_err());
    }

    #[test]
    fn test_output_directory_generation() {
        let cli = Cli {
            repository_url: "https://github.com/rust-lang/book".to_string(),
            output: None,
            formats: None,
            exclude: None,
            max_size: None,
            config: None,
            output_format: OutputFormat::Human,
            preserve_structure: None,
            timeout: None,
            branch: None,
            verbose: 0,
            quiet: false,
            force: false,
            dry_run: false,
            generate_config: false,
        };

        assert_eq!(cli.get_output_directory_name().unwrap(), "docs_book");
    }
}
