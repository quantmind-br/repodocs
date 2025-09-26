use repodocs::{
    Cli, RepoDocs, Config, RepoDocsError, UserFriendlyError,
    OutputMode, OutputFormatter
};
use clap::Parser;
use std::process;

#[tokio::main]
async fn main() {
    let exit_code = run().await;
    process::exit(exit_code);
}

async fn run() -> i32 {
    // Parse CLI arguments
    let cli = Cli::parse();

    // Handle special commands first
    if cli.generate_config {
        return handle_generate_config(&cli);
    }

    // Create RepoDocs instance
    let repodocs = match RepoDocs::from_cli(&cli) {
        Ok(repodocs) => repodocs,
        Err(e) => {
            print_startup_error(&e);
            return 1;
        }
    };

    // Handle dry run mode
    if cli.dry_run {
        return handle_dry_run(&cli, &repodocs);
    }

    // Execute main extraction workflow
    match repodocs.extract_documentation(&cli.repository_url).await {
        Ok(report) => {
            // Display final report based on output format
            repodocs.output_formatter().print_extraction_report(&report);

            // Return appropriate exit code
            if report.errors.is_empty() {
                0 // Success
            } else {
                2 // Success with warnings
            }
        }
        Err(e) => {
            repodocs.handle_error(&e);

            // Map error types to appropriate exit codes
            match e {
                RepoDocsError::Cancelled => 130, // Interrupted (SIGINT)
                RepoDocsError::InvalidUrl { .. } => 2,
                RepoDocsError::RepositoryNotFound { .. } => 3,
                RepoDocsError::AuthenticationFailed { .. } => 4,
                RepoDocsError::NetworkError { .. } => 5,
                RepoDocsError::NoDocumentationFound { .. } => 6,
                RepoDocsError::Permission { .. } => 7,
                RepoDocsError::OutputDirectoryExists { .. } => 8,
                RepoDocsError::Timeout { .. } => 9,
                _ => 1, // General error
            }
        }
    }
}

fn handle_generate_config(cli: &Cli) -> i32 {
    let config_path = cli.config.as_ref()
        .map(|p| p.to_string_lossy().to_string())
        .unwrap_or_else(|| "repodocs.toml".to_string());

    match RepoDocs::generate_sample_config(&config_path) {
        Ok(()) => {
            println!("Generated sample configuration file: {}", config_path);
            println!("\nTo use this configuration:");
            println!("  repodocs <repository-url> --config {}", config_path);
            println!("\nEdit the file to customize settings for your needs.");
            0
        }
        Err(e) => {
            eprintln!("Failed to generate configuration file: {}", e.user_message());
            if let Some(suggestion) = e.suggestion() {
                eprintln!("Suggestion: {}", suggestion);
            }
            1
        }
    }
}

fn handle_dry_run(cli: &Cli, repodocs: &RepoDocs) -> i32 {
    let formatter = repodocs.output_formatter();

    formatter.info("DRY RUN MODE - No files will be extracted");
    formatter.print_separator();

    // Validate repository URL
    match repodocs::validate_repository_url(&cli.repository_url) {
        Ok(_) => formatter.success(&format!("✓ Repository URL is valid: {}", cli.repository_url)),
        Err(e) => {
            formatter.error(&format!("✗ Invalid repository URL: {}", e.user_message()));
            return 1;
        }
    }

    // Display configuration that would be used
    formatter.info("Configuration that would be used:");
    let config = repodocs.config();

    println!("  Extensions: {}", config.filters.extensions.join(", "));
    println!("  Max file size: {} bytes", config.filters.max_file_size);
    println!("  Exclude directories: {}", config.filters.exclude_dirs.join(", "));
    println!("  Preserve structure: {}", config.output.preserve_structure);
    println!("  Base directory: {}", config.output.base_directory.display());

    if let Some(ref branch) = config.git.branch {
        println!("  Git branch: {}", branch);
    }
    println!("  Git timeout: {} seconds", config.git.timeout);

    formatter.print_separator();

    // Extract repository information
    let (owner, repo_name) = match cli.extract_repo_info() {
        Ok(info) => info,
        Err(e) => {
            formatter.error(&format!("Failed to parse repository info: {}", e.user_message()));
            return 1;
        }
    };

    let output_dir = match cli.get_output_directory_name() {
        Ok(name) => name,
        Err(e) => {
            formatter.error(&format!("Failed to determine output directory: {}", e.user_message()));
            return 1;
        }
    };

    formatter.info("Extraction plan:");
    println!("  Repository: {}/{}", owner, repo_name);
    println!("  Output directory: {}", output_dir);

    if cli.force {
        formatter.warning("Force mode enabled - would overwrite existing directory");
    }

    formatter.print_separator();
    formatter.success("Dry run completed successfully");
    formatter.info("Run without --dry-run to perform actual extraction");

    0
}

fn print_startup_error(error: &RepoDocsError) {
    // Create a basic formatter for startup errors
    let formatter = OutputFormatter::new(OutputMode::Human, 0, false);
    formatter.print_user_friendly_error(error);
}

fn setup_logging() {
    // Basic logging setup - in a real implementation, this could be more sophisticated
    if std::env::var("RUST_LOG").is_err() {
        std::env::set_var("RUST_LOG", "repodocs=info");
    }

    // Initialize logging if needed
    // env_logger::init();
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;
    use std::fs;

    #[test]
    fn test_generate_config_command() {
        let temp_dir = TempDir::new().unwrap();
        let config_path = temp_dir.path().join("test.toml");

        let cli = Cli {
            repository_url: "https://github.com/test/repo".to_string(),
            output: None,
            formats: None,
            exclude: None,
            max_size: None,
            config: Some(config_path.clone()),
            output_format: repodocs::cli::OutputFormat::Human,
            preserve_structure: None,
            timeout: None,
            branch: None,
            verbose: 0,
            quiet: false,
            force: false,
            dry_run: false,
            generate_config: true,
        };

        let exit_code = handle_generate_config(&cli);
        assert_eq!(exit_code, 0);
        assert!(config_path.exists());

        let content = fs::read_to_string(&config_path).unwrap();
        assert!(content.contains("[filters]"));
    }

    #[test]
    fn test_dry_run_mode() {
        let config = Config::default();
        let repodocs = RepoDocs::new(config, OutputMode::Plain, 0, true).unwrap();

        let cli = Cli {
            repository_url: "https://github.com/microsoft/vscode".to_string(),
            output: None,
            formats: None,
            exclude: None,
            max_size: None,
            config: None,
            output_format: repodocs::cli::OutputFormat::Plain,
            preserve_structure: None,
            timeout: None,
            branch: None,
            verbose: 0,
            quiet: true,
            force: false,
            dry_run: true,
            generate_config: false,
        };

        let exit_code = handle_dry_run(&cli, &repodocs);
        assert_eq!(exit_code, 0);
    }

    #[test]
    fn test_invalid_url_handling() {
        let config = Config::default();
        let repodocs = RepoDocs::new(config, OutputMode::Plain, 0, true).unwrap();

        let cli = Cli {
            repository_url: "invalid-url".to_string(),
            output: None,
            formats: None,
            exclude: None,
            max_size: None,
            config: None,
            output_format: repodocs::cli::OutputFormat::Plain,
            preserve_structure: None,
            timeout: None,
            branch: None,
            verbose: 0,
            quiet: true,
            force: false,
            dry_run: true,
            generate_config: false,
        };

        let exit_code = handle_dry_run(&cli, &repodocs);
        assert_eq!(exit_code, 1);
    }
}