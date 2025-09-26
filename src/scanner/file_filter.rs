use crate::config::FilterConfig;
use regex::Regex;
use std::path::Path;

pub struct FileFilter {
    doc_extensions: Vec<String>,
    max_file_size: u64,
    exclude_dirs: Vec<String>,
    exclude_patterns: Vec<Regex>,
}

impl FileFilter {
    pub fn new(config: &FilterConfig) -> Self {
        let exclude_patterns = config
            .exclude_patterns
            .iter()
            .filter_map(|pattern| Regex::new(pattern).ok())
            .collect();

        Self {
            doc_extensions: config.extensions.clone(),
            max_file_size: config.max_file_size,
            exclude_dirs: config.exclude_dirs.clone(),
            exclude_patterns,
        }
    }

    pub fn is_documentation_file(&self, path: &Path) -> bool {
        // Check by extension first
        if let Some(extension) = path.extension().and_then(|s| s.to_str()) {
            let ext_lower = extension.to_lowercase();
            if self.doc_extensions.contains(&ext_lower) {
                return true;
            }
        }

        // Check for extensionless documentation files
        if let Some(filename) = path.file_name().and_then(|s| s.to_str()) {
            let filename_lower = filename.to_lowercase();
            return self.is_extensionless_doc(&filename_lower);
        }

        false
    }

    fn is_extensionless_doc(&self, filename: &str) -> bool {
        matches!(
            filename,
            "readme"
                | "license"
                | "licence"
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
                | "acknowledgements"
                | "code_of_conduct"
                | "security"
                | "support"
                | "codeofconduct"
        )
    }

    pub fn should_traverse_directory(&self, path: &Path) -> bool {
        if let Some(dir_name) = path.file_name().and_then(|s| s.to_str()) {
            let dir_name_lower = dir_name.to_lowercase();

            // Check against excluded directories
            if self
                .exclude_dirs
                .iter()
                .any(|exclude| exclude.to_lowercase() == dir_name_lower)
            {
                return false;
            }

            // Check against exclude patterns
            let path_str = path.to_string_lossy();
            for pattern in &self.exclude_patterns {
                if pattern.is_match(&path_str) {
                    return false;
                }
            }

            // Skip hidden directories (starting with .)
            if dir_name.starts_with('.') && dir_name != "." && dir_name != ".." {
                // Allow some common documentation directories
                if !matches!(
                    dir_name_lower.as_str(),
                    ".github" | ".vscode" | ".devcontainer"
                ) {
                    return false;
                }
            }

            // Skip common build/output directories
            if matches!(
                dir_name_lower.as_str(),
                "target"
                    | "build"
                    | "dist"
                    | "out"
                    | "output"
                    | "bin"
                    | "obj"
                    | "node_modules"
                    | "vendor"
                    | ".cache"
                    | "tmp"
                    | "temp"
                    | "__pycache__"
                    | ".pytest_cache"
                    | ".mypy_cache"
                    | "coverage"
                    | ".coverage"
                    | "htmlcov"
            ) {
                return false;
            }
        }

        true
    }

    pub fn is_size_allowed(&self, size: u64) -> bool {
        size <= self.max_file_size
    }

    pub fn get_extensions(&self) -> &Vec<String> {
        &self.doc_extensions
    }

    pub fn matches_any_pattern(&self, text: &str) -> bool {
        self.exclude_patterns
            .iter()
            .any(|pattern| pattern.is_match(text))
    }

    pub fn add_extension<S: Into<String>>(&mut self, extension: S) {
        let ext = extension.into().to_lowercase();
        if !self.doc_extensions.contains(&ext) {
            self.doc_extensions.push(ext);
        }
    }

    pub fn remove_extension(&mut self, extension: &str) {
        let ext_lower = extension.to_lowercase();
        self.doc_extensions.retain(|e| e != &ext_lower);
    }

    pub fn add_exclude_directory<S: Into<String>>(&mut self, directory: S) {
        let dir = directory.into();
        if !self.exclude_dirs.contains(&dir) {
            self.exclude_dirs.push(dir);
        }
    }

    pub fn set_max_file_size(&mut self, size: u64) {
        self.max_file_size = size;
    }

    pub fn get_max_file_size(&self) -> u64 {
        self.max_file_size
    }

    pub fn get_exclude_dirs(&self) -> &Vec<String> {
        &self.exclude_dirs
    }
}

