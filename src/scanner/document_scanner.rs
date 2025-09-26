use crate::config::FilterConfig;
use crate::error::{RepoDocsError, Result};
use crate::scanner::file_filter::FileFilter;
use std::path::{Path, PathBuf};
use std::time::SystemTime;
use walkdir::{DirEntry, WalkDir};

#[derive(Debug, Clone)]
pub struct DocumentFile {
    pub source_path: PathBuf,
    pub relative_path: PathBuf,
    pub filename: String,
    pub extension: String,
    pub size: u64,
    pub modified: SystemTime,
}

impl DocumentFile {
    pub fn new(
        source_path: PathBuf,
        relative_path: PathBuf,
        size: u64,
        modified: SystemTime,
    ) -> Self {
        let filename = source_path
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("")
            .to_string();

        let extension = source_path
            .extension()
            .and_then(|e| e.to_str())
            .unwrap_or("")
            .to_lowercase();

        Self {
            source_path,
            relative_path,
            filename,
            extension,
            size,
            modified,
        }
    }

    pub fn is_extensionless_doc(&self) -> bool {
        if !self.extension.is_empty() {
            return false;
        }

        let filename = self.filename.to_lowercase();
        matches!(
            filename.as_str(),
            "readme"
                | "license"
                | "changelog"
                | "contributing"
                | "authors"
                | "notice"
                | "install"
                | "usage"
                | "todo"
                | "copying"
                | "news"
                | "history"
                | "credits"
                | "maintainers"
                | "thanks"
                | "acknowledgments"
        )
    }

    pub fn display_path(&self) -> String {
        self.relative_path.display().to_string()
    }

    pub fn format_size(&self) -> String {
        format_bytes(self.size)
    }
}

pub struct DocumentScanner {
    filter: FileFilter,
    max_depth: usize,
    repo_root: Option<PathBuf>,
}

impl DocumentScanner {
    pub fn new(config: &FilterConfig) -> Self {
        Self {
            filter: FileFilter::new(config),
            max_depth: config.max_depth,
            repo_root: None,
        }
    }

    pub fn with_repo_root<P: Into<PathBuf>>(mut self, root: P) -> Self {
        self.repo_root = Some(root.into());
        self
    }

    pub fn scan_directory<P: AsRef<Path>>(&self, root: P) -> Result<Vec<DocumentFile>> {
        let root_path = root.as_ref();

        if !root_path.exists() {
            return Err(RepoDocsError::InvalidPath {
                path: root_path.display().to_string(),
            });
        }

        if !root_path.is_dir() {
            return Err(RepoDocsError::InvalidPath {
                path: format!("{} is not a directory", root_path.display()),
            });
        }

        let mut documents = Vec::new();
        let mut scan_errors = Vec::new();

        let walker = WalkDir::new(root_path)
            .max_depth(self.max_depth)
            .follow_links(false) // Security: don't follow symlinks
            .into_iter()
            .filter_entry(|e| self.should_traverse(e));

        for entry in walker {
            let entry = match entry {
                Ok(entry) => entry,
                Err(err) => {
                    // Log permission errors but continue scanning
                    if err
                        .io_error()
                        .is_some_and(|e| e.kind() == std::io::ErrorKind::PermissionDenied)
                    {
                        scan_errors.push(format!("Permission denied: {}", err));
                    } else {
                        scan_errors.push(format!("Scan error: {}", err));
                    }
                    continue;
                }
            };

            if entry.file_type().is_file() {
                match self.process_file(&entry, root_path) {
                    Ok(Some(doc_file)) => documents.push(doc_file),
                    Ok(None) => {} // File filtered out
                    Err(err) => {
                        scan_errors.push(format!(
                            "Error processing {}: {}",
                            entry.path().display(),
                            err
                        ));
                    }
                }
            }
        }

        // Log errors but don't fail the entire scan
        if !scan_errors.is_empty() && documents.is_empty() {
            return Err(RepoDocsError::Permission {
                path: format!("Multiple scan errors: {}", scan_errors.join(", ")),
            });
        }

        if documents.is_empty() {
            return Err(RepoDocsError::NoDocumentationFound {
                searched_extensions: self.filter.get_extensions().clone(),
            });
        }

        // Sort by relative path for consistent output
        documents.sort_by(|a, b| a.relative_path.cmp(&b.relative_path));

        Ok(documents)
    }

