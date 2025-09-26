use crate::error::{RepoDocsError, Result};
use crate::scanner::DocumentFile;
use std::fs;
use std::io::{BufReader, BufWriter, Read, Write};
use std::path::{Path, Component};
use std::time::{Duration, Instant};

#[derive(Debug, Clone)]
pub struct ExtractionProgress {
    pub files_processed: usize,
    pub total_files: usize,
    pub bytes_processed: u64,
    pub total_bytes: u64,
    pub current_file: Option<String>,
    pub start_time: Instant,
    pub errors: Vec<String>,
}

impl ExtractionProgress {
    pub fn new(total_files: usize, total_bytes: u64) -> Self {
        Self {
            files_processed: 0,
            total_files,
            bytes_processed: 0,
            total_bytes,
            current_file: None,
            start_time: Instant::now(),
            errors: Vec::new(),
        }
    }

    pub fn update_file(&mut self, filename: String, bytes: u64) {
        self.files_processed += 1;
        self.bytes_processed += bytes;
        self.current_file = Some(filename);
    }

    pub fn add_error<S: Into<String>>(&mut self, error: S) {
        self.errors.push(error.into());
    }

    pub fn percentage(&self) -> f64 {
        if self.total_files == 0 {
            0.0
        } else {
            (self.files_processed as f64 / self.total_files as f64) * 100.0
        }
    }

    pub fn elapsed(&self) -> Duration {
        self.start_time.elapsed()
    }

    pub fn estimated_remaining(&self) -> Duration {
        if self.files_processed == 0 {
            return Duration::from_secs(0);
        }

        let elapsed = self.elapsed();
        let rate = self.files_processed as f64 / elapsed.as_secs_f64();
        let remaining_files = self.total_files - self.files_processed;

        if rate > 0.0 {
            Duration::from_secs_f64(remaining_files as f64 / rate)
        } else {
            Duration::from_secs(0)
        }
    }
}

pub struct FileOperations {
    preserve_structure: bool,
    force_overwrite: bool,
    buffer_size: usize,
}

impl FileOperations {
    pub fn new() -> Self {
        Self {
            preserve_structure: true,
            force_overwrite: false,
            buffer_size: 64 * 1024, // 64KB buffer
        }
    }

    pub fn with_preserve_structure(mut self, preserve: bool) -> Self {
        self.preserve_structure = preserve;
        self
    }

    pub fn with_force_overwrite(mut self, force: bool) -> Self {
        self.force_overwrite = force;
        self
    }

    pub fn with_buffer_size(mut self, size: usize) -> Self {
        self.buffer_size = size.max(4096); // Minimum 4KB buffer
        self
    }

    pub fn extract_files(
        &self,
        documents: &[DocumentFile],
        output_root: &Path,
        progress_callback: Option<&dyn Fn(&ExtractionProgress)>,
    ) -> Result<ExtractionProgress> {
        let total_bytes = documents.iter().map(|d| d.size).sum();
        let mut progress = ExtractionProgress::new(documents.len(), total_bytes);

        // Create output directory if it doesn't exist
        if !output_root.exists() {
            fs::create_dir_all(output_root)
                .map_err(|e| RepoDocsError::Io(e))?;
        }

        for document in documents {
            if let Some(callback) = progress_callback {
                callback(&progress);
            }

            match self.copy_document(document, output_root) {
                Ok(bytes_copied) => {
                    progress.update_file(
                        document.filename.clone(),
                        bytes_copied,
                    );
                },
                Err(e) => {
                    let error_msg = format!(
                        "Failed to copy {}: {}",
                        document.source_path.display(),
                        e
                    );
                    progress.add_error(error_msg);
                    // Continue with other files instead of failing completely
                },
            }
        }

        // Final progress update
        if let Some(callback) = progress_callback {
            callback(&progress);
        }

        Ok(progress)
    }

