package strategies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPkgGoStrategy tests creating a new pkg.go.dev strategy
func TestNewPkgGoStrategy(t *testing.T) {
	deps := &Dependencies{
		Fetcher:   &mockFetcher{},
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewPkgGoStrategy(deps)

	assert.NotNil(t, strategy)
	assert.NotNil(t, strategy.deps)
	assert.NotNil(t, strategy.fetcher)
	assert.NotNil(t, strategy.converter)
	assert.NotNil(t, strategy.writer)
	assert.NotNil(t, strategy.logger)
}

// TestPkgGoStrategy_Name tests the Name method
func TestPkgGoStrategy_Name(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewPkgGoStrategy(deps)

	assert.Equal(t, "pkggo", strategy.Name())
}

// TestPkgGoStrategy_CanHandle tests the CanHandle method
func TestPkgGoStrategy_CanHandle(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewPkgGoStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://pkg.go.dev/github.com/gorilla/mux", true},
		{"https://pkg.go.dev/golang.org/x/text", true},
		{"https://pkg.go.dev/std", true},
		{"http://pkg.go.dev/github.com/gorilla/mux", true},
		{"https://pkggo.dev/github.com/gorilla/mux", false},
		{"https://github.com/gorilla/mux", false},
		{"https://example.com/docs", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPkgGoStrategy_SetFetcher tests the SetFetcher method
func TestPkgGoStrategy_SetFetcher(t *testing.T) {
	deps, _ := NewDependencies(DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		DryRun:      true,
	})
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)
	originalFetcher := strategy.fetcher

	// Create a mock fetcher
	mockFetcher := &mockFetcher{}

	strategy.SetFetcher(mockFetcher)

	assert.NotEqual(t, originalFetcher, strategy.fetcher)
	assert.Equal(t, mockFetcher, strategy.fetcher)
}

// TestPkgGoStrategy_Execute_SinglePage tests basic execution without split
func TestPkgGoStrategy_Execute_SinglePage(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package - pkg.go.dev</title></head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test-package</h1>
	<div class="Documentation-content">
		<h2>Overview</h2>
		<p>This is a test package.</p>
	</div>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  false,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.NoError(t, err)
}

// TestPkgGoStrategy_Execute_SplitMode tests execution with split enabled
func TestPkgGoStrategy_Execute_SplitMode(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package - pkg.go.dev</title></head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test-package</h1>
	<div id="pkg-overview">
		<h2>Overview</h2>
		<p>This is the overview section.</p>
	</div>
	<div id="pkg-index">
		<h2>Index</h2>
		<p>Index content.</p>
	</div>
	<div id="pkg-constants">
		<h2>Constants</h2>
		<p>Constants section.</p>
	</div>
	<div id="pkg-functions">
		<h2>Functions</h2>
		<p>Functions section.</p>
	</div>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  true,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.NoError(t, err)
}

// TestPkgGoStrategy_Execute_ContextCancellation tests context cancellation
func TestPkgGoStrategy_Execute_ContextCancellation(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package</title></head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test-package</h1>
	<div class="Documentation-content">Content</div>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{
		Split:  false,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.Error(t, err)
}

// TestPkgGoStrategy_Execute_WithoutDocumentationContent tests fallback to main
func TestPkgGoStrategy_Execute_WithoutDocumentationContent(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package</title></head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test-package</h1>
	<main>
		<p>Main content area.</p>
	</main>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  false,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.NoError(t, err)
}

// TestPkgGoStrategy_Execute_WithEmptySection tests skipping empty sections
func TestPkgGoStrategy_Execute_WithEmptySection(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package</title></head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test-package</h1>
	<div id="pkg-overview">
		<p>Overview content</p>
	</div>
	<div id="pkg-constants">
	</div>
	<div id="pkg-functions">
		<p>Functions content</p>
	</div>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  true,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.NoError(t, err)
}

// TestPkgGoStrategy_Execute_FetchError tests error handling on fetch failure
func TestPkgGoStrategy_Execute_FetchError(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  false,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.Error(t, err)
}

// TestPkgGoStrategy_Execute_SplitWithSectionSkip tests section extraction with skip
func TestPkgGoStrategy_Execute_SplitWithSectionSkip(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package</title></head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test-package</h1>
	<div id="pkg-overview">
		<p>Overview</p>
	</div>
	<div id="pkg-types">
		<p>Types section</p>
	</div>
	<div id="pkg-variables">
		<p>Variables</p>
	</div>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  true,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.NoError(t, err)
}

// TestPkgGoStrategy_Execute_MissingPackageTitle tests with missing package name
func TestPkgGoStrategy_Execute_MissingPackageTitle(t *testing.T) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>test/package</title></head>
<body>
	<div class="Documentation-content">
		<p>Content without title</p>
	</div>
</body>
</html>
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Split:  false,
		DryRun: true,
	}

	err = strategy.Execute(ctx, server.URL+"/github.com/example/test-package", opts)
	assert.NoError(t, err)
}
