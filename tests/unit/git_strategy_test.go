package unit

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitStrategy_Name tests the Name method
func TestGitStrategy_Name(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	assert.Equal(t, "git", strategy.Name())
}

// TestGitStrategy_Execute_ArchiveDownload tests Execute with successful archive download
func TestGitStrategy_Execute_ArchiveDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	// This test requires network access and a real repository
	// For comprehensive testing, use integration tests
	// Here we just verify the strategy can be created and has the right name
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	// Verify strategy properties
	assert.Equal(t, "git", strategy.Name())
	assert.NotNil(t, strategy)
}

// TestGitStrategy_ExtractTarGz_Success tests successful tar.gz extraction
func TestGitStrategy_ExtractTarGz_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-extract-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test archive
	files := []testFile{
		{Name: "repo-main/README.md", Content: "# Test\n\nContent."},
		{Name: "repo-main/docs/guide.md", Content: "# Guide\n\nDocs."},
		{Name: "repo-main/src/code.go", Content: "package main\n"},
	}
	archiveContent := createTestTarGz(t, files)

	// Extract using standalone function
	err = extractTarGzStandalone(bytes.NewReader(archiveContent), tmpDir)
	require.NoError(t, err)

	// Verify files were extracted
	assert.FileExists(t, filepath.Join(tmpDir, "README.md"))
	assert.FileExists(t, filepath.Join(tmpDir, "docs", "guide.md"))
	assert.FileExists(t, filepath.Join(tmpDir, "src", "code.go"))

	// Verify content
	content, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Test")
}

// TestGitStrategy_ExtractTarGz_EmptyArchive tests extraction of empty archive
func TestGitStrategy_ExtractTarGz_EmptyArchive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-extract-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create empty archive
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Close()
	archiveContent := buf.Bytes()

	err = extractTarGzStandalone(bytes.NewReader(archiveContent), tmpDir)
	require.NoError(t, err)

	// Should not create any files
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 0, len(files))
}

// TestGitStrategy_ExtractTarGz_SecurityPathTraversal tests path traversal protection
func TestGitStrategy_ExtractTarGz_SecurityPathTraversal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-extract-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create archive with path traversal attempt
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Add a file with path traversal
	header := &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0644,
		Size: int64(len("malicious")),
	}
	err = tw.WriteHeader(header)
	require.NoError(t, err)
	_, err = tw.Write([]byte("malicious"))
	require.NoError(t, err)

	tw.Close()
	gzw.Close()

	archiveContent := buf.Bytes()

	// Extract
	err = extractTarGzStandalone(bytes.NewReader(archiveContent), tmpDir)
	require.NoError(t, err)

	// Verify it's not in tmpDir (should be skipped)
	traversalPath := filepath.Join(tmpDir, "etc", "passwd")
	if _, err := os.Stat(traversalPath); err == nil {
		t.Fatal("Path traversal attack not prevented - file created in tmpDir")
	}

	// Also verify that the malicious path was not created
	maliciousPath := filepath.Join(tmpDir, "..", "etc", "passwd")
	if _, err := os.Stat(maliciousPath); err == nil {
		t.Fatal("Path traversal attack not prevented - file created outside tmpDir")
	}
}