    fn copy_document(&self, document: &DocumentFile, output_root: &Path) -> Result<u64> {
        let _dest_path = if self.preserve_structure {
            output_root.join(&document.relative_path)
        } else {
            output_root.join(&document.filename)
        };

        self.copy_preserving_structure(
            &document.source_path,
            output_root,
            &document.relative_path,
        )
    }

    pub fn copy_preserving_structure(
        &self,
        source: &Path,
        dest_root: &Path,
        relative_path: &Path,
    ) -> Result<u64> {
        let dest_path = if self.preserve_structure {
            dest_root.join(relative_path)
        } else {
            dest_root.join(
                relative_path.file_name()
                    .ok_or_else(|| RepoDocsError::InvalidPath {
                        path: relative_path.display().to_string(),
                    })?
            )
        };

        // Create parent directories
        if let Some(parent) = dest_path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| RepoDocsError::Io(e))?;
        }

        // Secure copy operation
        self.secure_copy(source, &dest_path)
    }

    fn secure_copy(&self, source: &Path, dest: &Path) -> Result<u64> {
        // Validate source exists and is readable
        if !source.exists() {
            return Err(RepoDocsError::InvalidPath {
                path: format!("Source file does not exist: {}", source.display()),
            });
        }

        if !source.is_file() {
            return Err(RepoDocsError::InvalidPath {
                path: format!("Source is not a file: {}", source.display()),
            });
        }

        // Validate destination path
        self.validate_destination_path(dest)?;

        // Security: Prevent overwriting existing files unless force is enabled
        if dest.exists() && !self.force_overwrite {
            return Err(RepoDocsError::OutputDirectoryExists {
                path: dest.display().to_string(),
            });
        }

        // Perform the copy operation
        self.copy_file_with_buffer(source, dest)
    }

    fn copy_file_with_buffer(&self, source: &Path, dest: &Path) -> Result<u64> {
        let source_file = fs::File::open(source)
            .map_err(|e| RepoDocsError::Io(e))?;

        let dest_file = fs::File::create(dest)
            .map_err(|e| RepoDocsError::Io(e))?;

        let mut reader = BufReader::with_capacity(self.buffer_size, source_file);
        let mut writer = BufWriter::with_capacity(self.buffer_size, dest_file);

        let mut total_bytes = 0u64;
        let mut buffer = vec![0u8; 8192]; // 8KB chunks

        loop {
            let bytes_read = reader.read(&mut buffer)
                .map_err(|e| RepoDocsError::Io(e))?;

            if bytes_read == 0 {
                break; // End of file
            }

            writer.write_all(&buffer[..bytes_read])
                .map_err(|e| RepoDocsError::Io(e))?;

            total_bytes += bytes_read as u64;
        }

        writer.flush().map_err(|e| RepoDocsError::Io(e))?;

        // Set file modification time to match source
        if let Ok(source_metadata) = fs::metadata(source) {
            if let Ok(modified_time) = source_metadata.modified() {
                let _ = filetime::set_file_mtime(dest, filetime::FileTime::from_system_time(modified_time));
            }
        }

        Ok(total_bytes)
    }

    fn validate_destination_path(&self, path: &Path) -> Result<()> {
        let path_str = path.to_string_lossy();

        // Security: Prevent directory traversal
        if path.components().any(|c| matches!(c, Component::ParentDir)) {
            return Err(RepoDocsError::InvalidPath {
                path: format!("Directory traversal not allowed: {}", path_str),
            });
        }

        // Security: Prevent very long paths
        if path_str.len() > 4096 {
            return Err(RepoDocsError::InvalidPath {
                path: format!("Path too long: {} characters", path_str.len()),
            });
        }

        // Cross-platform path validation
        self.validate_cross_platform_path(path)?;

        Ok(())
    }

    fn validate_cross_platform_path(&self, path: &Path) -> Result<()> {
        if let Some(filename) = path.file_name().and_then(|s| s.to_str()) {
            // Windows reserved names
            #[cfg(windows)]
            {
                let reserved_names = [
                    "CON", "PRN", "AUX", "NUL",
                    "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
                    "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"
                ];

                let name_upper = filename.to_uppercase();
                let base_name = if let Some(dot_pos) = name_upper.find('.') {
                    &name_upper[..dot_pos]
                } else {
                    &name_upper
                };

                if reserved_names.contains(&base_name) {
                    return Err(RepoDocsError::InvalidPath {
                        path: format!("Reserved filename on Windows: {}", filename),
                    });
                }
            }

            // Check for invalid characters
            let invalid_chars = ['<', '>', ':', '"', '|', '?', '*'];
            if filename.chars().any(|c| {
                invalid_chars.contains(&c) || c.is_control() || c == '\0'
            }) {
                return Err(RepoDocsError::InvalidPath {
                    path: format!("Filename contains invalid characters: {}", filename),
                });
            }

            // Check for names ending with space or dot (problematic on Windows)
            if filename.ends_with(' ') || filename.ends_with('.') {
                return Err(RepoDocsError::InvalidPath {
                    path: format!("Filename cannot end with space or dot: {}", filename),
                });
            }
        }

        Ok(())
    }

    pub fn create_index_file(&self, documents: &[DocumentFile], output_dir: &Path) -> Result<()> {
        let index_path = output_dir.join("_index.md");
        let mut index_file = fs::File::create(&index_path)
            .map_err(|e| RepoDocsError::Io(e))?;

        writeln!(index_file, "# Documentation Index")?;
        writeln!(index_file)?;
        writeln!(index_file, "Generated by RepoDocs on {}",
                 chrono::Utc::now().format("%Y-%m-%d %H:%M:%S UTC"))?;
        writeln!(index_file)?;

        // Group files by directory
        let mut files_by_dir: std::collections::BTreeMap<String, Vec<&DocumentFile>> =
            std::collections::BTreeMap::new();

        for doc in documents {
            let dir = doc.relative_path.parent()
                .map(|p| p.to_string_lossy().to_string())
                .unwrap_or_else(|| ".".to_string());

            files_by_dir.entry(dir).or_default().push(doc);
        }

        for (dir, files) in files_by_dir {
            if dir != "." {
                writeln!(index_file, "## {}/", dir)?;
            } else {
                writeln!(index_file, "## Root Directory")?;
            }
            writeln!(index_file)?;

            for file in files {
                let link_path = if self.preserve_structure {
                    file.relative_path.to_string_lossy()
                } else {
                    file.filename.as_str().into()
                };

                writeln!(
                    index_file,
                    "- [{}]({}) ({} bytes)",
                    file.filename,
                    link_path.replace('\\', "/"), // Use forward slashes for markdown links
                    file.size
                )?;
            }
            writeln!(index_file)?;
        }

        writeln!(index_file, "---")?;
        writeln!(index_file, "Total files: {}", documents.len())?;
        writeln!(index_file, "Total size: {} bytes",
                 documents.iter().map(|d| d.size).sum::<u64>())?;

        Ok(())
    }
}