    fn should_traverse(&self, entry: &DirEntry) -> bool {
        let path = entry.path();

        // Security: Check depth limit
        if entry.depth() > self.max_depth {
            return false;
        }

        // Always allow traversing files
        if entry.file_type().is_file() {
            return true;
        }

        // Always allow traversing the root directory (depth 0)
        if entry.depth() == 0 {
            return true;
        }

        // For other directories, check against exclude patterns
        if entry.file_type().is_dir() {
            return self.filter.should_traverse_directory(path);
        }

        true
    }

    fn process_file(&self, entry: &DirEntry, root_path: &Path) -> Result<Option<DocumentFile>> {
        let path = entry.path();

        // Check if it's a documentation file
        if !self.filter.is_documentation_file(path) {
            return Ok(None);
        }

        // Get file metadata
        let metadata = entry.metadata().map_err(|e| RepoDocsError::Io(e.into()))?;

        // Check file size limits
        if !self.filter.is_size_allowed(metadata.len()) {
            return Ok(None);
        }

        // Calculate relative path
        let relative_path = self.calculate_relative_path(path, root_path)?;

        // Get modification time
        let modified = metadata.modified().unwrap_or(SystemTime::UNIX_EPOCH);

        let doc_file =
            DocumentFile::new(path.to_path_buf(), relative_path, metadata.len(), modified);

        Ok(Some(doc_file))
    }

    fn calculate_relative_path(&self, file_path: &Path, root_path: &Path) -> Result<PathBuf> {
        let relative =
            file_path
                .strip_prefix(root_path)
                .map_err(|_| RepoDocsError::InvalidPath {
                    path: format!(
                        "Cannot calculate relative path for {} from root {}",
                        file_path.display(),
                        root_path.display()
                    ),
                })?;

        // Security: Ensure the relative path doesn't contain parent directory references
        if relative
            .components()
            .any(|c| matches!(c, std::path::Component::ParentDir))
        {
            return Err(RepoDocsError::InvalidPath {
                path: format!(
                    "Path contains parent directory references: {}",
                    relative.display()
                ),
            });
        }

        Ok(relative.to_path_buf())
    }

    pub fn get_statistics(&self, documents: &[DocumentFile]) -> ScanStatistics {
        let total_files = documents.len();
        let total_size = documents.iter().map(|d| d.size).sum();

        // Group by extension
        let mut files_by_extension = std::collections::HashMap::new();
        for doc in documents {
            let ext = if doc.extension.is_empty() {
                "no_extension".to_string()
            } else {
                doc.extension.clone()
            };
            *files_by_extension.entry(ext).or_insert(0) += 1;
        }

        // Find largest file
        let (largest_file_size, largest_file_path) = documents
            .iter()
            .max_by_key(|d| d.size)
            .map(|d| (d.size, d.relative_path.clone()))
            .unwrap_or((0, PathBuf::new()));

        ScanStatistics {
            total_files,
            total_size,
            files_by_extension,
            largest_file_size,
            largest_file_path,
        }
    }
}

#[derive(Debug, Default)]
pub struct ScanStatistics {
    pub total_files: usize,
    pub total_size: u64,
    pub files_by_extension: std::collections::HashMap<String, usize>,
    pub largest_file_size: u64,
    pub largest_file_path: PathBuf,
}

