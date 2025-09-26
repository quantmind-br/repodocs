pub mod cli;
pub mod config;
pub mod error;
pub mod cloner;
pub mod scanner;
pub mod extractor;
pub mod ui;

// Public API re-exports
pub use cli::{Cli, OutputFormat};
pub use config::{Config, FilterConfig, OutputConfig, GitConfig, CliOverrides};
pub use error::{RepoDocsError, UserFriendlyError, Result};

// Core functionality re-exports
pub use cloner::{SafeCloner, RepositoryInfo, CloneProgress};
pub use scanner::{DocumentScanner, DocumentFile, FileFilter};
pub use extractor::{FileOperations, OutputManager, ExtractionProgress, ExtractionReport, ConfigSnapshot};
pub use ui::{ProgressManager, OutputFormatter, OutputMode, GracefulShutdown};

use std::path::Path;
use std::time::Instant;
use tokio::task;

/// Main library interface for RepoDocs functionality
pub struct RepoDocs {
    config: Config,
    output_formatter: OutputFormatter,
    progress_manager: ProgressManager,
    shutdown: GracefulShutdown,
}

impl RepoDocs {
    /// Create a new RepoDocs instance with the provided configuration
    pub fn new(config: Config, output_mode: OutputMode, verbose: u8, quiet: bool) -> Result<Self> {
        let output_formatter = OutputFormatter::new(output_mode, verbose, quiet);
        let progress_manager = ProgressManager::new(!quiet);
        let shutdown = GracefulShutdown::new()?;

        Ok(Self {
            config,
            output_formatter,
            progress_manager,
            shutdown,
        })
    }

    /// Create a new RepoDocs instance for testing (no signal handler conflicts)
    #[cfg(test)]
    pub fn new_for_test(config: Config, output_mode: OutputMode, verbose: u8, quiet: bool) -> Self {
        let output_formatter = OutputFormatter::new(output_mode, verbose, quiet);
        let progress_manager = ProgressManager::new(!quiet);
        let shutdown = GracefulShutdown::new_for_test();

        Self {
            config,
            output_formatter,
            progress_manager,
            shutdown,
        }
    }

    /// Create RepoDocs instance from CLI arguments
    pub fn from_cli(cli_args: &Cli) -> Result<Self> {
        let config = cli_args.load_config()?;
        let output_mode = match cli_args.output_format {
            crate::cli::OutputFormat::Human => OutputMode::Human,
            crate::cli::OutputFormat::Json => OutputMode::Json,
            crate::cli::OutputFormat::Plain => OutputMode::Plain,
        };

        Self::new(config, output_mode, cli_args.verbose, cli_args.quiet)
    }

    /// Extract documentation from a repository URL
    pub async fn extract_documentation(&self, repository_url: &str) -> Result<ExtractionReport> {
        let _start_time = Instant::now();

        // Validate the operation can proceed
        self.shutdown.check_shutdown()?;

        self.output_formatter.start_operation("Starting documentation extraction");

        // Step 1: Clone repository
        let (_repo, temp_dir, repo_info) = self.clone_repository(repository_url).await?;
        self.shutdown.check_shutdown()?;

        // Step 2: Scan for documentation files
        let documents = self.scan_documentation(&temp_dir.path())?;
        self.shutdown.check_shutdown()?;

        if documents.is_empty() {
            return Err(RepoDocsError::NoDocumentationFound {
                searched_extensions: self.config.filters.extensions.clone(),
            });
        }

        self.output_formatter.info(&format!("Found {} documentation files", documents.len()));

        // Step 3: Setup output directory
        let output_manager = self.setup_output_directory(&repo_info)?;
        self.shutdown.check_shutdown()?;

        // Step 4: Extract files
        let extraction_progress = self.extract_files(&documents, output_manager.get_output_directory())?;
        self.shutdown.check_shutdown()?;

        // Step 5: Generate reports
        let config_snapshot = self.create_config_snapshot();
        let report = output_manager.create_extraction_report(
            &repo_info,
            &documents,
            &extraction_progress,
            &config_snapshot,
        )?;

        // Step 6: Create index file if requested
        if self.config.output.create_index {
            let file_ops = FileOperations::new().with_preserve_structure(self.config.output.preserve_structure);
            file_ops.create_index_file(&documents, output_manager.get_output_directory())?;
        }

        // Display summary
        self.output_formatter.print_extraction_summary(&extraction_progress);

        Ok(report)
    }