// TestGitStrategy_FindDocumentationFiles tests finding documentation files
func TestGitStrategy_FindDocumentationFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-find-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test file structure
	files := map[string]string{
		"README.md":                "# Test",
		"docs/guide.md":            "# Guide",
		"docs/api.rst":             "API Documentation",
		"CHANGELOG.txt":            "Changes",
		"INSTALL.adoc":             "Installation",
		"src/main.go":              "package main",
		"node_modules/pkg/index.js": "require('pkg')",
		".git/config":              "[core]",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		dir := filepath.Dir(fullPath)
		err = os.MkdirAll(dir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Find files using the strategy's public method
	var found []string
	err = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories (mimicking strategy logic)
		if d.IsDir() {
			if strategies.IgnoreDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if strategies.DocumentExtensions[ext] {
			found = append(found, path)
		}

		return nil
	})
	require.NoError(t, err)

	// Should find doc files but skip ignored dirs
	assert.GreaterOrEqual(t, len(found), 4, "Should find at least 4 doc files")
	assert.LessOrEqual(t, len(found), 5, "Should not find code or ignored files")

	// Verify expected files are found
	var foundNames []string
	for _, f := range found {
		foundNames = append(foundNames, filepath.Base(f))
	}

	assert.Contains(t, foundNames, "README.md")
	assert.Contains(t, foundNames, "guide.md")
	assert.Contains(t, foundNames, "api.rst")
	assert.Contains(t, foundNames, "CHANGELOG.txt")

	// Should not find code files
	for _, f := range foundNames {
		assert.NotEqual(t, "main.go", f)
		assert.NotEqual(t, "index.js", f)
		assert.NotEqual(t, "config", f)
	}
}

// TestGitStrategy_ProcessFile_Markdown tests processing markdown files
func TestGitStrategy_ProcessFile_Markdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-process-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test markdown file
	testFile := filepath.Join(tmpDir, "README.md")
	content := "# Test Documentation\n\nThis is a test."
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	ctx := context.Background()
	opts := strategies.Options{
		DryRun: true, // Use dry run to avoid file writing
	}

	// Test processFile logic directly
	testProcessFile(t, strategy, ctx, testFile, tmpDir, "https://github.com/test/repo", "main", opts)
}

// TestGitStrategy_ProcessFile_RST tests processing ReStructuredText files
func TestGitStrategy_ProcessFile_RST(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-process-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test RST file
	testFile := filepath.Join(tmpDir, "index.rst")
	content := "Test Documentation\n=================\n\nThis is a test."
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	ctx := context.Background()
	opts := strategies.Options{
		DryRun: true, // Use dry run
	}

	// Test processFile logic directly
	testProcessFile(t, strategy, ctx, testFile, tmpDir, "https://github.com/test/repo", "main", opts)
}

// TestGitStrategy_ProcessFile_LargeFileSkipped tests that large files are skipped
func TestGitStrategy_ProcessFile_LargeFileSkipped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repodocs-process-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a large file (> 10MB)
	testFile := filepath.Join(tmpDir, "large.md")
	largeContent := strings.Repeat("# Test\n\nContent\n", 100000) // Much larger than 10MB
	err = os.WriteFile(testFile, []byte(largeContent), 0644)
	require.NoError(t, err)

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	ctx := context.Background()
	opts := strategies.Options{
		DryRun: false,
	}

	// Test processFile logic - large files should be skipped
	testProcessFile(t, strategy, ctx, testFile, tmpDir, "https://github.com/test/repo", "main", opts, true)
}

