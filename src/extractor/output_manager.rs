use crate::cloner::RepositoryInfo;
use crate::error::{RepoDocsError, Result};
use crate::extractor::ExtractionProgress;
use crate::scanner::DocumentFile;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::time::{Duration, SystemTime};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExtractionReport {
    pub repository_info: RepositoryInfo,
    pub extraction_summary: ExtractionSummary,
    pub files: Vec<FileInfo>,
    pub extraction_time: DateTime<Utc>,
    pub errors: Vec<String>,
    pub config_used: ConfigSnapshot,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExtractionSummary {
    pub total_files_processed: usize,
    pub total_bytes_processed: u64,
    pub extraction_duration: Duration,
    pub files_by_extension: std::collections::HashMap<String, usize>,
    pub largest_file: Option<FileInfo>,
    pub average_file_size: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileInfo {
    pub filename: String,
    pub relative_path: String,
    pub extension: String,
    pub size: u64,
    pub modified: SystemTime,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigSnapshot {
    pub extensions: Vec<String>,
    pub max_file_size: u64,
    pub exclude_dirs: Vec<String>,
    pub preserve_structure: bool,
}

impl From<&DocumentFile> for FileInfo {
    fn from(doc: &DocumentFile) -> Self {
        Self {
            filename: doc.filename.clone(),
            relative_path: doc.relative_path.to_string_lossy().to_string(),
            extension: doc.extension.clone(),
            size: doc.size,
            modified: doc.modified,
        }
    }
}

pub struct OutputManager {
    base_path: PathBuf,
    #[allow(dead_code)]
    repo_name: String,
    output_directory: PathBuf,
    force_overwrite: bool,
}

impl OutputManager {
    pub fn new(base_path: PathBuf, repo_name: String) -> Result<Self> {
        let output_directory = base_path.join(format!("docs_{}", sanitize_repo_name(&repo_name)));

        let manager = Self {
            base_path,
            repo_name,
            output_directory,
            force_overwrite: false,
        };

        manager.validate_paths()?;
        Ok(manager)
    }

    pub fn with_force_overwrite(mut self, force: bool) -> Self {
        self.force_overwrite = force;
        self
    }

    pub fn with_custom_output_name<S: Into<String>>(mut self, name: S) -> Self {
        let name = sanitize_repo_name(&name.into());
        self.output_directory = self.base_path.join(name);
        self
    }

    pub fn initialize(&self) -> Result<()> {
        if self.output_directory.exists() {
            if !self.force_overwrite {
                return Err(RepoDocsError::OutputDirectoryExists {
                    path: self.output_directory.display().to_string(),
                });
            } else {
                // Remove existing directory
                fs::remove_dir_all(&self.output_directory).map_err(RepoDocsError::Io)?;
            }
        }

        // Create output directory
        fs::create_dir_all(&self.output_directory).map_err(RepoDocsError::Io)?;

        // Create .repodocs metadata directory
        let metadata_dir = self.output_directory.join(".repodocs");
        fs::create_dir_all(&metadata_dir).map_err(RepoDocsError::Io)?;

        Ok(())
    }

    pub fn get_output_directory(&self) -> &Path {
        &self.output_directory
    }

    pub fn create_extraction_report(
        &self,
        repository_info: &RepositoryInfo,
        documents: &[DocumentFile],
        progress: &ExtractionProgress,
        config: &ConfigSnapshot,
    ) -> Result<ExtractionReport> {
        let extraction_summary = self.create_extraction_summary(documents, progress);
        let file_infos: Vec<FileInfo> = documents.iter().map(FileInfo::from).collect();

        let report = ExtractionReport {
            repository_info: repository_info.clone(),
            extraction_summary,
            files: file_infos,
            extraction_time: Utc::now(),
            errors: progress.errors.clone(),
            config_used: config.clone(),
        };

        // Save report in multiple formats
        self.save_report_json(&report)?;
        self.save_report_text(&report)?;
        self.create_summary_file(&report)?;

        Ok(report)
    }

    fn create_extraction_summary(
        &self,
        documents: &[DocumentFile],
        progress: &ExtractionProgress,
    ) -> ExtractionSummary {
        let mut files_by_extension: std::collections::HashMap<String, usize> =
            std::collections::HashMap::new();
        let mut largest_file: Option<&DocumentFile> = None;

        for doc in documents {
            let ext = if doc.extension.is_empty() {
                "no_extension".to_string()
            } else {
                doc.extension.clone()
            };

            *files_by_extension.entry(ext).or_insert(0) += 1;

            if largest_file.is_none_or(|f| doc.size > f.size) {
                largest_file = Some(doc);
            }
        }

        let total_bytes: u64 = documents.iter().map(|d| d.size).sum();
        let average_file_size = if documents.is_empty() {
            0
        } else {
            total_bytes / documents.len() as u64
        };

        ExtractionSummary {
            total_files_processed: progress.files_processed,
            total_bytes_processed: progress.bytes_processed,
            extraction_duration: progress.elapsed(),
            files_by_extension,
            largest_file: largest_file.map(FileInfo::from),
            average_file_size,
        }
    }

    fn save_report_json(&self, report: &ExtractionReport) -> Result<()> {
        let report_path = self
            .output_directory
            .join(".repodocs")
            .join("extraction_report.json");
        let json_content =
            serde_json::to_string_pretty(report).map_err(|e| RepoDocsError::Config {
                message: format!("Failed to serialize report to JSON: {}", e),
            })?;

        fs::write(&report_path, json_content).map_err(RepoDocsError::Io)?;

        Ok(())
    }

    fn save_report_text(&self, report: &ExtractionReport) -> Result<()> {
        let report_path = self
            .output_directory
            .join(".repodocs")
            .join("extraction_report.txt");
        let mut file = fs::File::create(&report_path).map_err(RepoDocsError::Io)?;

        writeln!(file, "RepoDocs Extraction Report")?;
        writeln!(file, "==========================")?;
        writeln!(file)?;

        // Repository information
        writeln!(
            file,
            "Repository: {}/{}",
            report.repository_info.owner, report.repository_info.name
        )?;
        writeln!(file, "URL: {}", report.repository_info.url)?;
        writeln!(file, "Branch: {}", report.repository_info.default_branch)?;
        writeln!(
            file,
            "Total commits: {}",
            report.repository_info.total_commits
        )?;
        writeln!(
            file,
            "Repository empty: {}",
            report.repository_info.is_empty
        )?;
        writeln!(file)?;

        // Extraction summary
        writeln!(file, "Extraction Summary:")?;
        writeln!(
            file,
            "  Extracted at: {}",
            report.extraction_time.format("%Y-%m-%d %H:%M:%S UTC")
        )?;
        writeln!(
            file,
            "  Duration: {:?}",
            report.extraction_summary.extraction_duration
        )?;
        writeln!(
            file,
            "  Files processed: {}",
            report.extraction_summary.total_files_processed
        )?;
        writeln!(
            file,
            "  Bytes processed: {} ({})",
            report.extraction_summary.total_bytes_processed,
            format_bytes(report.extraction_summary.total_bytes_processed)
        )?;
        writeln!(
            file,
            "  Average file size: {} ({})",
            report.extraction_summary.average_file_size,
            format_bytes(report.extraction_summary.average_file_size)
        )?;
        writeln!(file)?;

        // Files by extension
        if !report.extraction_summary.files_by_extension.is_empty() {
            writeln!(file, "Files by extension:")?;
            let mut extensions: Vec<_> = report
                .extraction_summary
                .files_by_extension
                .iter()
                .collect();
            extensions.sort_by(|a, b| b.1.cmp(a.1)); // Sort by count descending

            for (ext, count) in extensions {
                writeln!(file, "  {}: {} files", ext, count)?;
            }
            writeln!(file)?;
        }

        // Largest file
        if let Some(ref largest) = report.extraction_summary.largest_file {
            writeln!(file, "Largest file:")?;
            writeln!(file, "  Name: {}", largest.filename)?;
            writeln!(file, "  Path: {}", largest.relative_path)?;
            writeln!(
                file,
                "  Size: {} ({})",
                largest.size,
                format_bytes(largest.size)
            )?;
            writeln!(file)?;
        }

        // Configuration used
        writeln!(file, "Configuration used:")?;
        writeln!(
            file,
            "  Extensions: {}",
            report.config_used.extensions.join(", ")
        )?;
        writeln!(
            file,
            "  Max file size: {} ({})",
            report.config_used.max_file_size,
            format_bytes(report.config_used.max_file_size)
        )?;
        writeln!(
            file,
            "  Excluded directories: {}",
            report.config_used.exclude_dirs.join(", ")
        )?;
        writeln!(
            file,
            "  Preserve structure: {}",
            report.config_used.preserve_structure
        )?;
        writeln!(file)?;

        // Errors (if any)
        if !report.errors.is_empty() {
            writeln!(file, "Errors encountered:")?;
            for error in &report.errors {
                writeln!(file, "  - {}", error)?;
            }
            writeln!(file)?;
        }

        // File listing
        writeln!(file, "Extracted files:")?;
        for file_info in &report.files {
            writeln!(
                file,
                "  {} ({} bytes) - {}",
                file_info.relative_path,
                file_info.size,
                file_info.extension.as_str()
            )?;
        }

        Ok(())
    }

    fn create_summary_file(&self, report: &ExtractionReport) -> Result<()> {
        let summary_path = self.output_directory.join("EXTRACTION_SUMMARY.md");
        let mut file = fs::File::create(&summary_path).map_err(RepoDocsError::Io)?;

        writeln!(file, "# Documentation Extraction Summary")?;
        writeln!(file)?;
        writeln!(
            file,
            "**Repository:** [{}/{}]({})",
            report.repository_info.owner, report.repository_info.name, report.repository_info.url
        )?;
        writeln!(
            file,
            "**Extracted:** {}",
            report.extraction_time.format("%Y-%m-%d %H:%M UTC")
        )?;
        writeln!(
            file,
            "**Duration:** {:?}",
            report.extraction_summary.extraction_duration
        )?;
        writeln!(file)?;

        writeln!(file, "## Statistics")?;
        writeln!(file)?;
        writeln!(
            file,
            "- **Files processed:** {}",
            report.extraction_summary.total_files_processed
        )?;
        writeln!(
            file,
            "- **Total size:** {}",
            format_bytes(report.extraction_summary.total_bytes_processed)
        )?;
        writeln!(
            file,
            "- **Average file size:** {}",
            format_bytes(report.extraction_summary.average_file_size)
        )?;
        writeln!(file)?;

        if !report.extraction_summary.files_by_extension.is_empty() {
            writeln!(file, "## File Types")?;
            writeln!(file)?;
            let mut extensions: Vec<_> = report
                .extraction_summary
                .files_by_extension
                .iter()
                .collect();
            extensions.sort_by(|a, b| b.1.cmp(a.1));

            for (ext, count) in extensions {
                let display_ext = if ext == "no_extension" {
                    "no extension"
                } else {
                    ext
                };
                writeln!(file, "- **{}**: {} files", display_ext, count)?;
            }
            writeln!(file)?;
        }

        if !report.errors.is_empty() {
            writeln!(file, "## Issues Encountered")?;
            writeln!(file)?;
            for error in &report.errors {
                writeln!(file, "- {}", error)?;
            }
            writeln!(file)?;
        }

        writeln!(file, "---")?;
        writeln!(file, "*Generated by RepoDocs*")?;

        Ok(())
    }

    fn validate_paths(&self) -> Result<()> {
        // Check if base path is writable
        if !self.base_path.exists() {
            fs::create_dir_all(&self.base_path).map_err(|e| RepoDocsError::Permission {
                path: format!(
                    "Cannot create base directory {}: {}",
                    self.base_path.display(),
                    e
                ),
            })?;
        }

        // Test write permissions
        let test_file = self.base_path.join(".repodocs_write_test");
        match fs::File::create(&test_file) {
            Ok(_) => {
                let _ = fs::remove_file(&test_file); // Clean up test file
            }
            Err(e) => {
                return Err(RepoDocsError::Permission {
                    path: format!(
                        "No write permission for directory {}: {}",
                        self.base_path.display(),
                        e
                    ),
                });
            }
        }

        Ok(())
    }

    pub fn cleanup_on_error(&self) -> Result<()> {
        if self.output_directory.exists() {
            fs::remove_dir_all(&self.output_directory).map_err(RepoDocsError::Io)?;
        }
        Ok(())
    }

    pub fn get_metadata_dir(&self) -> PathBuf {
        self.output_directory.join(".repodocs")
    }
}

fn sanitize_repo_name(name: &str) -> String {
    let mut sanitized = String::new();

    for ch in name.chars() {
        match ch {
            // Replace invalid filesystem characters
            '<' | '>' | ':' | '"' | '|' | '?' | '*' | '/' | '\\' => sanitized.push('_'),
            // Keep valid characters
            c if c.is_alphanumeric() || c == '-' || c == '.' || c == '_' => sanitized.push(c),
            // Replace other characters with underscore
            _ => sanitized.push('_'),
        }
    }

    // Ensure it doesn't start or end with dots, spaces, or underscores
    let sanitized = sanitized.trim_matches(|c| c == '.' || c == ' ' || c == '_');

    // Ensure it's not empty and not too long
    if sanitized.is_empty() {
        "unnamed_repo".to_string()
    } else if sanitized.len() > 100 {
        sanitized[..100].to_string()
    } else {
        sanitized.to_string()
    }
}

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
    use std::time::SystemTime;
    use tempfile::TempDir;

    fn create_test_repo_info() -> RepositoryInfo {
        RepositoryInfo {
            name: "test-repo".to_string(),
            owner: "test-owner".to_string(),
            default_branch: "main".to_string(),
            is_empty: false,
            total_commits: 42,
            url: "https://github.com/test-owner/test-repo".to_string(),
        }
    }

    fn create_test_document(name: &str, size: u64) -> DocumentFile {
        use crate::scanner::DocumentFile;
        DocumentFile::new(
            PathBuf::from(name),
            PathBuf::from(name),
            size,
            SystemTime::UNIX_EPOCH,
        )
    }

    fn create_test_config() -> ConfigSnapshot {
        ConfigSnapshot {
            extensions: vec!["md".to_string(), "txt".to_string()],
            max_file_size: 1024 * 1024,
            exclude_dirs: vec![".git".to_string()],
            preserve_structure: true,
        }
    }

    #[test]
    fn test_output_manager_creation() {
        let temp_dir = TempDir::new().unwrap();
        let manager =
            OutputManager::new(temp_dir.path().to_path_buf(), "test-repo".to_string()).unwrap();

        assert_eq!(manager.repo_name, "test-repo");
        assert_eq!(
            manager.output_directory,
            temp_dir.path().join("docs_test-repo")
        );
    }

    #[test]
    fn test_output_directory_initialization() {
        let temp_dir = TempDir::new().unwrap();
        let manager =
            OutputManager::new(temp_dir.path().to_path_buf(), "test-repo".to_string()).unwrap();

        manager.initialize().unwrap();

        assert!(manager.get_output_directory().exists());
        assert!(manager.get_metadata_dir().exists());
    }

    #[test]
    fn test_extraction_report_creation() {
        let temp_dir = TempDir::new().unwrap();
        let manager =
            OutputManager::new(temp_dir.path().to_path_buf(), "test-repo".to_string()).unwrap();

        manager.initialize().unwrap();

        let repo_info = create_test_repo_info();
        let documents = vec![
            create_test_document("README.md", 100),
            create_test_document("guide.txt", 200),
        ];

        let mut progress = ExtractionProgress::new(2, 300);
        progress.files_processed = 2;
        progress.bytes_processed = 300;

        let config = create_test_config();

        let report = manager
            .create_extraction_report(&repo_info, &documents, &progress, &config)
            .unwrap();

        assert_eq!(report.files.len(), 2);
        assert_eq!(report.extraction_summary.total_files_processed, 2);
        assert_eq!(report.extraction_summary.total_bytes_processed, 300);

        // Check that report files were created
        assert!(manager
            .get_metadata_dir()
            .join("extraction_report.json")
            .exists());
        assert!(manager
            .get_metadata_dir()
            .join("extraction_report.txt")
            .exists());
        assert!(manager
            .get_output_directory()
            .join("EXTRACTION_SUMMARY.md")
            .exists());
    }

    #[test]
    fn test_repo_name_sanitization() {
        assert_eq!(sanitize_repo_name("normal-repo"), "normal-repo");
        assert_eq!(sanitize_repo_name("repo/with/slashes"), "repo_with_slashes");
        assert_eq!(sanitize_repo_name("repo:with:colons"), "repo_with_colons");
        assert_eq!(sanitize_repo_name(""), "unnamed_repo");
        assert_eq!(sanitize_repo_name("   "), "unnamed_repo");

        let long_name = "a".repeat(150);
        let sanitized = sanitize_repo_name(&long_name);
        assert_eq!(sanitized.len(), 100);
    }

    #[test]
    fn test_force_overwrite() {
        let temp_dir = TempDir::new().unwrap();
        let manager =
            OutputManager::new(temp_dir.path().to_path_buf(), "test-repo".to_string()).unwrap();

        // Create initial directory
        manager.initialize().unwrap();
        assert!(manager.get_output_directory().exists());

        // Create a file in the directory
        fs::write(manager.get_output_directory().join("test.txt"), "test").unwrap();

        // Try to initialize again without force - should fail
        assert!(manager.initialize().is_err());

        // Try with force overwrite - should succeed
        let manager_with_force = manager.with_force_overwrite(true);
        manager_with_force.initialize().unwrap();
        assert!(manager_with_force.get_output_directory().exists());
        assert!(!manager_with_force
            .get_output_directory()
            .join("test.txt")
            .exists());
    }

    #[test]
    fn test_format_bytes() {
        assert_eq!(format_bytes(500), "500 B");
        assert_eq!(format_bytes(1024), "1.0 KB");
        assert_eq!(format_bytes(1048576), "1.0 MB");
        assert_eq!(format_bytes(1073741824), "1.0 GB");
    }
}