    /// Clone repository with progress indication
    async fn clone_repository(&self, url: &str) -> Result<(git2::Repository, tempfile::TempDir, RepositoryInfo)> {
        self.output_formatter.start_operation("Cloning repository");

        let clone_progress = self.progress_manager.create_clone_progress();
        let progress_callback = {
            let pb = clone_progress.clone();
            move |progress: CloneProgress| {
                ui::progress::update_clone_progress(&pb, &progress);
            }
        };

        let cloner = SafeCloner::new()
            .with_timeout(self.config.git_timeout_duration())
            .with_progress(progress_callback);

        let cloner = if let Some(ref branch) = self.config.git.branch {
            cloner.with_branch(branch)
        } else {
            cloner
        };

        let url_clone = url.to_string();
        let (repo, temp_dir) = task::spawn_blocking(move || {
            cloner.clone_to_temp(&url_clone)
        }).await
        .map_err(|e| RepoDocsError::Config {
            message: format!("Clone task failed: {}", e),
        })??;

        ui::progress::finish_progress_with_summary(
            &clone_progress,
            "Repository cloned successfully",
            clone_progress.elapsed(),
        );

        let repo_info = RepositoryInfo::from_repository(&repo, url)?;
        self.output_formatter.debug(&repo_info.display_summary());

        Ok((repo, temp_dir, repo_info))
    }

    /// Scan for documentation files
    fn scan_documentation(&self, repo_path: &Path) -> Result<Vec<DocumentFile>> {
        self.output_formatter.start_operation("Scanning for documentation files");

        let scanner = DocumentScanner::new(&self.config.filters)
            .with_repo_root(repo_path);

        let documents = scanner.scan_directory(repo_path)?;

        // Display scan statistics if verbose
        let stats = scanner.get_statistics(&documents);
        self.output_formatter.debug(&stats.display_summary());

        Ok(documents)
    }

    /// Setup output directory management
    fn setup_output_directory(&self, repo_info: &RepositoryInfo) -> Result<OutputManager> {
        let output_manager = OutputManager::new(
            self.config.output.base_directory.clone(),
            repo_info.name.clone(),
        )?;

        // Configure force overwrite based on CLI arguments (would need to be passed through)
        let manager = output_manager; // .with_force_overwrite(force);

        manager.initialize()?;

        self.output_formatter.success(&format!(
            "Initialized output directory: {}",
            manager.get_output_directory().display()
        ));

        Ok(manager)
    }

    /// Extract files with progress tracking
    fn extract_files(&self, documents: &[DocumentFile], output_dir: &Path) -> Result<ExtractionProgress> {
        self.output_formatter.start_operation("Extracting documentation files");

        let file_progress = self.progress_manager.create_file_progress(documents.len() as u64);
        let progress_callback = {
            let pb = file_progress.clone();
            move |progress: &ExtractionProgress| {
                ui::progress::update_file_progress(&pb, progress);
            }
        };

        let file_ops = FileOperations::new()
            .with_preserve_structure(self.config.output.preserve_structure);

        let extraction_progress = file_ops.extract_files(
            documents,
            output_dir,
            Some(&progress_callback),
        )?;

        ui::progress::finish_progress_with_summary(
            &file_progress,
            &format!("Extracted {} files", extraction_progress.files_processed),
            extraction_progress.elapsed(),
        );

        Ok(extraction_progress)
    }

    /// Create configuration snapshot for reporting
    fn create_config_snapshot(&self) -> ConfigSnapshot {
        ConfigSnapshot {
            extensions: self.config.filters.extensions.clone(),
            max_file_size: self.config.filters.max_file_size,
            exclude_dirs: self.config.filters.exclude_dirs.clone(),
            preserve_structure: self.config.output.preserve_structure,
        }
    }

    /// Generate sample configuration file
    pub fn generate_sample_config<P: AsRef<Path>>(output_path: P) -> Result<()> {
        let sample_config = Config::create_sample_config();
        std::fs::write(output_path.as_ref(), sample_config)
            .map_err(|e| RepoDocsError::Io(e))?;
        Ok(())
    }

    /// Get configuration reference
    pub fn config(&self) -> &Config {
        &self.config
    }

    /// Get output formatter reference
    pub fn output_formatter(&self) -> &OutputFormatter {
        &self.output_formatter
    }

