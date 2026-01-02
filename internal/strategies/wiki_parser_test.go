package strategies

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseWikiURL_GitHubWiki tests parsing GitHub wiki URLs
func TestParseWikiURL_GitHubWiki(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOwner   string
		wantRepo    string
		wantClone   string
		wantPage    string
		wantErr     bool
	}{
		{
			name:      "standard wiki URL",
			url:       "https://github.com/owner/repo/wiki",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantClone: "https://github.com/owner/repo.wiki.git",
			wantPage:  "",
		},
		{
			name:      "wiki URL with page",
			url:       "https://github.com/owner/repo/wiki/Page-Name",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantClone: "https://github.com/owner/repo.wiki.git",
			wantPage:  "Page-Name",
		},
		{
			name:      "wiki.git URL",
			url:       "https://github.com/owner/repo.wiki.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantClone: "https://github.com/owner/repo.wiki.git",
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
			name:    "invalid URL",
			url:     "https://example.com/not-a-wiki",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "missing owner",
			url:     "https://github.com//repo/wiki",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseWikiURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, info)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOwner, info.Owner)
			assert.Equal(t, tt.wantRepo, info.Repo)
			assert.Equal(t, tt.wantClone, info.CloneURL)
			assert.Equal(t, tt.wantPage, info.TargetPage)
			assert.Equal(t, "github", info.Platform)
		})
	}
}