impl Default for FileOperations {
    fn default() -> Self {
        Self::new()
    }
}

// Cross-platform filename sanitization
pub fn sanitize_filename(name: &str) -> String {
    let mut sanitized = String::new();

    for ch in name.chars() {
        match ch {
            // Windows/Unix reserved characters
            '<' | '>' | ':' | '"' | '|' | '?' | '*' => sanitized.push('_'),
            // Path separators
            '/' | '\\' => sanitized.push('_'),
            // Control characters
            c if c.is_control() => sanitized.push('_'),
            // Valid character
            c => sanitized.push(c),
        }
    }

    // Trim dots and spaces (problematic on Windows)
    let sanitized = sanitized.trim_end_matches(&['.', ' '][..]).to_string();

    // Ensure it's not empty
    if sanitized.is_empty() {
        "unnamed_file".to_string()
    } else {
        sanitized
    }
}

// Check if path exceeds platform limits
pub fn check_path_length(path: &Path) -> Result<()> {
    let path_str = path.to_string_lossy();

    #[cfg(windows)]
    const MAX_PATH: usize = 260;

    #[cfg(unix)]
    const MAX_PATH: usize = 4096;

    if path_str.len() > MAX_PATH {
        Err(RepoDocsError::InvalidPath {
            path: format!("Path too long: {} characters (max: {})", path_str.len(), MAX_PATH),
        })
    } else {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;
    use std::fs;
    use std::path::PathBuf;
    use std::time::SystemTime;

    fn create_test_document(name: &str, content: &str, temp_dir: &Path) -> DocumentFile {
        let file_path = temp_dir.join(name);
        fs::write(&file_path, content).unwrap();
        let metadata = fs::metadata(&file_path).unwrap();

        DocumentFile::new(
            file_path,
            PathBuf::from(name),
            metadata.len(),
            metadata.modified().unwrap_or(SystemTime::UNIX_EPOCH),
        )
    }

    #[test]
    fn test_file_extraction() {
        let source_dir = TempDir::new().unwrap();
        let dest_dir = TempDir::new().unwrap();

        // Create test files
        let doc1 = create_test_document("README.md", "# Test", source_dir.path());
        let doc2 = create_test_document("guide.txt", "Guide content", source_dir.path());

        let documents = vec![doc1, doc2];
        let operations = FileOperations::new();

        let progress = operations.extract_files(
            &documents,
            dest_dir.path(),
            None,
        ).unwrap();

        assert_eq!(progress.files_processed, 2);
        assert_eq!(progress.errors.len(), 0);
        assert!(dest_dir.path().join("README.md").exists());
        assert!(dest_dir.path().join("guide.txt").exists());
    }

    #[test]
    fn test_structure_preservation() {
        let source_dir = TempDir::new().unwrap();
        let dest_dir = TempDir::new().unwrap();

        // Create subdirectory and file
        let subdir = source_dir.path().join("docs");
        fs::create_dir(&subdir).unwrap();
        let file_path = subdir.join("nested.md");
        fs::write(&file_path, "nested content").unwrap();

        let metadata = fs::metadata(&file_path).unwrap();
        let document = DocumentFile::new(
            file_path,
            PathBuf::from("docs/nested.md"),
            metadata.len(),
            metadata.modified().unwrap_or(SystemTime::UNIX_EPOCH),
        );

        let operations = FileOperations::new().with_preserve_structure(true);
        let progress = operations.extract_files(
            &[document],
            dest_dir.path(),
            None,
        ).unwrap();

        assert_eq!(progress.files_processed, 1);
        assert!(dest_dir.path().join("docs").join("nested.md").exists());
    }

    #[test]
    fn test_filename_sanitization() {
        assert_eq!(sanitize_filename("normal_file.txt"), "normal_file.txt");
        assert_eq!(sanitize_filename("file<>with|bad*chars.txt"), "file__with_bad_chars.txt");
        assert_eq!(sanitize_filename("file/with\\slashes.txt"), "file_with_slashes.txt");
        assert_eq!(sanitize_filename("   "), "unnamed_file");
        assert_eq!(sanitize_filename("file..."), "file");
    }

    #[test]
    fn test_progress_tracking() {
        let mut progress = ExtractionProgress::new(10, 1000);

        assert_eq!(progress.percentage(), 0.0);

        progress.update_file("file1.txt".to_string(), 100);
        assert_eq!(progress.percentage(), 10.0);
        assert_eq!(progress.bytes_processed, 100);
        assert_eq!(progress.files_processed, 1);

        progress.add_error("Test error");
        assert_eq!(progress.errors.len(), 1);
    }

    #[test]
    fn test_index_file_creation() {
        let temp_dir = TempDir::new().unwrap();
        let doc = create_test_document("README.md", "# Test", temp_dir.path());

        let operations = FileOperations::new();
        operations.create_index_file(&[doc], temp_dir.path()).unwrap();

        let index_path = temp_dir.path().join("_index.md");
        assert!(index_path.exists());

        let content = fs::read_to_string(index_path).unwrap();
        assert!(content.contains("# Documentation Index"));
        assert!(content.contains("README.md"));
    }
}