impl ScanStatistics {
    pub fn display_summary(&self) -> String {
        let mut summary = format!(
            "Scan Results:\n  Total files: {}\n  Total size: {}\n",
            self.total_files,
            format_bytes(self.total_size)
        );

        if !self.files_by_extension.is_empty() {
            summary.push_str("  Files by type:\n");
            let mut extensions: Vec<_> = self.files_by_extension.iter().collect();
            extensions.sort_by(|a, b| b.1.cmp(a.1)); // Sort by count descending

            for (ext, count) in extensions {
                summary.push_str(&format!("    {}: {} files\n", ext, count));
            }
        }

        if self.largest_file_size > 0 {
            summary.push_str(&format!(
                "  Largest file: {} ({})\n",
                self.largest_file_path.display(),
                format_bytes(self.largest_file_size)
            ));
        }

        summary
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
    use std::fs;
    use tempfile::TempDir;

    fn create_test_config() -> FilterConfig {
        FilterConfig {
            extensions: vec!["md".to_string(), "txt".to_string()],
            max_file_size: 1024 * 1024, // 1MB
            exclude_dirs: vec![".git".to_string(), "node_modules".to_string()],
            exclude_patterns: vec![],
            max_depth: 5,
        }
    }

    #[test]
    fn test_document_file_creation() {
        let path = PathBuf::from("test.md");
        let relative_path = PathBuf::from("docs/test.md");
        let doc = DocumentFile::new(path, relative_path, 100, SystemTime::UNIX_EPOCH);

        assert_eq!(doc.filename, "test.md");
        assert_eq!(doc.extension, "md");
        assert_eq!(doc.size, 100);
    }

    #[test]
    fn test_extensionless_doc_detection() {
        let path = PathBuf::from("README");
        let relative_path = PathBuf::from("README");
        let doc = DocumentFile::new(path, relative_path, 100, SystemTime::UNIX_EPOCH);

        assert!(doc.is_extensionless_doc());
        assert_eq!(doc.extension, "");
    }

    #[test]
    fn test_scanner_with_test_files() {
        let temp_dir = TempDir::new().unwrap();
        let root = temp_dir.path();

        // Create a subdirectory that won't be filtered out
        let test_dir = root.join("test_docs");
        fs::create_dir(&test_dir).unwrap();

        // Create test files in the subdirectory
        fs::write(test_dir.join("README.md"), "# Test").unwrap();
        fs::write(test_dir.join("test.txt"), "test content").unwrap();

        // Verify files were actually created
        assert!(test_dir.join("README.md").exists());
        assert!(test_dir.join("test.txt").exists());

        // Use default config which includes md and txt
        let config = FilterConfig::default();
        let scanner = DocumentScanner::new(&config);

        // Test file filter directly
        assert!(scanner
            .filter
            .is_documentation_file(&test_dir.join("README.md")));
        assert!(scanner
            .filter
            .is_documentation_file(&test_dir.join("test.txt")));

        // Test directory traversal logic - the subdirectory should be traversed
        assert!(
            scanner.filter.should_traverse_directory(&test_dir),
            "Should traverse test directory"
        );

        // The scanner should find these files
        let documents = scanner.scan_directory(&test_dir).unwrap();

        // Both files should be found
        assert!(
            !documents.is_empty(),
            "Scanner should find at least one file"
        );
        assert!(
            documents.iter().any(|d| d.filename == "README.md"),
            "Should find README.md"
        );
        assert!(
            documents.iter().any(|d| d.filename == "test.txt"),
            "Should find test.txt"
        );
    }

    #[test]
    fn test_scan_statistics() {
        let documents = vec![
            DocumentFile::new(
                PathBuf::from("test.md"),
                PathBuf::from("test.md"),
                100,
                SystemTime::UNIX_EPOCH,
            ),
            DocumentFile::new(
                PathBuf::from("README"),
                PathBuf::from("README"),
                200,
                SystemTime::UNIX_EPOCH,
            ),
        ];

        let config = create_test_config();
        let scanner = DocumentScanner::new(&config);
        let stats = scanner.get_statistics(&documents);

        assert_eq!(stats.total_files, 2);
        assert_eq!(stats.total_size, 300);
        assert_eq!(stats.largest_file_size, 200);
    }

    #[test]
    fn test_format_bytes() {
        assert_eq!(format_bytes(500), "500 B");
        assert_eq!(format_bytes(1024), "1.0 KB");
        assert_eq!(format_bytes(1048576), "1.0 MB");
        assert_eq!(format_bytes(2 * 1024 * 1024), "2.0 MB");
    }
}