impl Default for FileFilter {
    fn default() -> Self {
        let config = FilterConfig::default();
        Self::new(&config)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_config() -> FilterConfig {
        FilterConfig {
            extensions: vec![
                "md".to_string(),
                "rst".to_string(),
                "txt".to_string(),
                "adoc".to_string(),
            ],
            max_file_size: 1024 * 1024, // 1MB
            exclude_dirs: vec![
                ".git".to_string(),
                "node_modules".to_string(),
                "target".to_string(),
            ],
            exclude_patterns: vec![r".*\.min\..*".to_string(), r".*\.lock".to_string()],
            max_depth: 10,
        }
    }

    #[test]
    fn test_documentation_file_detection() {
        let config = create_test_config();
        let filter = FileFilter::new(&config);

        // Test extension-based detection
        assert!(filter.is_documentation_file(Path::new("README.md")));
        assert!(filter.is_documentation_file(Path::new("guide.rst")));
        assert!(filter.is_documentation_file(Path::new("notes.txt")));
        assert!(filter.is_documentation_file(Path::new("manual.adoc")));

        // Test case insensitivity
        assert!(filter.is_documentation_file(Path::new("README.MD")));
        assert!(filter.is_documentation_file(Path::new("guide.RST")));

        // Test extensionless files
        assert!(filter.is_documentation_file(Path::new("README")));
        assert!(filter.is_documentation_file(Path::new("LICENSE")));
        assert!(filter.is_documentation_file(Path::new("CHANGELOG")));
        assert!(filter.is_documentation_file(Path::new("CONTRIBUTING")));

        // Test non-documentation files
        assert!(!filter.is_documentation_file(Path::new("script.js")));
        assert!(!filter.is_documentation_file(Path::new("style.css")));
        assert!(!filter.is_documentation_file(Path::new("image.png")));
        assert!(!filter.is_documentation_file(Path::new("data.json")));
    }

    #[test]
    fn test_directory_traversal_rules() {
        let config = create_test_config();
        let filter = FileFilter::new(&config);

        // Should traverse normal directories
        assert!(filter.should_traverse_directory(Path::new("docs")));
        assert!(filter.should_traverse_directory(Path::new("documentation")));
        assert!(filter.should_traverse_directory(Path::new("examples")));

        // Should not traverse excluded directories
        assert!(!filter.should_traverse_directory(Path::new(".git")));
        assert!(!filter.should_traverse_directory(Path::new("node_modules")));
        assert!(!filter.should_traverse_directory(Path::new("target")));

        // Should not traverse build/output directories
        assert!(!filter.should_traverse_directory(Path::new("build")));
        assert!(!filter.should_traverse_directory(Path::new("dist")));
        assert!(!filter.should_traverse_directory(Path::new("__pycache__")));

        // Should not traverse most hidden directories
        assert!(!filter.should_traverse_directory(Path::new(".cache")));
        assert!(!filter.should_traverse_directory(Path::new(".pytest_cache")));

        // Should traverse some special hidden directories
        assert!(filter.should_traverse_directory(Path::new(".github")));
        assert!(filter.should_traverse_directory(Path::new(".vscode")));
    }

    #[test]
    fn test_size_limits() {
        let config = create_test_config();
        let filter = FileFilter::new(&config);

        assert!(filter.is_size_allowed(1024)); // 1KB - allowed
        assert!(filter.is_size_allowed(1024 * 1024)); // 1MB - allowed
        assert!(!filter.is_size_allowed(2 * 1024 * 1024)); // 2MB - not allowed
    }

    #[test]
    fn test_pattern_matching() {
        let config = create_test_config();
        let filter = FileFilter::new(&config);

        assert!(filter.matches_any_pattern("app.min.js"));
        assert!(filter.matches_any_pattern("package.lock"));
        assert!(!filter.matches_any_pattern("regular-file.js"));
    }

    #[test]
    fn test_filter_modification() {
        let config = create_test_config();
        let mut filter = FileFilter::new(&config);

        // Test adding extension
        assert!(!filter.is_documentation_file(Path::new("test.org")));
        filter.add_extension("org");
        assert!(filter.is_documentation_file(Path::new("test.org")));

        // Test removing extension
        assert!(filter.is_documentation_file(Path::new("test.md")));
        filter.remove_extension("md");
        assert!(!filter.is_documentation_file(Path::new("test.md")));

        // Test size modification
        filter.set_max_file_size(2048);
        assert_eq!(filter.get_max_file_size(), 2048);
        assert!(filter.is_size_allowed(2048));
        assert!(!filter.is_size_allowed(4096));
    }

    #[test]
    fn test_case_insensitive_extensions() {
        let config = create_test_config();
        let filter = FileFilter::new(&config);

        assert!(filter.is_documentation_file(Path::new("README.md")));
        assert!(filter.is_documentation_file(Path::new("README.MD")));
        assert!(filter.is_documentation_file(Path::new("README.Md")));
        assert!(filter.is_documentation_file(Path::new("readme.MD")));
    }

    #[test]
    fn test_extensionless_documentation_files() {
        let config = create_test_config();
        let filter = FileFilter::new(&config);

        let doc_files = [
            "README",
            "LICENSE",
            "LICENCE",
            "CHANGELOG",
            "CONTRIBUTING",
            "AUTHORS",
            "NOTICE",
            "INSTALL",
            "USAGE",
            "TODO",
            "COPYING",
            "NEWS",
            "HISTORY",
            "CREDITS",
            "MAINTAINERS",
            "THANKS",
            "ACKNOWLEDGMENTS",
            "ACKNOWLEDGEMENTS",
            "CODE_OF_CONDUCT",
            "SECURITY",
            "SUPPORT",
            "CODEOFCONDUCT",
        ];

        for file in &doc_files {
            assert!(
                filter.is_documentation_file(Path::new(file)),
                "Should recognize {} as documentation",
                file
            );
        }

        let non_doc_files = ["BINARY", "EXECUTABLE", "CONFIG", "MAKEFILE"];
        for file in &non_doc_files {
            assert!(
                !filter.is_documentation_file(Path::new(file)),
                "Should not recognize {} as documentation",
                file
            );
        }
    }
}
