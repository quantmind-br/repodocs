package strategies

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGitClient is a mock implementation of git.Client for testing
type mockGitClient struct {
	cloneFunc func(ctx context.Context, path string, isBare bool, opts *gogit.CloneOptions) (*gogit.Repository, error)
}

func (m *mockGitClient) PlainCloneContext(ctx context.Context, path string, isBare bool, opts *gogit.CloneOptions) (*gogit.Repository, error) {
	if m.cloneFunc != nil {
		return m.cloneFunc(ctx, path, isBare, opts)
	}
	return nil, nil
}

// TestNewWikiStrategy tests creating a new wiki strategy
func TestNewWikiStrategy(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	assert.NotNil(t, strategy)
	assert.NotNil(t, strategy.deps)
	assert.NotNil(t, strategy.writer)
	assert.NotNil(t, strategy.logger)
	assert.NotNil(t, strategy.gitClient)
}

// TestWikiStrategy_Name tests the Name method
func TestWikiStrategy_Name(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	assert.Equal(t, "wiki", strategy.Name())
}

// TestWikiStrategy_SetGitClient tests the SetGitClient method
func TestWikiStrategy_SetGitClient(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)
	originalClient := strategy.gitClient

	mockClient := &mockGitClient{}
	strategy.SetGitClient(mockClient)

	assert.NotEqual(t, originalClient, strategy.gitClient)
	assert.Equal(t, mockClient, strategy.gitClient)
}