    /// Get progress manager reference
    pub fn progress_manager(&self) -> &ProgressManager {
        &self.progress_manager
    }

    /// Check if shutdown has been requested
    pub fn is_running(&self) -> bool {
        self.shutdown.is_running()
    }

    /// Request graceful shutdown
    pub fn request_shutdown(&self) {
        self.shutdown.request_shutdown();
    }

    /// Handle error with user-friendly output
    pub fn handle_error(&self, error: &RepoDocsError) {
        self.output_formatter.print_user_friendly_error(error);
    }
}

/// Convenience function to extract documentation with minimal setup
pub async fn extract_docs_simple(
    repository_url: &str,
    output_dir: Option<&Path>,
    verbose: bool,
) -> Result<ExtractionReport> {
    let mut config = Config::default();

    if let Some(output_path) = output_dir {
        config.output.base_directory = output_path.to_path_buf();
    }

    let repodocs = RepoDocs::new(
        config,
        OutputMode::Human,
        if verbose { 1 } else { 0 },
        false,
    )?;

    repodocs.extract_documentation(repository_url).await
}

/// Validate a GitHub repository URL
pub fn validate_repository_url(url: &str) -> Result<String> {
    cli::validate_github_url(url)
        .map_err(|msg| RepoDocsError::InvalidUrl { url: msg.to_string() })
}

/// Get version information
pub fn version_info() -> &'static str {
    env!("CARGO_PKG_VERSION")
}

/// Get build information
pub fn build_info() -> BuildInfo {
    BuildInfo {
        version: env!("CARGO_PKG_VERSION"),
        git_hash: option_env!("GIT_HASH").unwrap_or("unknown"),
        build_date: option_env!("BUILD_DATE").unwrap_or("unknown"),
        target: std::env::consts::ARCH.to_string(),
    }
}

#[derive(Debug, Clone)]
pub struct BuildInfo {
    pub version: &'static str,
    pub git_hash: &'static str,
    pub build_date: &'static str,
    pub target: String,
}

impl std::fmt::Display for BuildInfo {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "RepoDocs {} ({}) built on {} for {}",
            self.version, self.git_hash, self.build_date, self.target
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    #[test]
    fn test_repodocs_creation() {
        let config = Config::default();
        let repodocs = RepoDocs::new(config, OutputMode::Human, 1, false);
        assert!(repodocs.is_ok());

        let repodocs = repodocs.unwrap();
        assert!(repodocs.is_running());
        assert_eq!(repodocs.config().filters.extensions.len(), 14); // Default extensions
    }

    #[test]
    fn test_config_snapshot_creation() {
        let config = Config::default();
        let repodocs = RepoDocs::new_for_test(config, OutputMode::Human, 0, true);

        let snapshot = repodocs.create_config_snapshot();
        assert!(!snapshot.extensions.is_empty());
        assert!(snapshot.preserve_structure);
    }

    #[test]
    fn test_sample_config_generation() {
        let temp_dir = TempDir::new().unwrap();
        let config_path = temp_dir.path().join("sample.toml");

        let result = RepoDocs::generate_sample_config(&config_path);
        assert!(result.is_ok());
        assert!(config_path.exists());

        let content = std::fs::read_to_string(&config_path).unwrap();
        assert!(content.contains("[filters]"));
        assert!(content.contains("[output]"));
        assert!(content.contains("[git]"));
    }

    #[test]
    fn test_url_validation() {
        assert!(validate_repository_url("https://github.com/microsoft/vscode").is_ok());
        assert!(validate_repository_url("https://gitlab.com/owner/repo").is_err());
        assert!(validate_repository_url("not-a-url").is_err());
    }

    #[test]
    fn test_version_info() {
        let version = version_info();
        assert!(!version.is_empty());

        let build_info = build_info();
        assert!(!build_info.version.is_empty());
        assert!(!build_info.target.is_empty());
    }

    #[test]
    fn test_build_info_display() {
        let build_info = build_info();
        let display_string = build_info.to_string();
        assert!(display_string.contains("RepoDocs"));
        assert!(display_string.contains(build_info.version));
    }

    #[test]
    fn test_shutdown_handling() {
        let config = Config::default();
        let repodocs = RepoDocs::new_for_test(config, OutputMode::Human, 0, true);

        assert!(repodocs.is_running());

        repodocs.request_shutdown();
        assert!(!repodocs.is_running());
    }
}