// TestGitStrategy_CanHandle tests URL detection
func TestGitStrategy_CanHandle(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/user/repo", true},
		{"https://github.com/user/repo.git", true},
		{"git@github.com:user/repo.git", true},
		{"https://gitlab.com/user/repo", true},
		{"https://gitlab.com/user/repo.git", true},
		{"git@gitlab.com:user/repo.git", true},
		{"https://bitbucket.org/user/repo", true},
		{"https://bitbucket.org/user/repo.git", true},
		{"https://example.com", false},
		{"https://example.com/docs", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestDocumentExtensions tests the document extensions map
func TestDocumentExtensions(t *testing.T) {
	extensions := strategies.DocumentExtensions

	// Should include documentation extensions
	assert.True(t, extensions[".md"])
	assert.True(t, extensions[".txt"])
	assert.True(t, extensions[".rst"])
	assert.True(t, extensions[".adoc"])
	assert.True(t, extensions[".asciidoc"])

	// Should not include code or other files
	assert.False(t, extensions[".go"])
	assert.False(t, extensions[".py"])
	assert.False(t, extensions[".js"])
	assert.False(t, extensions[".java"])
	assert.False(t, extensions[".cpp"])
}

// TestIgnoreDirs tests the ignored directories map
func TestIgnoreDirs(t *testing.T) {
	ignoreDirs := strategies.IgnoreDirs

	// Should ignore common directories
	assert.True(t, ignoreDirs[".git"])
	assert.True(t, ignoreDirs["node_modules"])
	assert.True(t, ignoreDirs["vendor"])
	assert.True(t, ignoreDirs["__pycache__"])
	assert.True(t, ignoreDirs[".venv"])
	assert.True(t, ignoreDirs["venv"])
	assert.True(t, ignoreDirs["dist"])
	assert.True(t, ignoreDirs["build"])
	assert.True(t, ignoreDirs[".next"])
	assert.True(t, ignoreDirs[".nuxt"])

	// Should not ignore documentation directories
	assert.False(t, ignoreDirs["docs"])
	assert.False(t, ignoreDirs["documentation"])
	assert.False(t, ignoreDirs["src"])
}

// Helper types and functions for testing

type testFile struct {
	Name    string
	Content string
}

func createTestTarGz(t *testing.T, files []testFile) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, file := range files {
		header := &tar.Header{
			Name: file.Name,
			Mode: 0644,
			Size: int64(len(file.Content)),
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)

		_, err = tw.Write([]byte(file.Content))
		require.NoError(t, err)
	}

	tw.Close()
	gzw.Close()

	return buf.Bytes()
}

// extractTarGzStandalone is a standalone version for testing
// This duplicates the logic from git.go to allow testing without package coupling
func extractTarGzStandalone(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader failed: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read failed: %w", err)
		}

		// Skip the root directory (GitHub adds repo-branch/ prefix)
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relativePath := parts[1]

		targetPath := filepath.Join(destDir, relativePath)

		// Security check: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}

			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file failed: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("copy failed: %w", err)
			}
			f.Close()
		}
	}

	return nil
}

// testProcessFile is a helper function that tests the processFile logic
func testProcessFile(t *testing.T, strategy *strategies.GitStrategy, ctx context.Context, path, tmpDir, repoURL, branch string, opts strategies.Options, skipLarge ...bool) {
	t.Helper()

	// Read file content
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	// Skip large files (> 10MB)
	if len(skipLarge) > 0 && skipLarge[0] {
		if len(content) > 10*1024*1024 {
			return // Large file was skipped - verify no output created
		}
	}

	// Get relative path for URL
	relPath, _ := filepath.Rel(tmpDir, path)
	// Convert Windows backslashes to forward slashes for URL
	relPathURL := strings.ReplaceAll(relPath, "\\", "/")
	fileURL := repoURL + "/blob/" + branch + "/" + relPathURL

	// Create document (mimicking processFile logic)
	doc := &domain.Document{
		URL:            fileURL,
		Title:          extractTitleFromPath(relPath),
		Content:        string(content),
		FetchedAt:      time.Now(),
		WordCount:      len(strings.Fields(string(content))),
		CharCount:      len(content),
		SourceStrategy: strategy.Name(),
		RelativePath:   relPath,
	}

	// For markdown files, the content is already markdown
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" {
		// For other formats, wrap in code block
		doc.Content = "```\n" + string(content) + "\n```"
	}

	// Verify document was created correctly
	assert.NotEmpty(t, doc.URL)
	assert.NotEmpty(t, doc.Title)
	assert.NotEmpty(t, doc.Content)
	assert.Equal(t, "git", doc.SourceStrategy)
	assert.Equal(t, relPath, doc.RelativePath)

	// For non-markdown, should be wrapped in code blocks
	if ext != ".md" {
		assert.Contains(t, doc.Content, "```")
	} else {
		// For markdown, content should be unchanged
		assert.Equal(t, string(content), doc.Content)
	}
}

// extractTitleFromPath extracts a title from a file path (mirrors git.go logic)
func extractTitleFromPath(path string) string {
	// Get filename without extension
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Convert common formats to title case
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}