// TestIsWikiURL tests the IsWikiURL function
func TestIsWikiURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		// GitHub wiki URLs
		{"https://github.com/owner/repo/wiki", true},
		{"https://github.com/owner/repo/wiki/", true},
		{"https://github.com/owner/repo/wiki/Page-Name", true},
		{"https://github.com/owner/repo/wiki/Getting-Started", true},
		{"http://github.com/owner/repo/wiki", true},
		{"https://GITHUB.COM/OWNER/REPO/WIKI", true},
		// Git clone URLs
		{"https://github.com/owner/repo.wiki.git", true},
		{"http://github.com/owner/repo.wiki.git", true},
		{"https://github.com/owner/repo.Wiki.Git", true},
		// Not wiki URLs
		{"https://github.com/owner/repo", false},
		{"https://github.com/owner/repo.git", false},
		{"https://gitlab.com/owner/repo/wiki", false},
		{"https://example.com/wiki", false},
		{"https://github.com/owner/repo/blob/main/README.md", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := IsWikiURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWikiStrategy_CanHandle tests the CanHandle method
func TestWikiStrategy_CanHandle(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/owner/repo/wiki", true},
		{"https://github.com/owner/repo/wiki/Page-Name", true},
		{"https://github.com/owner/repo.wiki.git", true},
		{"https://github.com/owner/repo", false},
		{"https://gitlab.com/owner/repo/wiki", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWikiStrategy_Execute_Success tests successful wiki extraction
func TestWikiStrategy_Execute_Success(t *testing.T) {
	// Create a temporary directory to simulate wiki content
	tmpDir := t.TempDir()

	// Create some wiki pages
	pages := map[string]string{
		"Home.md":            "# Home\n\nWelcome to the wiki.",
		"Getting-Started.md": "# Getting Started\n\nHow to get started.",
		"API.md":             "# API\n\nAPI documentation.",
		"_Sidebar.md":        "[[Home]]\n[[Getting Started]]\n[[API]]",
	}

	for filename, content := range pages {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{
			BaseDir: t.TempDir(),
			Flat:    false,
		}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	mockClient := &mockGitClient{
		cloneFunc: func(ctx context.Context, path string, isBare bool, opts *gogit.CloneOptions) (*gogit.Repository, error) {
			// Simulate cloning by copying our test files
			for filename, content := range pages {
				err := os.WriteFile(filepath.Join(path, filename), []byte(content), 0644)
				if err != nil && !os.IsExist(err) {
					return nil, err
				}
			}
			return nil, nil
		},
	}

	strategy := NewWikiStrategy(deps)
	strategy.SetGitClient(mockClient)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	}

	// Parse the wiki URL first to get info
	wikiInfo, err := ParseWikiURL("https://github.com/owner/repo/wiki")
	require.NoError(t, err)

	// Directly test parseWikiStructure and processPages
	structure, err := strategy.parseWikiStructure(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, 4, len(structure.Pages)) // All pages including special ones
	assert.True(t, structure.HasSidebar)

	// Count non-special pages
	nonSpecialCount := 0
	for _, page := range structure.Pages {
		if !page.IsSpecial {
			nonSpecialCount++
		}
	}
	assert.Equal(t, 3, nonSpecialCount)

	// Test processPages
	err = strategy.processPages(ctx, structure, wikiInfo, opts)
	assert.NoError(t, err)
}

// TestWikiStrategy_ParseWikiStructure tests parsing wiki structure
func TestWikiStrategy_ParseWikiStructure(t *testing.T) {
	t.Run("with sidebar", func(t *testing.T) {
		tmpDir := t.TempDir()

		pages := map[string]string{
			"Home.md":     "# Home",
			"Guide.md":    "# Guide",
			"_Sidebar.md": "[[Home]]\n[[Guide]]",
		}

		for filename, content := range pages {
			err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}

		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		strategy := NewWikiStrategy(deps)

		structure, err := strategy.parseWikiStructure(tmpDir)
		require.NoError(t, err)
		assert.True(t, structure.HasSidebar)
		assert.Equal(t, 3, len(structure.Pages)) // Home, Guide, and _Sidebar
		assert.NotEmpty(t, structure.Sections)
	})

	t.Run("without sidebar", func(t *testing.T) {
		tmpDir := t.TempDir()

		pages := map[string]string{
			"Home.md":  "# Home",
			"Guide.md": "# Guide",
		}

		for filename, content := range pages {
			err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}

		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		strategy := NewWikiStrategy(deps)

		structure, err := strategy.parseWikiStructure(tmpDir)
		require.NoError(t, err)
		assert.False(t, structure.HasSidebar)
		assert.Equal(t, 2, len(structure.Pages))
		assert.Len(t, structure.Sections, 1) // Default section
	})

	t.Run("with special files only", func(t *testing.T) {
		tmpDir := t.TempDir()

		pages := map[string]string{
			"_Footer.md":  "Footer content",
			"_Sidebar.md": "Sidebar content",
		}

		for filename, content := range pages {
			err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}

		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		strategy := NewWikiStrategy(deps)

		structure, err := strategy.parseWikiStructure(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, 2, len(structure.Pages))
		assert.Equal(t, 0, len(structure.Sections))
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		strategy := NewWikiStrategy(deps)

		structure, err := strategy.parseWikiStructure(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, 0, len(structure.Pages))
	})
}

// TestWikiStrategy_ProcessPage tests processing a single page
func TestWikiStrategy_ProcessPage(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{
			BaseDir: t.TempDir(),
			Flat:    false,
		}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	page := &WikiPage{
		Filename:  "Getting-Started.md",
		Title:     "Getting Started",
		Content:   "# Getting Started\n\nThis is a guide.\n[[Home]]",
		IsHome:    false,
		IsSpecial: false,
		Section:   "Documentation",
	}

	structure := &WikiStructure{
		Pages: map[string]*WikiPage{
			"Home.md": {
				Filename: "Home.md",
				Title:    "Home",
				Content:  "# Home",
			},
		},
		Sections: []WikiSection{
			{Name: "Documentation", Order: 1, Pages: []string{"Getting-Started.md"}},
		},
	}

	ctx := context.Background()
	opts := Options{
		NoFolders: false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	}

	err := strategy.processPage(ctx, page, structure, "https://github.com/owner/repo/wiki", opts)
	assert.NoError(t, err)
}

// TestWikiStrategy_CloneWiki tests the cloneWiki method
func TestWikiStrategy_CloneWiki(t *testing.T) {
	t.Run("successful clone", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		mockClient := &mockGitClient{
			cloneFunc: func(ctx context.Context, path string, isBare bool, opts *gogit.CloneOptions) (*gogit.Repository, error) {
				// Create a file to simulate successful clone
				err := os.WriteFile(filepath.Join(path, "Home.md"), []byte("# Home"), 0644)
				return nil, err
			},
		}

		strategy := NewWikiStrategy(deps)
		strategy.SetGitClient(mockClient)

		ctx := context.Background()
		err := strategy.cloneWiki(ctx, "https://github.com/owner/repo.wiki.git", tmpDir)
		assert.NoError(t, err)
	})

	t.Run("clone failure", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		mockClient := &mockGitClient{
			cloneFunc: func(ctx context.Context, path string, isBare bool, opts *gogit.CloneOptions) (*gogit.Repository, error) {
				return nil, assert.AnError
			},
		}

		strategy := NewWikiStrategy(deps)
		strategy.SetGitClient(mockClient)

		ctx := context.Background()
		err := strategy.cloneWiki(ctx, "https://github.com/owner/repo.wiki.git", tmpDir)
		assert.Error(t, err)
	})
}

// TestWikiStrategy_ProcessPages tests the processPages method
func TestWikiStrategy_ProcessPages(t *testing.T) {
	t.Run("with limit", func(t *testing.T) {
		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{
				BaseDir: t.TempDir(),
				Flat:    true,
			}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		strategy := NewWikiStrategy(deps)

		structure := &WikiStructure{
			Pages: map[string]*WikiPage{
				"Home.md":    {Filename: "Home.md", Title: "Home", Content: "# Home", IsSpecial: false},
				"Guide.md":   {Filename: "Guide.md", Title: "Guide", Content: "# Guide", IsSpecial: false},
				"API.md":     {Filename: "API.md", Title: "API", Content: "# API", IsSpecial: false},
				"_Footer.md": {Filename: "_Footer.md", Title: "Footer", Content: "Footer", IsSpecial: true},
			},
			Sections: []WikiSection{
				{Name: "Documentation", Order: 1, Pages: []string{"Home.md", "Guide.md", "API.md"}},
			},
		}

		wikiInfo := &WikiInfo{
			Owner: "owner",
			Repo:  "repo",
		}

		ctx := context.Background()
		opts := Options{
			CommonOptions: domain.CommonOptions{
				Limit:  2,
				DryRun: true,
			},
		}

		err := strategy.processPages(ctx, structure, wikiInfo, opts)
		assert.NoError(t, err)
	})

	t.Run("no processable pages", func(t *testing.T) {
		deps := &Dependencies{
			Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
			Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		}

		strategy := NewWikiStrategy(deps)

		structure := &WikiStructure{
			Pages: map[string]*WikiPage{
				"_Sidebar.md": {Filename: "_Sidebar.md", Title: "Sidebar", IsSpecial: true},
				"_Footer.md":  {Filename: "_Footer.md", Title: "Footer", IsSpecial: true},
			},
			Sections: []WikiSection{},
		}

		wikiInfo := &WikiInfo{
			Owner: "owner",
			Repo:  "repo",
		}

		ctx := context.Background()
		opts := Options{
			CommonOptions: domain.CommonOptions{
				DryRun: true,
			},
		}

		err := strategy.processPages(ctx, structure, wikiInfo, opts)
		assert.NoError(t, err)
	})
}

// TestWikiStrategy_Execute_WithMarkdownExtension tests with .markdown extension
func TestWikiStrategy_Execute_WithMarkdownExtension(t *testing.T) {
	tmpDir := t.TempDir()

	pages := map[string]string{
		"Home.markdown": "# Home\n\nWelcome.",
	}

	for filename, content := range pages {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	structure, err := strategy.parseWikiStructure(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(structure.Pages))
}

// TestWikiStrategy_ContextCancellation tests context cancellation
func TestWikiStrategy_ContextCancellation(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	mockClient := &mockGitClient{
		cloneFunc: func(ctx context.Context, path string, isBare bool, opts *gogit.CloneOptions) (*gogit.Repository, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// Create a file
				err := os.WriteFile(filepath.Join(path, "Home.md"), []byte("# Home"), 0644)
				return nil, err
			}
		},
	}

	strategy := NewWikiStrategy(deps)
	strategy.SetGitClient(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	}

	err := strategy.Execute(ctx, "https://github.com/owner/repo/wiki", opts)
	assert.Error(t, err)
}

// TestWikiStrategy_Execute_URLParseError tests with invalid URL
func TestWikiStrategy_Execute_URLParseError(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	}

	err := strategy.Execute(ctx, "https://example.com/not-a-wiki", opts)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "invalid wiki url")
}