// TestFilenameToTitle tests converting filename to title
func TestFilenameToTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Home.md", "Home"},
		{"Getting-Started.md", "Getting Started"},
		{"API_Reference.md", "API Reference"},
		{"how_to_use.md", "How To Use"},
		{"REST-API-Guide.mdx", "REST API Guide"},
		{"underscore_file_name.md", "Underscore File Name"},
		{"mixed-separator_file.md", "Mixed Separator File"},
		{"Single.md", "Single"},
		{"123-Numbers.md", "123 Numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FilenameToTitle(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTitleToFilename tests converting title to filename
func TestTitleToFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Getting Started", "Getting-Started"},
		{"API Reference", "API-Reference"},
		{"How To Use", "How-To-Use"},
		{"REST API Guide", "REST-API-Guide"},
		{"Single", "Single"},
		{"Multiple   Spaces", "Multiple---Spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TitleToFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseSidebarContent tests parsing sidebar content
func TestParseSidebarContent(t *testing.T) {
	t.Run("sidebar with sections and links", func(t *testing.T) {
		content := `# Getting Started
[[Home]]
[[Installation]]

# API Reference
[[API-Overview|API Overview]]
[[Endpoints]]

[Advanced Guide](advanced-guide)
`

		pages := map[string]*WikiPage{
			"Home.md":        {Filename: "Home.md", Title: "Home"},
			"Installation.md": {Filename: "Installation.md", Title: "Installation"},
			"API-Overview.md": {Filename: "API-Overview.md", Title: "API Overview"},
			"Endpoints.md":    {Filename: "Endpoints.md", Title: "Endpoints"},
		}

		sections := ParseSidebarContent(content, pages)

		assert.Len(t, sections, 2)
		assert.Equal(t, "Getting Started", sections[0].Name)
		assert.Equal(t, 2, len(sections[0].Pages))
		assert.Equal(t, "API Reference", sections[1].Name)
		assert.Equal(t, "Home.md", sections[0].Pages[0])
		assert.Equal(t, "Installation.md", sections[0].Pages[1])
	})

	t.Run("sidebar without sections", func(t *testing.T) {
		content := `[[Home]]
[[Getting-Started|Getting Started]]
[[API-Reference]]
`

		pages := map[string]*WikiPage{
			"Home.md":         {Filename: "Home.md", Title: "Home"},
			"Getting-Started.md": {Filename: "Getting-Started.md", Title: "Getting Started"},
			"API-Reference.md": {Filename: "API-Reference.md", Title: "API Reference"},
		}

		sections := ParseSidebarContent(content, pages)

		assert.Len(t, sections, 1)
		assert.Equal(t, "General", sections[0].Name)
		assert.Equal(t, 3, len(sections[0].Pages))
	})

	t.Run("empty sidebar", func(t *testing.T) {
		content := ``
		pages := map[string]*WikiPage{}

		sections := ParseSidebarContent(content, pages)

		assert.Len(t, sections, 0)
	})

	t.Run("sidebar with markdown links", func(t *testing.T) {
		content := `[Home](Home.md)
[Installation](installation.md)
[API Guide](api-guide.md)
`

		pages := map[string]*WikiPage{
			"Home.md":        {Filename: "Home.md", Title: "Home"},
			"installation.md": {Filename: "installation.md", Title: "Installation"},
			"api-guide.md":   {Filename: "api-guide.md", Title: "API Guide"},
		}

		sections := ParseSidebarContent(content, pages)

		assert.Len(t, sections, 1)
		assert.Equal(t, 3, len(sections[0].Pages))
	})

	t.Run("sidebar with mixed link types", func(t *testing.T) {
		content := `# Section 1
[[Home]]
[Link](page.md)

# Section 2
[[Another]]
`

		pages := map[string]*WikiPage{
			"Home.md":  {Filename: "Home.md", Title: "Home"},
			"page.md":  {Filename: "page.md", Title: "Page"},
			"Another.md": {Filename: "Another.md", Title: "Another"},
		}

		sections := ParseSidebarContent(content, pages)

		assert.Len(t, sections, 2)
	})
}

// TestFindPageFilename tests finding page filename by various name formats
func TestFindPageFilename(t *testing.T) {
	pages := map[string]*WikiPage{
		"Home.md":              {Filename: "Home.md"},
		"Getting-Started.md":   {Filename: "Getting-Started.md"},
		"API_Reference.md":     {Filename: "API_Reference.md"},
		"installation-guide.md": {Filename: "installation-guide.md"},
	}

	tests := []struct {
		name     string
		pageName string
		expected string
	}{
		{"exact match", "Home", "Home.md"},
		{"exact match with extension", "Home.md", "Home.md"},
		{"hyphenated match", "Getting Started", "Getting-Started.md"},
		{"case insensitive match", "api_reference", "API_Reference.md"},
		{"hyphenated case insensitive", "installation guide", "installation-guide.md"},
		{"not found", "NonExistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findPageFilename(tt.pageName, pages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCreateDefaultStructure tests creating default wiki structure
func TestCreateDefaultStructure(t *testing.T) {
	t.Run("with Home page", func(t *testing.T) {
		pages := map[string]*WikiPage{
			"Home.md":          {Filename: "Home.md", Title: "Home", IsSpecial: false},
			"API.md":           {Filename: "API.md", Title: "API", IsSpecial: false},
			"Guide.md":         {Filename: "Guide.md", Title: "Guide", IsSpecial: false},
			"_Footer.md":       {Filename: "_Footer.md", Title: "Footer", IsSpecial: true},
		}

		sections := CreateDefaultStructure(pages)

		assert.Len(t, sections, 1)
		assert.Equal(t, "Documentation", sections[0].Name)
		assert.Equal(t, 3, len(sections[0].Pages))
		assert.Equal(t, "Home.md", sections[0].Pages[0]) // Home should be first

		// Check page order
		assert.Equal(t, 1, pages["Home.md"].Order)
		assert.Equal(t, "Documentation", pages["Home.md"].Section)
	})

	t.Run("without Home page", func(t *testing.T) {
		pages := map[string]*WikiPage{
			"API.md":    {Filename: "API.md", Title: "API", IsSpecial: false},
			"Guide.md":  {Filename: "Guide.md", Title: "Guide", IsSpecial: false},
			"_Sidebar.md": {Filename: "_Sidebar.md", Title: "Sidebar", IsSpecial: true},
		}

		sections := CreateDefaultStructure(pages)

		assert.Len(t, sections, 1)
		assert.Equal(t, 2, len(sections[0].Pages))
	})

	t.Run("alphabetical sorting", func(t *testing.T) {
		pages := map[string]*WikiPage{
			"Zebra.md":  {Filename: "Zebra.md", Title: "Zebra", IsSpecial: false},
			"Apple.md":  {Filename: "Apple.md", Title: "Apple", IsSpecial: false},
			"Banana.md": {Filename: "Banana.md", Title: "Banana", IsSpecial: false},
		}

		sections := CreateDefaultStructure(pages)

		assert.Equal(t, "Apple.md", sections[0].Pages[0])
		assert.Equal(t, "Banana.md", sections[0].Pages[1])
		assert.Equal(t, "Zebra.md", sections[0].Pages[2])
	})

	t.Run("only special files", func(t *testing.T) {
		pages := map[string]*WikiPage{
			"_Sidebar.md": {Filename: "_Sidebar.md", Title: "Sidebar", IsSpecial: true},
			"_Footer.md":  {Filename: "_Footer.md", Title: "Footer", IsSpecial: true},
		}

		sections := CreateDefaultStructure(pages)

		assert.Len(t, sections, 1)
		assert.Len(t, sections[0].Pages, 0)
	})

	t.Run("empty pages", func(t *testing.T) {
		pages := map[string]*WikiPage{}

		sections := CreateDefaultStructure(pages)

		assert.Len(t, sections, 1)
		assert.Len(t, sections[0].Pages, 0)
	})
}

// TestConvertWikiLinks tests converting wiki links to markdown
func TestConvertWikiLinks(t *testing.T) {
	t.Run("simple wiki links", func(t *testing.T) {
		content := `[[Home]]
[[Getting Started]]
[[API Reference]]
`

		result := ConvertWikiLinks(content, nil)

		assert.Contains(t, result, "[Home](./home.md)")
		assert.Contains(t, result, "[Getting Started](./getting-started.md)")
		assert.Contains(t, result, "[API Reference](./api-reference.md)")
	})

	t.Run("wiki links with custom text", func(t *testing.T) {
		content := `[[Home|Return to Home]]
[[Getting Started|Start Here]]
[[API|View API]]
`

		result := ConvertWikiLinks(content, nil)

		assert.Contains(t, result, "[Return to Home](./home.md)")
		assert.Contains(t, result, "[Start Here](./getting-started.md)")
		assert.Contains(t, result, "[View API](./api.md)")
	})

	t.Run("wiki links with sections", func(t *testing.T) {
		content := `[[Installation#Quick Start]]
[[API#Authentication]]
[[Guide#Advanced Usage]]
`

		result := ConvertWikiLinks(content, nil)

		assert.Contains(t, result, "[Installation](./installation.md#quick-start)")
		assert.Contains(t, result, "[API](./api.md#authentication)")
		assert.Contains(t, result, "[Guide](./guide.md#advanced-usage)")
	})

	t.Run("mixed link types", func(t *testing.T) {
		content := `[[Home]] | [External](https://example.com) | [[Page|Custom]]
`

		result := ConvertWikiLinks(content, nil)

		assert.Contains(t, result, "[Home](./home.md)")
		assert.Contains(t, result, "[External](https://example.com)")
		assert.Contains(t, result, "[Custom](./page.md)")
	})

	t.Run("no wiki links", func(t *testing.T) {
		content := `This is just plain text with no wiki links.
[Standard markdown link](https://example.com)
`

		result := ConvertWikiLinks(content, nil)

		assert.Equal(t, content, result)
	})

	t.Run("lowercase filenames", func(t *testing.T) {
		content := `[[MyPage]]
`

		result := ConvertWikiLinks(content, nil)

		assert.Contains(t, result, "[MyPage](./mypage.md)")
	})
}

// TestBuildRelativePath tests building relative paths for wiki pages
func TestBuildRelativePath(t *testing.T) {
	t.Run("home page", func(t *testing.T) {
		page := &WikiPage{Filename: "Home.md", IsHome: true, Section: ""}
		structure := &WikiStructure{}

		result := BuildRelativePath(page, structure, false)

		assert.Equal(t, "index.md", result)
	})

	t.Run("flat mode", func(t *testing.T) {
		page := &WikiPage{Filename: "Guide.md", IsHome: false, Section: "Getting Started"}
		structure := &WikiStructure{}

		result := BuildRelativePath(page, structure, true)

		assert.Equal(t, "guide.md", result)
	})

	t.Run("with sections", func(t *testing.T) {
		page := &WikiPage{Filename: "API.md", IsHome: false, Section: "API Reference"}
		structure := &WikiStructure{
			Sections: []WikiSection{{Name: "API Reference"}},
		}

		result := BuildRelativePath(page, structure, false)

		assert.Equal(t, "api-reference/api.md", result)
	})

	t.Run("no sections flat", func(t *testing.T) {
		page := &WikiPage{Filename: "Guide.md", IsHome: false, Section: ""}
		structure := &WikiStructure{Sections: []WikiSection{}}

		result := BuildRelativePath(page, structure, false)

		assert.Equal(t, "guide.md", result)
	})

	t.Run("section with spaces", func(t *testing.T) {
		page := &WikiPage{Filename: "Tutorial.md", IsHome: false, Section: "Getting Started"}
		structure := &WikiStructure{
			Sections: []WikiSection{{Name: "Getting Started"}},
		}

		result := BuildRelativePath(page, structure, false)

		assert.Equal(t, "getting-started/tutorial.md", result)
	})

	t.Run("mixed case filename", func(t *testing.T) {
		page := &WikiPage{Filename: "API_Guide.md", IsHome: false, Section: "API"}
		structure := &WikiStructure{
			Sections: []WikiSection{{Name: "API"}},
		}

		result := BuildRelativePath(page, structure, false)

		assert.Equal(t, "api/api_guide.md", result)
	})
}

// TestWikiParserEdgeCases tests various edge cases
func TestWikiParserEdgeCases(t *testing.T) {
	t.Run("ParseWikiURL with SSH URL", func(t *testing.T) {
		// SSH URLs should also work
		_, err := ParseWikiURL("git@github.com:owner/repo.wiki.git")
		assert.Error(t, err) // Currently only HTTPS URLs are supported
	})

	t.Run("FilenameToTitle with only extension", func(t *testing.T) {
		result := FilenameToTitle(".md")
		assert.Equal(t, "", result)
	})

	t.Run("FilenameToTitle with no extension", func(t *testing.T) {
		result := FilenameToTitle("README")
		assert.Equal(t, "README", result)
	})

	t.Run("TitleToFilename with special chars", func(t *testing.T) {
		result := TitleToFilename("API @#$% Guide")
		assert.Equal(t, "API-@#$%-Guide", result)
	})

	t.Run("ConvertWikiLinks with nested brackets", func(t *testing.T) {
		content := `[[Link]] [[Another]]`
		result := ConvertWikiLinks(content, nil)
		assert.Contains(t, result, "[Link](./link.md)")
		assert.Contains(t, result, "[Another](./another.md)")
	})
}
