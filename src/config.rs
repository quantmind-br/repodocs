use crate::error::{RepoDocsError, Result};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use std::time::Duration;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Config {
    pub filters: FilterConfig,
    pub output: OutputConfig,
    pub git: GitConfig,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct FilterConfig {
    pub extensions: Vec<String>,
    pub max_file_size: u64,
    pub exclude_dirs: Vec<String>,
    pub exclude_patterns: Vec<String>,
    pub max_depth: usize,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct OutputConfig {
    pub preserve_structure: bool,
    pub create_index: bool,
    pub generate_report: bool,
    pub base_directory: PathBuf,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct GitConfig {
    pub clone_depth: Option<u32>,
    pub timeout: u64,
    pub branch: Option<String>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            filters: FilterConfig::default(),
            output: OutputConfig::default(),
            git: GitConfig::default(),
        }
    }
}

impl Default for FilterConfig {
    fn default() -> Self {
        Self {
            extensions: vec![
                "md".to_string(),
                "markdown".to_string(),
                "mdown".to_string(),
                "rst".to_string(),
                "rest".to_string(),
                "adoc".to_string(),
                "asciidoc".to_string(),
                "asc".to_string(),
                "txt".to_string(),
                "text".to_string(),
                "org".to_string(),
                "wiki".to_string(),
                "tex".to_string(),
                "latex".to_string(),
            ],
            max_file_size: 10 * 1024 * 1024, // 10MB
            exclude_dirs: vec![
                "node_modules".to_string(),
                ".git".to_string(),
                "target".to_string(),
                "build".to_string(),
                "dist".to_string(),
                "vendor".to_string(),
                ".vscode".to_string(),
                ".idea".to_string(),
            ],
            exclude_patterns: vec![
                r".*\.min\..*".to_string(),
                r".*\.lock".to_string(),
                r"package-lock\.json".to_string(),
                r"yarn\.lock".to_string(),
            ],
            max_depth: 10,
        }
    }
}

impl Default for OutputConfig {
    fn default() -> Self {
        Self {
            preserve_structure: true,
            create_index: true,
            generate_report: true,
            base_directory: std::env::current_dir().unwrap_or_else(|_| PathBuf::from(".")),
        }
    }
}

impl Default for GitConfig {
    fn default() -> Self {
        Self {
            clone_depth: None, // Full clone by default
            timeout: 300, // 5 minutes
            branch: None, // Default branch
        }
    }
}

impl Config {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn load_from_file<P: AsRef<Path>>(path: P) -> Result<Self> {
        let path = path.as_ref();

        if !path.exists() {
            return Err(RepoDocsError::Config {
                message: format!("Configuration file not found: {}", path.display()),
            });
        }

        let content = std::fs::read_to_string(path)
            .map_err(|e| RepoDocsError::Config {
                message: format!("Failed to read config file {}: {}", path.display(), e),
            })?;

        let config: Config = toml::from_str(&content)
            .map_err(|e| RepoDocsError::Config {
                message: format!("Failed to parse config file {}: {}", path.display(), e),
            })?;

        Ok(config)
    }

    pub fn load_with_defaults<P: AsRef<Path>>(config_path: Option<P>) -> Result<Self> {
        match config_path {
            Some(path) => Self::load_from_file(path),
            None => {
                // Try to load from default locations
                let default_paths = [
                    "repodocs.toml",
                    "repodocs.config.toml",
                    ".repodocs.toml",
                ];

                for default_path in &default_paths {
                    if Path::new(default_path).exists() {
                        return Self::load_from_file(default_path);
                    }
                }

                // If no config file found, use defaults
                Ok(Self::default())
            }
        }
    }

    pub fn merge_with_cli_args(&mut self, cli_args: &CliOverrides) {
        if let Some(ref formats) = cli_args.formats {
            self.filters.extensions = formats
                .split(',')
                .map(|s| s.trim().to_lowercase())
                .filter(|s| !s.is_empty())
                .collect();
        }

        if let Some(ref exclude) = cli_args.exclude {
            self.filters.exclude_dirs.extend(exclude.clone());
        }

        if let Some(max_size) = cli_args.max_file_size {
            self.filters.max_file_size = max_size;
        }

        if let Some(ref output_dir) = cli_args.output_dir {
            self.output.base_directory = output_dir.clone();
        }

        if let Some(preserve_structure) = cli_args.preserve_structure {
            self.output.preserve_structure = preserve_structure;
        }

        if let Some(timeout) = cli_args.timeout {
            self.git.timeout = timeout;
        }

        if let Some(ref branch) = cli_args.branch {
            self.git.branch = Some(branch.clone());
        }
    }

    pub fn save_to_file<P: AsRef<Path>>(&self, path: P) -> Result<()> {
        let path = path.as_ref();
        let content = toml::to_string_pretty(self)
            .map_err(|e| RepoDocsError::Config {
                message: format!("Failed to serialize config: {}", e),
            })?;

        std::fs::write(path, content)
            .map_err(|e| RepoDocsError::Config {
                message: format!("Failed to write config file {}: {}", path.display(), e),
            })?;

        Ok(())
    }

    pub fn validate(&self) -> Result<()> {
        // Validate file extensions
        if self.filters.extensions.is_empty() {
            return Err(RepoDocsError::Config {
                message: "At least one file extension must be specified".to_string(),
            });
        }

        // Validate max file size
        if self.filters.max_file_size == 0 {
            return Err(RepoDocsError::Config {
                message: "Maximum file size must be greater than 0".to_string(),
            });
        }

        // Validate timeout
        if self.git.timeout == 0 {
            return Err(RepoDocsError::Config {
                message: "Git timeout must be greater than 0".to_string(),
            });
        }

        // Validate max depth
        if self.filters.max_depth == 0 {
            return Err(RepoDocsError::Config {
                message: "Maximum directory depth must be greater than 0".to_string(),
            });
        }

        // Validate output directory
        if let Some(parent) = self.output.base_directory.parent() {
            if !parent.exists() {
                return Err(RepoDocsError::Config {
                    message: format!(
                        "Parent directory does not exist: {}",
                        parent.display()
                    ),
                });
            }
        }

        Ok(())
    }

    pub fn git_timeout_duration(&self) -> Duration {
        Duration::from_secs(self.git.timeout)
    }

    pub fn create_sample_config() -> String {
        let sample_config = Self::default();
        toml::to_string_pretty(&sample_config).unwrap_or_else(|_| String::new())
    }
}

#[derive(Debug, Default)]
pub struct CliOverrides {
    pub formats: Option<String>,
    pub exclude: Option<Vec<String>>,
    pub max_file_size: Option<u64>,
    pub output_dir: Option<PathBuf>,
    pub preserve_structure: Option<bool>,
    pub timeout: Option<u64>,
    pub branch: Option<String>,
}

impl CliOverrides {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_formats(mut self, formats: Option<String>) -> Self {
        self.formats = formats;
        self
    }

    pub fn with_exclude(mut self, exclude: Option<Vec<String>>) -> Self {
        self.exclude = exclude;
        self
    }

    pub fn with_max_file_size(mut self, max_size: Option<u64>) -> Self {
        self.max_file_size = max_size;
        self
    }

    pub fn with_output_dir(mut self, output_dir: Option<PathBuf>) -> Self {
        self.output_dir = output_dir;
        self
    }

    pub fn with_preserve_structure(mut self, preserve: Option<bool>) -> Self {
        self.preserve_structure = preserve;
        self
    }

    pub fn with_timeout(mut self, timeout: Option<u64>) -> Self {
        self.timeout = timeout;
        self
    }

    pub fn with_branch(mut self, branch: Option<String>) -> Self {
        self.branch = branch;
        self
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::NamedTempFile;
    use std::io::Write;

    #[test]
    fn test_default_config() {
        let config = Config::default();
        assert!(!config.filters.extensions.is_empty());
        assert!(config.filters.extensions.contains(&"md".to_string()));
        assert_eq!(config.git.timeout, 300);
    }

    #[test]
    fn test_config_validation() {
        let mut config = Config::default();
        assert!(config.validate().is_ok());

        config.filters.extensions.clear();
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_config_file_operations() {
        let config = Config::default();
        let mut temp_file = NamedTempFile::new().unwrap();

        // Test saving
        config.save_to_file(temp_file.path()).unwrap();

        // Test loading
        let loaded_config = Config::load_from_file(temp_file.path()).unwrap();
        assert_eq!(config.git.timeout, loaded_config.git.timeout);
    }

    #[test]
    fn test_cli_overrides() {
        let mut config = Config::default();
        let original_timeout = config.git.timeout;

        let overrides = CliOverrides::new()
            .with_timeout(Some(600))
            .with_formats(Some("md,txt".to_string()));

        config.merge_with_cli_args(&overrides);

        assert_eq!(config.git.timeout, 600);
        assert_ne!(config.git.timeout, original_timeout);
        assert_eq!(config.filters.extensions, vec!["md", "txt"]);
    }

    #[test]
    fn test_sample_config_generation() {
        let sample = Config::create_sample_config();
        assert!(!sample.is_empty());
        assert!(sample.contains("[filters]"));
        assert!(sample.contains("[output]"));
        assert!(sample.contains("[git]"));
    }
}