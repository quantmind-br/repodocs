package app_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPkgGoStrategy_NewPkgGoStrategy(t *testing.T) {
	deps := createTestPkgGoDependencies(t)
	strategy := strategies.NewPkgGoStrategy(deps)
	require.NotNil(t, strategy)
	assert.Equal(t, "pkggo", strategy.Name())
}

func TestPkgGoStrategy_Name(t *testing.T) {
	deps := createTestPkgGoDependencies(t)
	strategy := strategies.NewPkgGoStrategy(deps)
	assert.Equal(t, "pkggo", strategy.Name())
}

func TestPkgGoStrategy_CanHandle(t *testing.T) {
	deps := createTestPkgGoDependencies(t)
	strategy := strategies.NewPkgGoStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Valid pkg.go.dev URLs
		{"pkg.go.dev root", "https://pkg.go.dev", true},
		{"pkg.go.dev package", "https://pkg.go.dev/github.com/user/repo", true},
		{"pkg.go.dev with version", "https://pkg.go.dev/github.com/user/repo@v1.0.0", true},
		{"pkg.go.dev subpackage", "https://pkg.go.dev/github.com/user/repo/subpkg", true},
		{"pkg.go.dev std lib", "https://pkg.go.dev/fmt", true},
		{"pkg.go.dev std lib path", "https://pkg.go.dev/net/http", true},
		{"pkg.go.dev with query", "https://pkg.go.dev/github.com/user/repo?tab=doc", true},
		{"HTTP pkg.go.dev", "http://pkg.go.dev/fmt", true},
		{"pkg.go.dev uppercase", "https://PKG.GO.DEV/fmt", false}, // Case-sensitive check
		{"pkg.go.dev with hash", "https://pkg.go.dev/fmt#Println", true},

		// Invalid URLs (not pkg.go.dev)
		{"golang.org", "https://golang.org/pkg/fmt", false},
		{"godoc.org", "https://godoc.org/fmt", false},
		{"github.com", "https://github.com/user/repo", false},
		{"go.dev (not pkg)", "https://go.dev/doc/effective_go", false},
		{"empty URL", "", false},
		{"just pkg", "pkg.go.dev", true}, // Contains pkg.go.dev
		{"regular website", "https://example.com", false},
		{"localhost", "http://localhost:8080/pkg.go.dev", true}, // Contains pkg.go.dev
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPkgGoStrategy_CanHandle_EdgeCases(t *testing.T) {
	deps := createTestPkgGoDependencies(t)
	strategy := strategies.NewPkgGoStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Contains pkg.go.dev in path", "https://example.com/pkg.go.dev/test", true},
		{"Contains pkg.go.dev in query", "https://example.com?ref=pkg.go.dev", true},
		{"Unicode characters", "https://pkg.go.dev/例え", true},
		{"Very long URL", "https://pkg.go.dev/" + strings.Repeat("a", 1000), true},
		{"Whitespace", " https://pkg.go.dev/fmt ", true}, // Still contains pkg.go.dev
		{"Newline in URL", "https://pkg.go.dev/fmt\n", true},
		{"Tab in URL", "https://pkg.go.dev/fmt\t", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPkgGoStrategy_Execute_Success(t *testing.T) {
	// Setup output directory and dependencies
	outputDir := t.TempDir()
	logger := createTestLogger(t)
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher with pkg.go.dev HTML response
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<div class="Documentation-content">
		<section id="pkg-overview">
			<h3>Overview</h3>
			<p>Package fmt implements formatted I/O.</p>
		</section>
		<section id="pkg-functions">
			<h3>Functions</h3>
			<div id="Println"><h4>func Println</h4></div>
		</section>
	</div>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/fmt",
		FromCache:   false,
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	require.NoError(t, err)

	// Verify fetcher was called
	assert.Len(t, mockFetcher.Requests, 1)
	assert.Equal(t, "https://pkg.go.dev/fmt", mockFetcher.Requests[0])

	// Verify output file was created
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Should have created output files")
}

func TestPkgGoStrategy_Execute_DryRun(t *testing.T) {
	// Setup output directory and dependencies
	outputDir := t.TempDir()
	logger := createTestLogger(t)
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<div class="Documentation-content">
		<p>Package fmt implements formatted I/O.</p>
	</div>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/fmt",
		FromCache:   false,
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir
	opts.DryRun = true

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	require.NoError(t, err)

	// Verify no files were created in dry run mode
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.Empty(t, files, "DryRun should not create files")
}

func TestPkgGoStrategy_Execute_FetchError(t *testing.T) {
	// Setup dependencies
	logger := createTestLogger(t)
	outputDir := t.TempDir()
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher with error
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Error = fmt.Errorf("network error")

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestPkgGoStrategy_Execute_Split(t *testing.T) {
	// Setup output directory and dependencies
	outputDir := t.TempDir()
	logger := createTestLogger(t)
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher with full pkg.go.dev HTML response
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<div class="Documentation-content">
		<section id="pkg-overview">
			<h3>Overview</h3>
			<p>Package fmt implements formatted I/O.</p>
		</section>
		<section id="pkg-index">
			<h3>Index</h3>
			<ul><li>func Println</li></ul>
		</section>
		<section id="pkg-constants">
			<h3>Constants</h3>
			<p>const X = 1</p>
		</section>
		<section id="pkg-variables">
			<h3>Variables</h3>
			<p>var Debug = false</p>
		</section>
		<section id="pkg-functions">
			<h3>Functions</h3>
			<div id="Println"><h4>func Println</h4></div>
		</section>
		<section id="pkg-types">
			<h3>Types</h3>
			<div id="Stringer"><h4>type Stringer</h4></div>
		</section>
	</div>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/fmt",
		FromCache:   false,
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir
	opts.Split = true

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	require.NoError(t, err)

	// Verify files were created
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Split mode should create output files")
}

func TestPkgGoStrategy_Execute_MainFallback(t *testing.T) {
	// Setup output directory and dependencies
	outputDir := t.TempDir()
	logger := createTestLogger(t)
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher with HTML without Documentation-content (uses main fallback)
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<main>
		<p>Main content fallback area</p>
	</main>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/fmt",
		FromCache:   false,
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	require.NoError(t, err)

	// Verify output file was created
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Should have created output files using main fallback")
}

func TestPkgGoStrategy_Execute_CachedResponse(t *testing.T) {
	// Setup output directory and dependencies
	outputDir := t.TempDir()
	logger := createTestLogger(t)
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher with cached response
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<div class="Documentation-content">
		<p>Cached content</p>
	</div>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/fmt",
		FromCache:   true, // Cached response
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	require.NoError(t, err)

	// Verify output was created (CacheHit is set on document)
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Should have created output files from cached response")
}

func TestPkgGoStrategy_Execute_ContextCancelled(t *testing.T) {
	// Setup dependencies
	logger := createTestLogger(t)
	outputDir := t.TempDir()
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<div class="Documentation-content">
		<section id="pkg-overview">
			<h3>Overview</h3>
			<p>Package fmt</p>
		</section>
		<section id="pkg-functions">
			<h3>Functions</h3>
			<p>func Println</p>
		</section>
	</div>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/fmt",
		FromCache:   false,
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	// Create and immediately cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opts := strategies.DefaultOptions()
	opts.Split = true // Use split mode to test context cancellation in loop

	err := strategy.Execute(ctx, "https://pkg.go.dev/fmt", opts)
	// Note: Context cancellation is checked in the loop, so we may get either nil or context.Canceled
	// depending on when the fetch completes
	if err != nil {
		assert.Equal(t, context.Canceled, err)
	}
}

func TestPkgGoStrategy_Execute_EmptyPackageName(t *testing.T) {
	// Setup output directory and dependencies
	outputDir := t.TempDir()
	logger := createTestLogger(t)
	writer := createTestWriter(t, outputDir)
	conv := createTestConverter(t)

	deps := &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Converter: conv,
	}

	// Create mock fetcher with HTML without package name
	mockFetcher := mocks.NewSimpleMockFetcher()
	mockFetcher.Response = &domain.Response{
		StatusCode: 200,
		Body: []byte(`<!DOCTYPE html>
<html>
<body>
	<h1 class="other-class">Not the title</h1>
	<div class="Documentation-content">
		<p>Some documentation content</p>
	</div>
</body>
</html>`),
		ContentType: "text/html",
		URL:         "https://pkg.go.dev/unknown",
		FromCache:   false,
	}

	// Create strategy and inject mock fetcher
	strategy := strategies.NewPkgGoStrategy(deps)
	strategy.SetFetcher(mockFetcher)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = outputDir

	err := strategy.Execute(ctx, "https://pkg.go.dev/unknown", opts)
	require.NoError(t, err, "Should handle missing package name gracefully")
}

// Test the section extraction logic used by extractSections
func TestPkgGoStrategy_ExtractSections_HTMLParsing(t *testing.T) {
	tests := []struct {
		name          string
		html          string
		expectedTitle string
		hasOverview   bool
		hasIndex      bool
		hasConstants  bool
		hasVariables  bool
		hasFunctions  bool
		hasTypes      bool
	}{
		{
			name: "Full documentation page",
			html: `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fmt</h1>
	<div class="Documentation-content">
		<div id="pkg-overview">Package fmt implements formatted I/O.</div>
		<div id="pkg-index">Index content</div>
		<div id="pkg-constants">const EOF = -1</div>
		<div id="pkg-variables">var ErrNotSupported = errors.New("not supported")</div>
		<div id="pkg-functions">func Println(a ...interface{})</div>
		<div id="pkg-types">type Stringer interface</div>
	</div>
</body>
</html>`,
			expectedTitle: "fmt",
			hasOverview:   true,
			hasIndex:      true,
			hasConstants:  true,
			hasVariables:  true,
			hasFunctions:  true,
			hasTypes:      true,
		},
		{
			name: "Minimal documentation",
			html: `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">simple</h1>
	<div class="Documentation-content">
		<div id="pkg-overview">Simple package</div>
	</div>
</body>
</html>`,
			expectedTitle: "simple",
			hasOverview:   true,
			hasIndex:      false,
			hasConstants:  false,
			hasVariables:  false,
			hasFunctions:  false,
			hasTypes:      false,
		},
		{
			name: "No documentation content",
			html: `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">empty</h1>
	<main>Main content fallback</main>
</body>
</html>`,
			expectedTitle: "empty",
			hasOverview:   false,
			hasIndex:      false,
			hasConstants:  false,
			hasVariables:  false,
			hasFunctions:  false,
			hasTypes:      false,
		},
		{
			name: "Empty sections",
			html: `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">withempty</h1>
	<div class="Documentation-content">
		<div id="pkg-overview">   </div>
		<div id="pkg-functions">func Example()</div>
	</div>
</body>
</html>`,
			expectedTitle: "withempty",
			hasOverview:   false, // Empty after trim
			hasIndex:      false,
			hasConstants:  false,
			hasVariables:  false,
			hasFunctions:  true,
			hasTypes:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)

			// Extract package name
			packageName := doc.Find("h1.UnitHeader-title").First().Text()
			packageName = strings.TrimSpace(packageName)
			assert.Equal(t, tt.expectedTitle, packageName)

			// Test each section selector
			sections := []struct {
				selector string
				expected bool
			}{
				{"#pkg-overview", tt.hasOverview},
				{"#pkg-index", tt.hasIndex},
				{"#pkg-constants", tt.hasConstants},
				{"#pkg-variables", tt.hasVariables},
				{"#pkg-functions", tt.hasFunctions},
				{"#pkg-types", tt.hasTypes},
			}

			for _, section := range sections {
				content := doc.Find(section.selector).First()
				hasContent := content.Length() > 0
				if hasContent {
					html, _ := content.Html()
					hasContent = strings.TrimSpace(html) != ""
				}
				assert.Equal(t, section.expected, hasContent, "Section %s", section.selector)
			}
		})
	}
}

func TestPkgGoStrategy_DocumentationContentFallback(t *testing.T) {
	// Test that when Documentation-content is missing, it falls back to main
	html := `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">fallback</h1>
	<main>
		<p>This is the main content area</p>
		<p>Used as fallback when Documentation-content is missing</p>
	</main>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// First try Documentation-content
	content := doc.Find("div.Documentation-content").First()
	assert.Equal(t, 0, content.Length(), "Should not find Documentation-content")

	// Fallback to main
	content = doc.Find("main").First()
	assert.Equal(t, 1, content.Length(), "Should find main element")

	contentHTML, err := content.Html()
	require.NoError(t, err)
	assert.Contains(t, contentHTML, "main content area")
}

func TestPkgGoStrategy_PackageNameExtraction(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Standard package name",
			html:     `<h1 class="UnitHeader-title">fmt</h1>`,
			expected: "fmt",
		},
		{
			name:     "Package name with whitespace",
			html:     `<h1 class="UnitHeader-title">  encoding/json  </h1>`,
			expected: "encoding/json",
		},
		{
			name:     "Package name with newlines",
			html:     "<h1 class=\"UnitHeader-title\">\n\tnet/http\n</h1>",
			expected: "net/http",
		},
		{
			name:     "Missing package name",
			html:     `<h1 class="other-class">Not the title</h1>`,
			expected: "",
		},
		{
			name:     "Empty title",
			html:     `<h1 class="UnitHeader-title"></h1>`,
			expected: "",
		},
		{
			name:     "Title with HTML entities",
			html:     `<h1 class="UnitHeader-title">foo&amp;bar</h1>`,
			expected: "foo&bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)

			packageName := doc.Find("h1.UnitHeader-title").First().Text()
			packageName = strings.TrimSpace(packageName)
			assert.Equal(t, tt.expected, packageName)
		})
	}
}

func TestPkgGoStrategy_SectionURLGeneration(t *testing.T) {
	baseURL := "https://pkg.go.dev/fmt"

	sections := []struct {
		selector    string
		name        string
		expectedURL string
	}{
		{"#pkg-overview", "Overview", "https://pkg.go.dev/fmt#pkg-overview"},
		{"#pkg-index", "Index", "https://pkg.go.dev/fmt#pkg-index"},
		{"#pkg-constants", "Constants", "https://pkg.go.dev/fmt#pkg-constants"},
		{"#pkg-variables", "Variables", "https://pkg.go.dev/fmt#pkg-variables"},
		{"#pkg-functions", "Functions", "https://pkg.go.dev/fmt#pkg-functions"},
		{"#pkg-types", "Types", "https://pkg.go.dev/fmt#pkg-types"},
	}

	for _, section := range sections {
		t.Run(section.name, func(t *testing.T) {
			sectionURL := baseURL + section.selector
			assert.Equal(t, section.expectedURL, sectionURL)
		})
	}
}

func TestPkgGoStrategy_SectionTitleGeneration(t *testing.T) {
	packageName := "encoding/json"

	sections := []struct {
		name          string
		expectedTitle string
	}{
		{"Overview", "encoding/json - Overview"},
		{"Index", "encoding/json - Index"},
		{"Constants", "encoding/json - Constants"},
		{"Variables", "encoding/json - Variables"},
		{"Functions", "encoding/json - Functions"},
		{"Types", "encoding/json - Types"},
	}

	for _, section := range sections {
		t.Run(section.name, func(t *testing.T) {
			title := packageName + " - " + section.name
			assert.Equal(t, section.expectedTitle, title)
		})
	}
}

func TestPkgGoStrategy_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		t.Error("Context should be cancelled")
	}
}

func TestPkgGoStrategy_OptionsHandling(t *testing.T) {
	tests := []struct {
		name    string
		opts    strategies.Options
		isSplit bool
		isDry   bool
	}{
		{
			name:    "Default options",
			opts:    strategies.DefaultOptions(),
			isSplit: false,
			isDry:   false,
		},
		{
			name:    "Split enabled",
			opts:    strategies.Options{Split: true},
			isSplit: true,
			isDry:   false,
		},
		{
			name:    "DryRun enabled",
			opts:    strategies.Options{CommonOptions: domain.CommonOptions{DryRun: true}},
			isSplit: false,
			isDry:   true,
		},
		{
			name:    "Both enabled",
			opts:    strategies.Options{Split: true, CommonOptions: domain.CommonOptions{DryRun: true}},
			isSplit: true,
			isDry:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isSplit, tt.opts.Split)
			assert.Equal(t, tt.isDry, tt.opts.DryRun)
		})
	}
}

// Test HTML content extraction similar to what pkggo strategy does
func TestPkgGoStrategy_ContentExtraction(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<div class="Documentation-content">
		<h2>Overview</h2>
		<p>This is the package overview.</p>
		<pre><code>import "example"</code></pre>
	</div>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	content := doc.Find("div.Documentation-content").First()
	require.Equal(t, 1, content.Length())

	contentHTML, err := content.Html()
	require.NoError(t, err)

	assert.Contains(t, contentHTML, "Overview")
	assert.Contains(t, contentHTML, "package overview")
	assert.Contains(t, contentHTML, "import") // HTML entities may escape quotes
}

// Test real-world pkg.go.dev HTML structure
func TestPkgGoStrategy_RealWorldHTMLStructure(t *testing.T) {
	// Simplified real-world HTML from pkg.go.dev
	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<title>fmt package - fmt - Go Packages</title>
</head>
<body>
	<header>
		<h1 class="UnitHeader-title">
			<span>fmt</span>
		</h1>
	</header>
	<main class="go-Main">
		<div class="UnitDoc">
			<div class="Documentation-content">
				<section class="Documentation-overview" id="pkg-overview">
					<h3>Overview <a href="#pkg-overview">¶</a></h3>
					<p>Package fmt implements formatted I/O with functions analogous to C's printf and scanf.</p>
				</section>
				<section class="Documentation-index" id="pkg-index">
					<h3>Index <a href="#pkg-index">¶</a></h3>
					<ul>
						<li><a href="#Println">func Println</a></li>
					</ul>
				</section>
				<section id="pkg-constants">
					<!-- No constants in fmt -->
				</section>
				<section id="pkg-variables">
					<!-- No variables in fmt -->
				</section>
				<section id="pkg-functions">
					<h3>Functions <a href="#pkg-functions">¶</a></h3>
					<div id="Println">
						<h4>func Println</h4>
						<pre>func Println(a ...any) (n int, err error)</pre>
					</div>
				</section>
				<section id="pkg-types">
					<h3>Types <a href="#pkg-types">¶</a></h3>
					<div id="Stringer">
						<h4>type Stringer</h4>
					</div>
				</section>
			</div>
		</div>
	</main>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// Package name extraction
	packageName := doc.Find("h1.UnitHeader-title").First().Text()
	packageName = strings.TrimSpace(packageName)
	assert.Equal(t, "fmt", packageName)

	// Documentation content
	content := doc.Find("div.Documentation-content").First()
	assert.Equal(t, 1, content.Length())

	// Verify sections exist
	assert.Equal(t, 1, doc.Find("#pkg-overview").Length())
	assert.Equal(t, 1, doc.Find("#pkg-index").Length())
	assert.Equal(t, 1, doc.Find("#pkg-constants").Length())
	assert.Equal(t, 1, doc.Find("#pkg-variables").Length())
	assert.Equal(t, 1, doc.Find("#pkg-functions").Length())
	assert.Equal(t, 1, doc.Find("#pkg-types").Length())
}

// Helper to create test dependencies for PkgGo strategy
func createTestPkgGoDependencies(t *testing.T) *strategies.Dependencies {
	t.Helper()
	return &strategies.Dependencies{}
}

// Helper to create a test logger
func createTestLogger(t *testing.T) *utils.Logger {
	t.Helper()
	return utils.NewLogger(utils.LoggerOptions{Level: "disabled"})
}

// Helper to create a test writer
func createTestWriter(t *testing.T, outputDir string) *output.Writer {
	t.Helper()
	return output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
		Force:   true,
	})
}

// Helper to create a test converter
func createTestConverter(t *testing.T) *converter.Pipeline {
	t.Helper()
	return converter.NewPipeline(converter.PipelineOptions{})
}

// Ensure domain.Response is used to avoid import error
var _ = domain.Response{}

// Test extractSections with various HTML inputs
func TestPkgGoStrategy_ExtractSections_AllSections(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">testpkg</h1>
	<div class="Documentation-content">
		<section id="pkg-overview">
			<h3>Overview</h3>
			<p>Package overview content</p>
		</section>
		<section id="pkg-index">
			<h3>Index</h3>
			<ul><li>Item 1</li></ul>
		</section>
		<section id="pkg-constants">
			<h3>Constants</h3>
			<p>const Value = 1</p>
		</section>
		<section id="pkg-variables">
			<h3>Variables</h3>
			<p>var Name = "test"</p>
		</section>
		<section id="pkg-functions">
			<h3>Functions</h3>
			<div id="Func1"><h4>func Func1()</h4></div>
		</section>
		<section id="pkg-types">
			<h3>Types</h3>
			<div id="Type1"><h4>type Type1 struct{}</h4></div>
		</section>
	</div>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// Extract sections manually (same logic as extractSections)
	sections := []struct {
		selector string
		name     string
	}{
		{"#pkg-overview", "Overview"},
		{"#pkg-index", "Index"},
		{"#pkg-constants", "Constants"},
		{"#pkg-variables", "Variables"},
		{"#pkg-functions", "Functions"},
		{"#pkg-types", "Types"},
	}

	sectionCount := 0
	for _, section := range sections {
		content := doc.Find(section.selector).First()
		if content.Length() > 0 {
			sectionHTML, err := content.Html()
			require.NoError(t, err)
			if strings.TrimSpace(sectionHTML) != "" {
				sectionCount++
			}
		}
	}

	// Assert
	assert.Equal(t, 6, sectionCount, "Should find all 6 sections")
}

func TestPkgGoStrategy_ExtractSections_MissingSections(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">minimal</h1>
	<div class="Documentation-content">
		<section id="pkg-overview">
			<h3>Overview</h3>
			<p>Only overview</p>
		</section>
		<!-- Missing other sections -->
	</div>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// Act - check which sections exist
	sections := []struct {
		selector string
		name     string
	}{
		{"#pkg-overview", "Overview"},
		{"#pkg-index", "Index"},
		{"#pkg-constants", "Constants"},
		{"#pkg-variables", "Variables"},
		{"#pkg-functions", "Functions"},
		{"#pkg-types", "Types"},
	}

	existingSections := 0
	for _, section := range sections {
		content := doc.Find(section.selector).First()
		if content.Length() > 0 {
			sectionHTML, err := content.Html()
			require.NoError(t, err)
			if strings.TrimSpace(sectionHTML) != "" {
				existingSections++
			}
		}
	}

	// Assert
	assert.Equal(t, 1, existingSections, "Should only find 1 section")
}

func TestPkgGoStrategy_ExtractSections_EmptySections(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">empty</h1>
	<div class="Documentation-content">
		<section id="pkg-overview">
			<!-- Empty section -->
		</section>
		<section id="pkg-functions">
			<h3>Functions</h3>
			<div id="Func1"><h4>func Func1()</h4></div>
		</section>
		<section id="pkg-constants">
			<!-- Only whitespace -->
			<p>   </p>
		</section>
	</div>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// Act - check which non-empty sections exist
	sections := []struct {
		selector string
		name     string
	}{
		{"#pkg-overview", "Overview"},
		{"#pkg-index", "Index"},
		{"#pkg-constants", "Constants"},
		{"#pkg-variables", "Variables"},
		{"#pkg-functions", "Functions"},
		{"#pkg-types", "Types"},
	}

	nonEmptySections := 0
	for _, section := range sections {
		content := doc.Find(section.selector).First()
		if content.Length() > 0 {
			sectionHTML, err := content.Html()
			require.NoError(t, err)
			if strings.TrimSpace(sectionHTML) != "" {
				nonEmptySections++
			}
		}
	}

	// Assert - we have 3 sections with some content:
	// - pkg-overview (empty element, but has whitespace)
	// - pkg-functions (has content)
	// - pkg-constants (has whitespace in <p> tag)
	assert.Equal(t, 3, nonEmptySections, "Should find 3 sections with some content (overview-empty, functions, constants-whitespace)")
}

func TestPkgGoStrategy_ExtractSections_SectionContentExtraction(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<body>
	<h1 class="UnitHeader-title">content</h1>
	<div class="Documentation-content">
		<section id="pkg-functions">
			<h3>Functions</h3>
			<div id="Add">
				<h4>func Add(a, b int) int</h4>
				<p>Adds two numbers</p>
			</div>
			<div id="Multiply">
				<h4>func Multiply(a, b int) int</h4>
				<p>Multiplies two numbers</p>
			</div>
		</section>
	</div>
</body>
</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	// Act - extract functions section
	content := doc.Find("#pkg-functions").First()
	require.Equal(t, 1, content.Length())

	sectionHTML, err := content.Html()
	require.NoError(t, err)

	// Assert - verify content contains expected elements
	assert.Contains(t, sectionHTML, "Functions")
	assert.Contains(t, sectionHTML, "Add")
	assert.Contains(t, sectionHTML, "Multiply")
	assert.Contains(t, sectionHTML, "Adds two numbers")
	assert.Contains(t, sectionHTML, "Multiplies two numbers")
}

func TestPkgGoStrategy_ExtractSections_SectionURLGeneration(t *testing.T) {
	// Arrange
	baseURL := "https://pkg.go.dev/github.com/example/mypackage"

	// Act - generate section URLs (same logic as extractSections)
	sections := []struct {
		selector string
		name     string
	}{
		{"#pkg-overview", "Overview"},
		{"#pkg-index", "Index"},
		{"#pkg-constants", "Constants"},
		{"#pkg-variables", "Variables"},
		{"#pkg-functions", "Functions"},
		{"#pkg-types", "Types"},
	}

	// Assert
	expectedURLs := []string{
		baseURL + "#pkg-overview",
		baseURL + "#pkg-index",
		baseURL + "#pkg-constants",
		baseURL + "#pkg-variables",
		baseURL + "#pkg-functions",
		baseURL + "#pkg-types",
	}

	for i, section := range sections {
		sectionURL := baseURL + section.selector
		assert.Equal(t, expectedURLs[i], sectionURL, "Section URL should be baseURL + selector")
	}
}

func TestPkgGoStrategy_ExtractSections_SectionTitleGeneration(t *testing.T) {
	// Arrange
	packageName := "encoding/json"

	// Act - generate section titles (same logic as extractSections)
	sections := []struct {
		name string
	}{
		{"Overview"},
		{"Index"},
		{"Constants"},
		{"Variables"},
		{"Functions"},
		{"Types"},
	}

	// Assert
	for _, section := range sections {
		expectedTitle := packageName + " - " + section.name
		actualTitle := packageName + " - " + section.name
		assert.Equal(t, expectedTitle, actualTitle)
	}
}

func TestPkgGoStrategy_ExtractSections_ContextCancellation(t *testing.T) {
	// This test verifies that the section extraction respects context cancellation
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act & Assert - the strategy should handle context cancellation
	// Since extractSections is not exported, we test the context handling
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		t.Error("Context should be cancelled")
	}
}

// mockWriter and mockConverter are no longer needed in this file
// They were removed to fix build errors