// TestWikiStrategy_BuildRelativePathEdgeCases tests edge cases for relative path building
func TestWikiStrategy_BuildRelativePathEdgeCases(t *testing.T) {
	page := &WikiPage{
		Filename:  "API_Guide.md",
		Title:     "API Guide",
		Content:   "# API Guide",
		IsHome:    false,
		IsSpecial: false,
		Section:   "API Reference",
	}

	structure := &WikiStructure{
		Sections: []WikiSection{{Name: "API Reference"}},
	}

	t.Run("flat mode", func(t *testing.T) {
		result := BuildRelativePath(page, structure, true)
		assert.Equal(t, "api_guide.md", result)
	})

	t.Run("no sections", func(t *testing.T) {
		page.Section = ""
		structure.Sections = []WikiSection{}
		result := BuildRelativePath(page, structure, false)
		assert.Equal(t, "api_guide.md", result)
	})

	t.Run("with section", func(t *testing.T) {
		page.Section = "API Reference"
		structure.Sections = []WikiSection{{Name: "API Reference"}}
		result := BuildRelativePath(page, structure, false)
		assert.Equal(t, "api-reference/api_guide.md", result)
	})
}

// TestWikiStrategy_ConvertWikiLinks tests wiki link conversion in pages
func TestWikiStrategy_ConvertWikiLinks(t *testing.T) {
	content := `# Guide

See [[Home]] for more info.
Also check [[API Reference]] for details.
[[Installation|Install Now]]

Section link: [[Advanced#Configuration]]
`

	pages := map[string]*WikiPage{
		"Home.md":          {Filename: "Home.md", Title: "Home"},
		"API-Reference.md": {Filename: "API-Reference.md", Title: "API Reference"},
	}

	result := ConvertWikiLinks(content, pages)

	assert.Contains(t, result, "[Home](./home.md)")
	assert.Contains(t, result, "[API Reference](./api-reference.md)")
	assert.Contains(t, result, "[Install Now](./installation.md)")
	assert.Contains(t, result, "#configuration")
}

// TestWikiStrategy_NonExistentDirectory tests with non-existent directory
func TestWikiStrategy_NonExistentDirectory(t *testing.T) {
	deps := &Dependencies{
		Writer: output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewWikiStrategy(deps)

	_, err := strategy.parseWikiStructure("/non/existent/directory")
	assert.Error(t, err)
}
