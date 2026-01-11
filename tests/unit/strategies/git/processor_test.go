package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/state"
	"github.com/quantmind-br/repodocs-go/internal/strategies/git"
)

func TestNewProcessor(t *testing.T) {
	p := git.NewProcessor(git.ProcessorOptions{})

	assert.NotNil(t, p)
}

func TestProcessor_FindDocumentationFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	p := git.NewProcessor(git.ProcessorOptions{})
	files, err := p.FindDocumentationFiles(tmpDir, "")

	assert.NoError(t, err)
	assert.Empty(t, files)
}

func TestProcessor_FindDocumentationFiles_MarkdownOnly(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Readme"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "CHANGELOG.md"), []byte("# Changelog"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "docs.md"), []byte("# Docs"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "config.txt"), []byte("config"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "image.png"), []byte("image"), 0644)

	p := git.NewProcessor(git.ProcessorOptions{})
	files, err := p.FindDocumentationFiles(tmpDir, "")

	assert.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestProcessor_FindDocumentationFiles_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	apiDir := filepath.Join(docsDir, "api")
	guideDir := filepath.Join(docsDir, "guides")

	require.NoError(t, os.MkdirAll(apiDir, 0755))
	require.NoError(t, os.MkdirAll(guideDir, 0755))

	os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Docs"), 0644)
	os.WriteFile(filepath.Join(apiDir, "api.md"), []byte("# API"), 0644)
	os.WriteFile(filepath.Join(guideDir, "install.md"), []byte("# Install"), 0644)

	p := git.NewProcessor(git.ProcessorOptions{})
	files, err := p.FindDocumentationFiles(tmpDir, "")

	assert.NoError(t, err)
	assert.Len(t, files, 3)

	for _, f := range files {
		assert.True(t, filepath.Ext(f) == ".md")
	}
}

func TestProcessor_FindDocumentationFiles_ExcludeDirs(t *testing.T) {
	tmpDir := t.TempDir()

	nodeModules := filepath.Join(tmpDir, "node_modules")
	vendor := filepath.Join(tmpDir, "vendor")
	gitDir := filepath.Join(tmpDir, ".git")
	dist := filepath.Join(tmpDir, "dist")
	build := filepath.Join(tmpDir, "build")
	docs := filepath.Join(tmpDir, "docs")

	require.NoError(t, os.MkdirAll(nodeModules, 0755))
	require.NoError(t, os.MkdirAll(vendor, 0755))
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.MkdirAll(dist, 0755))
	require.NoError(t, os.MkdirAll(build, 0755))
	require.NoError(t, os.MkdirAll(docs, 0755))

	os.WriteFile(filepath.Join(nodeModules, "package.md"), []byte("# Package"), 0644)
	os.WriteFile(filepath.Join(vendor, "vendor.md"), []byte("# Vendor"), 0644)
	os.WriteFile(filepath.Join(gitDir, "git.md"), []byte("# Git"), 0644)
	os.WriteFile(filepath.Join(dist, "dist.md"), []byte("# Dist"), 0644)
	os.WriteFile(filepath.Join(build, "build.md"), []byte("# Build"), 0644)
	os.WriteFile(filepath.Join(docs, "docs.md"), []byte("# Docs"), 0644)

	p := git.NewProcessor(git.ProcessorOptions{})
	files, err := p.FindDocumentationFiles(tmpDir, "")

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.True(t, filepath.Base(files[0]) == "docs.md")
}

func TestProcessor_FindDocumentationFiles_WithFilterPath(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	apiDir := filepath.Join(docsDir, "api")
	guidesDir := filepath.Join(docsDir, "guides")

	require.NoError(t, os.MkdirAll(apiDir, 0755))
	require.NoError(t, os.MkdirAll(guidesDir, 0755))

	os.WriteFile(filepath.Join(apiDir, "api.md"), []byte("# API"), 0644)
	os.WriteFile(filepath.Join(guidesDir, "install.md"), []byte("# Install"), 0644)
	os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Readme"), 0644)

	p := git.NewProcessor(git.ProcessorOptions{})
	files, err := p.FindDocumentationFiles(tmpDir, "docs/api")

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.True(t, filepath.Base(files[0]) == "api.md")
}

func TestProcessor_FindDocumentationFiles_FilterPathNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	p := git.NewProcessor(git.ProcessorOptions{})
	_, err := p.FindDocumentationFiles(tmpDir, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filter path does not exist")
}

func TestProcessor_FindDocumentationFiles_FilterPathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "README.md")
	os.WriteFile(filePath, []byte("# Readme"), 0644)

	p := git.NewProcessor(git.ProcessorOptions{})
	_, err := p.FindDocumentationFiles(tmpDir, "README.md")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filter path is not a directory")
}

func TestExtractTitleFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple filename",
			path:     "README.md",
			expected: "README",
		},
		{
			name:     "with extension",
			path:     "installation.md",
			expected: "Installation",
		},
		{
			name:     "nested path",
			path:     "docs/api/introduction.md",
			expected: "Introduction",
		},
		{
			name:     "with dashes",
			path:     "user-guide.md",
			expected: "User guide",
		},
		{
			name:     "with underscores",
			path:     "api_reference_v2.md",
			expected: "Api reference v2",
		},
		{
			name:     "numbered file",
			path:     "01-getting-started.md",
			expected: "01 getting started",
		},
		{
			name:     "all lowercase",
			path:     "readme.md",
			expected: "Readme",
		},
		{
			name:     "all uppercase",
			path:     "README.md",
			expected: "README",
		},
		{
			name:     "title case",
			path:     "GettingStarted.md",
			expected: "GettingStarted",
		},
		{
			name:     "with path separators",
			path:     "/absolute/path/to/file.md",
			expected: "File",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.ExtractTitleFromPath(tt.path)

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestProcessFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	content := "# Test Content\n\nThis is a test file."
	os.WriteFile(filePath, []byte(content), 0644)

	var processedDoc *domain.Document
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		processedDoc = doc
		return nil
	}

	opts := git.ProcessOptions{
		RepoURL:   "https://github.com/owner/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	p := git.NewProcessor(git.ProcessorOptions{})
	err := p.ProcessFile(context.Background(), filePath, tmpDir, opts)

	assert.NoError(t, err)
	assert.NotNil(t, processedDoc)
	assert.Equal(t, "https://github.com/owner/repo/blob/main/test.md", processedDoc.URL)
	assert.Equal(t, "Test", processedDoc.Title) // Title extracted from filename, not content
	assert.Equal(t, content, processedDoc.Content)
	assert.Equal(t, "git", processedDoc.SourceStrategy)
	assert.Equal(t, "test.md", processedDoc.RelativePath)
}

func TestProcessFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.md")

	largeContent := make([]byte, 11*1024*1024)
	os.WriteFile(filePath, largeContent, 0644)

	writeCalled := false
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		writeCalled = true
		return nil
	}

	opts := git.ProcessOptions{
		RepoURL:   "https://github.com/owner/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	p := git.NewProcessor(git.ProcessorOptions{})
	err := p.ProcessFile(context.Background(), filePath, tmpDir, opts)

	assert.NoError(t, err)
	assert.False(t, writeCalled)
}

func TestProcessFile_WithState(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	content := []byte("# Content")
	os.WriteFile(filePath, content, 0644)

	sm := state.NewManager(state.ManagerOptions{
		BaseDir:   stateDir,
		SourceURL: "https://github.com/owner/repo",
		Strategy:  "git",
	})

	// Compute the actual content hash first
	p := git.NewProcessor(git.ProcessorOptions{})

	// Create initial state file with matching hash
	sm.Update("https://github.com/owner/repo/blob/main/test.md", state.PageState{
		ContentHash: "", // Empty hash means it will be set on first load
	})
	require.NoError(t, sm.Save(context.Background()))

	// Load the state file
	require.NoError(t, sm.Load(context.Background()))

	// Now update with the actual hash that would be computed
	// First process to compute hash, then update state, then process again
	testDoc := &domain.Document{}
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		*testDoc = *doc
		return nil
	}

	opts := git.ProcessOptions{
		RepoURL:      "https://github.com/owner/repo",
		Branch:       "main",
		WriteFunc:    writeFunc,
		StateManager: sm,
	}

	// First call to get the hash
	err := p.ProcessFile(context.Background(), filePath, tmpDir, opts)
	assert.NoError(t, err)
	assert.True(t, testDoc.ContentHash != "")

	// Update state with the computed hash
	sm.Update("https://github.com/owner/repo/blob/main/test.md", state.PageState{
		ContentHash: testDoc.ContentHash,
	})
	require.NoError(t, sm.Save(context.Background()))
	require.NoError(t, sm.Load(context.Background()))

	// Now the write should NOT be called because content hasn't changed
	writeCalled := false
	writeFunc2 := func(ctx context.Context, doc *domain.Document) error {
		writeCalled = true
		return nil
	}
	opts.WriteFunc = writeFunc2

	err = p.ProcessFile(context.Background(), filePath, tmpDir, opts)
	assert.NoError(t, err)
	assert.False(t, writeCalled)
}

func TestProcessFile_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	os.WriteFile(filePath, []byte("# Content"), 0644)

	writeCalled := false
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		writeCalled = true
		return nil
	}

	opts := git.ProcessOptions{
		RepoURL:   "https://github.com/owner/repo",
		Branch:    "main",
		DryRun:    true,
		WriteFunc: writeFunc,
	}

	p := git.NewProcessor(git.ProcessorOptions{})
	err := p.ProcessFile(context.Background(), filePath, tmpDir, opts)

	assert.NoError(t, err)
	assert.False(t, writeCalled)
}

func TestProcessFile_NonMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.yaml")
	content := "key: value\n"
	os.WriteFile(filePath, []byte(content), 0644)

	var processedDoc *domain.Document
	writeFunc := func(ctx context.Context, doc *domain.Document) error {
		processedDoc = doc
		return nil
	}

	opts := git.ProcessOptions{
		RepoURL:   "https://github.com/owner/repo",
		Branch:    "main",
		WriteFunc: writeFunc,
	}

	p := git.NewProcessor(git.ProcessorOptions{})
	err := p.ProcessFile(context.Background(), filePath, tmpDir, opts)

	assert.NoError(t, err)
	assert.NotNil(t, processedDoc)
	assert.Contains(t, processedDoc.Content, "```")
	assert.Contains(t, processedDoc.Content, content)
}

func TestProcessFile_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nonexistent.md")

	p := git.NewProcessor(git.ProcessorOptions{})
	err := p.ProcessFile(context.Background(), filePath, tmpDir, git.ProcessOptions{})

	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}
