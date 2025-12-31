package app_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikiStrategy_NewWikiStrategy(t *testing.T) {
	deps := createTestWikiDependencies(t)
	strategy := strategies.NewWikiStrategy(deps)

	require.NotNil(t, strategy)
	assert.Equal(t, "wiki", strategy.Name())
}

func TestWikiStrategy_Name(t *testing.T) {
	deps := createTestWikiDependencies(t)
	strategy := strategies.NewWikiStrategy(deps)

	assert.Equal(t, "wiki", strategy.Name())
}

func TestWikiStrategy_CanHandle(t *testing.T) {
	deps := createTestWikiDependencies(t)
	strategy := strategies.NewWikiStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "wiki URL at root",
			url:      "https://github.com/Alexays/Waybar/wiki",
			expected: true,
		},
		{
			name:     "wiki URL with trailing slash",
			url:      "https://github.com/owner/repo/wiki/",
			expected: true,
		},
		{
			name:     "wiki URL with specific page",
			url:      "https://github.com/owner/repo/wiki/Configuration",
			expected: true,
		},
		{
			name:     "wiki.git clone URL",
			url:      "https://github.com/owner/repo.wiki.git",
			expected: true,
		},
		{
			name:     "SSH wiki clone URL",
			url:      "git@github.com:owner/repo.wiki.git",
			expected: true,
		},
		{
			name:     "regular GitHub repo",
			url:      "https://github.com/owner/repo",
			expected: false,
		},
		{
			name:     "GitHub blob URL",
			url:      "https://github.com/owner/repo/blob/main/README.md",
			expected: false,
		},
		{
			name:     "non-GitHub URL",
			url:      "https://example.com/docs",
			expected: false,
		},
		{
			name:     "sitemap URL",
			url:      "https://example.com/sitemap.xml",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWikiURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "github wiki URL",
			url:      "https://github.com/owner/repo/wiki",
			expected: true,
		},
		{
			name:     "github wiki with page",
			url:      "https://github.com/Alexays/Waybar/wiki/Configuration",
			expected: true,
		},
		{
			name:     "wiki.git URL",
			url:      "https://github.com/owner/repo.wiki.git",
			expected: true,
		},
		{
			name:     "case insensitive wiki.git",
			url:      "https://github.com/owner/repo.WIKI.GIT",
			expected: true,
		},
		{
			name:     "regular repo",
			url:      "https://github.com/owner/repo",
			expected: false,
		},
		{
			name:     "gitlab URL",
			url:      "https://gitlab.com/owner/repo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategies.IsWikiURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseWikiURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOwner   string
		wantRepo    string
		wantClone   string
		wantPage    string
		expectError bool
	}{
		{
			name:      "standard wiki URL",
			url:       "https://github.com/Alexays/Waybar/wiki",
			wantOwner: "Alexays",
			wantRepo:  "Waybar",
			wantClone: "https://github.com/Alexays/Waybar.wiki.git",
			wantPage:  "",
		},
		{
			name:      "wiki URL with trailing slash",
			url:       "https://github.com/owner/repo/wiki/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantClone: "https://github.com/owner/repo.wiki.git",
			wantPage:  "",
		},
		{
			name:      "wiki URL with specific page",
			url:       "https://github.com/owner/repo/wiki/Configuration",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantClone: "https://github.com/owner/repo.wiki.git",
			wantPage:  "Configuration",
		},
		{
			name:      "direct clone URL",
			url:       "https://github.com/owner/repo.wiki.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantClone: "https://github.com/owner/repo.wiki.git",
			wantPage:  "",
		},
		{
			name:        "invalid URL",
			url:         "https://example.com/not-github",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := strategies.ParseWikiURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.wantOwner, info.Owner)
			assert.Equal(t, tt.wantRepo, info.Repo)
			assert.Equal(t, tt.wantClone, info.CloneURL)
			assert.Equal(t, tt.wantPage, info.TargetPage)
			assert.Equal(t, "github", info.Platform)
		})
	}
}

func TestFilenameToTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Getting-Started.md", "Getting Started"},
		{"API_Reference.md", "API Reference"},
		{"Home.md", "Home"},
		{"advanced-configuration.md", "Advanced Configuration"},
		{"CHANGELOG.md", "CHANGELOG"},
		{"simple.md", "Simple"},
		{"multi-word-file-name.md", "Multi Word File Name"},
		{"mixed_and-delimiters.md", "Mixed And Delimiters"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strategies.FilenameToTitle(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTitleToFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Getting Started", "Getting-Started"},
		{"API Reference", "API-Reference"},
		{"Home", "Home"},
		{"Single", "Single"},
		{"Multiple Word Title", "Multiple-Word-Title"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strategies.TitleToFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertWikiLinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple wiki link",
			input:    "See [[Getting Started]] for more info",
			expected: "See [Getting Started](./getting-started.md) for more info",
		},
		{
			name:     "wiki link with custom text",
			input:    "Check the [[Configuration|config page]]",
			expected: "Check the [config page](./configuration.md)",
		},
		{
			name:     "wiki link with section anchor",
			input:    "Go to [[Setup#Installation]]",
			expected: "Go to [Setup](./setup.md#installation)",
		},
		{
			name:     "multiple wiki links",
			input:    "See [[Home]] and [[Configuration]]",
			expected: "See [Home](./home.md) and [Configuration](./configuration.md)",
		},
		{
			name:     "no wiki links",
			input:    "This is regular markdown [link](https://example.com)",
			expected: "This is regular markdown [link](https://example.com)",
		},
		{
			name:     "wiki link with multi-word title",
			input:    "Read [[Advanced Configuration]]",
			expected: "Read [Advanced Configuration](./advanced-configuration.md)",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategies.ConvertWikiLinks(tt.input, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSidebarContent(t *testing.T) {
	sidebar := `## Getting Started
* [[Home]]
* [[Installation]]

## Configuration
* [[Basic Config]]
* [[Advanced]]
`
	pages := map[string]*strategies.WikiPage{
		"Home.md":         {Filename: "Home.md"},
		"Installation.md": {Filename: "Installation.md"},
		"Basic-Config.md": {Filename: "Basic-Config.md"},
		"Advanced.md":     {Filename: "Advanced.md"},
	}

	sections := strategies.ParseSidebarContent(sidebar, pages)

	require.Len(t, sections, 2)

	assert.Equal(t, "Getting Started", sections[0].Name)
	assert.Equal(t, 1, sections[0].Order)
	require.Len(t, sections[0].Pages, 2)
	assert.Equal(t, "Home.md", sections[0].Pages[0])
	assert.Equal(t, "Installation.md", sections[0].Pages[1])

	assert.Equal(t, "Configuration", sections[1].Name)
	assert.Equal(t, 2, sections[1].Order)
	require.Len(t, sections[1].Pages, 2)
}

func TestParseSidebarContent_MarkdownLinks(t *testing.T) {
	sidebar := `# Documentation

## Guides
- [Home](Home)
- [Setup](Setup.md)

## Reference
- [API](API-Reference)
`
	pages := map[string]*strategies.WikiPage{
		"Home.md":          {Filename: "Home.md"},
		"Setup.md":         {Filename: "Setup.md"},
		"API-Reference.md": {Filename: "API-Reference.md"},
	}

	sections := strategies.ParseSidebarContent(sidebar, pages)

	require.Len(t, sections, 2)
	assert.Equal(t, "Guides", sections[0].Name)
	assert.Len(t, sections[0].Pages, 2)
	assert.Equal(t, "Reference", sections[1].Name)
	assert.Len(t, sections[1].Pages, 1)
}

func TestParseSidebarContent_Empty(t *testing.T) {
	sections := strategies.ParseSidebarContent("", nil)
	assert.Empty(t, sections)
}

func TestCreateDefaultStructure(t *testing.T) {
	pages := map[string]*strategies.WikiPage{
		"Home.md":          {Filename: "Home.md"},
		"Configuration.md": {Filename: "Configuration.md"},
		"API.md":           {Filename: "API.md"},
		"_Sidebar.md":      {Filename: "_Sidebar.md", IsSpecial: true},
	}

	sections := strategies.CreateDefaultStructure(pages)

	require.Len(t, sections, 1)
	assert.Equal(t, "Documentation", sections[0].Name)

	require.Len(t, sections[0].Pages, 3)
	assert.Equal(t, "Home.md", sections[0].Pages[0])
}

func TestBuildRelativePath(t *testing.T) {
	structure := &strategies.WikiStructure{
		Sections: []strategies.WikiSection{
			{Name: "Getting Started", Order: 1},
			{Name: "Configuration", Order: 2},
		},
	}

	tests := []struct {
		name     string
		page     *strategies.WikiPage
		flat     bool
		expected string
	}{
		{
			name:     "Home page becomes index.md",
			page:     &strategies.WikiPage{Filename: "Home.md", IsHome: true},
			flat:     false,
			expected: "index.md",
		},
		{
			name:     "Page with section",
			page:     &strategies.WikiPage{Filename: "Installation.md", Section: "Getting Started"},
			flat:     false,
			expected: "getting-started/installation.md",
		},
		{
			name:     "Page without section",
			page:     &strategies.WikiPage{Filename: "Misc.md", Section: ""},
			flat:     false,
			expected: "misc.md",
		},
		{
			name:     "Flat mode ignores section",
			page:     &strategies.WikiPage{Filename: "Installation.md", Section: "Getting Started"},
			flat:     true,
			expected: "installation.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategies.BuildRelativePath(tt.page, structure, tt.flat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func createTestWikiDependencies(t *testing.T) *strategies.Dependencies {
	t.Helper()
	return &strategies.Dependencies{}
}
