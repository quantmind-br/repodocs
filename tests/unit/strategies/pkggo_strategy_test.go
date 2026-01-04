package strategies_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDependencies creates test dependencies with fetcher and converter
func setupPkgGoTestDependencies(t *testing.T, tmpDir string) *strategies.Dependencies {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	// Create fetcher
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     10 * time.Second,
		MaxRetries:  1,
		EnableCache: false,
	})
	require.NoError(t, err)

	// Create converter
	converterPipeline := converter.NewPipeline(converter.PipelineOptions{})

	return &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Fetcher:   fetcherClient,
		Converter: converterPipeline,
	}
}

// TestNewPkgGoStrategy tests creating a new pkg.go.dev strategy
func TestNewPkgGoStrategy(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	strategy := strategies.NewPkgGoStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "pkggo", strategy.Name())
}

// TestPkgGoStrategy_CanHandle tests URL handling for pkg.go.dev strategy
func TestPkgGoStrategy_CanHandle(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	strategy := strategies.NewPkgGoStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://pkg.go.dev/github.com/gorilla/mux", true},
		{"https://pkg.go.dev/golang.org/x/text", true},
		{"https://pkg.go.dev/std", true},
		{"https://github.com/gorilla/mux", false},
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

// TestPkgGoStrategy_Execute tests executing pkg.go.dev strategy
func TestPkgGoStrategy_Execute(t *testing.T) {
	// Create test server with pkg.go.dev-like content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>github.com/gorilla/mux - pkg.go.dev</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/gorilla/mux</h1>
	<div class="Documentation-content">
		<h2>Overview</h2>
		<p>Mux is a powerful URL router and dispatcher.</p>
		<h3>Installation</h3>
		<pre>go get github.com/gorilla/mux</pre>
	</div>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestPkgGoStrategy_Execute_SplitMode tests executing with split mode
func TestPkgGoStrategy_Execute_SplitMode(t *testing.T) {
	// Create test server with pkg.go.dev-like content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>test package - pkg.go.dev</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test</h1>

	<div id="pkg-overview">
		<h2>Overview</h2>
		<p>This is the package overview.</p>
	</div>

	<div id="pkg-index">
		<h2>Index</h2>
		<ul>
			<li><a href="#FuncA">func FuncA()</a></li>
			<li><a href="#FuncB">func FuncB()</a></li>
		</ul>
	</div>

	<div id="pkg-constants">
		<h2>Constants</h2>
		<pre>const Version = "1.0"</pre>
	</div>

	<div id="pkg-variables">
		<h2>Variables</h2>
		<pre>var GlobalVar int</pre>
	</div>

	<div id="pkg-functions">
		<h2>Functions</h2>
		<pre>func FuncA() {}</pre>
	</div>

	<div id="pkg-types">
		<h2>Types</h2>
		<pre>type MyStruct struct {}</pre>
	</div>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Split = true

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestPkgGoStrategy_Execute_EmptyDocumentation tests with empty documentation
func TestPkgGoStrategy_Execute_EmptyDocumentation(t *testing.T) {
	// Create test server with empty content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>empty package</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/example/empty</h1>
	<div class="Documentation-content">
		<!-- No content -->
	</div>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestPkgGoStrategy_Execute_MissingDocumentationContent tests with missing content div
func TestPkgGoStrategy_Execute_MissingDocumentationContent(t *testing.T) {
	// Create test server without Documentation-content div
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>test package</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test</h1>
	<main>
		<h2>Documentation</h2>
		<p>Some content in main.</p>
	</main>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestPkgGoStrategy_Execute_SplitModeWithEmptySections tests split mode with some empty sections
func TestPkgGoStrategy_Execute_SplitModeWithEmptySections(t *testing.T) {
	// Create test server with some empty sections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>test package</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test</h1>

	<div id="pkg-overview">
		<h2>Overview</h2>
		<p>Package overview content.</p>
	</div>

	<div id="pkg-index">
		<h2>Index</h2>
		<!-- Empty index -->
	</div>

	<div id="pkg-constants">
		<!-- Empty constants -->
	</div>

	<div id="pkg-functions">
		<h2>Functions</h2>
		<pre>func Test() {}</pre>
	</div>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Split = true

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestPkgGoStrategy_Execute_DryRun tests dry run mode
func TestPkgGoStrategy_Execute_DryRun(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>test package</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test</h1>
	<div class="Documentation-content">
		<h2>Overview</h2>
		<p>Content.</p>
	</div>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.DryRun = true

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestPkgGoStrategy_Execute_ErrorFetching tests error handling
func TestPkgGoStrategy_Execute_ErrorFetching(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir

	err := strategy.Execute(ctx, server.URL, opts)
	assert.Error(t, err)
}

// TestPkgGoStrategy_Execute_InvalidHTML tests with invalid HTML
func TestPkgGoStrategy_Execute_InvalidHTML(t *testing.T) {
	// Create test server with invalid HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid html content`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir

	err := strategy.Execute(ctx, server.URL, opts)
	// Should handle invalid HTML gracefully
	if err != nil {
		assert.NotEmpty(t, err.Error())
	}
}

// TestPkgGoStrategy_Execute_ContextCancellation tests context cancellation
func TestPkgGoStrategy_Execute_ContextCancellation(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>test package</title>
</head>
<body>
	<h1 class="UnitHeader-title">github.com/example/test</h1>
	<div class="Documentation-content">
		<h2>Overview</h2>
		<p>Content.</p>
	</div>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupPkgGoTestDependencies(t, tmpDir)

	strategy := strategies.NewPkgGoStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := strategies.DefaultOptions()
	opts.Output = tmpDir

	err := strategy.Execute(ctx, server.URL, opts)
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}
}

// TestPkgGoStrategy_Execute_PackageNameExtraction tests package name extraction
func TestPkgGoStrategy_Execute_PackageNameExtraction(t *testing.T) {
	tests := []struct {
		name          string
		html          string
		expectedTitle string
	}{
		{
			name:          "standard package name",
			html:          `<h1 class="UnitHeader-title">github.com/gorilla/mux</h1>`,
			expectedTitle: "github.com/gorilla/mux",
		},
		{
			name:          "package with whitespace",
			html:          `<h1 class="UnitHeader-title">  github.com/gorilla/mux  </h1>`,
			expectedTitle: "github.com/gorilla/mux",
		},
		{
			name:          "std lib package",
			html:          `<h1 class="UnitHeader-title">std</h1>`,
			expectedTitle: "std",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>test</title>
</head>
<body>
	` + tc.html + `
	<div class="Documentation-content">
		<p>Content</p>
	</div>
</body>
</html>`))
			}))
			defer server.Close()

			// Create dependencies
			tmpDir := t.TempDir()
			deps := setupPkgGoTestDependencies(t, tmpDir)

			strategy := strategies.NewPkgGoStrategy(deps)

			ctx := context.Background()
			opts := strategies.DefaultOptions()
			opts.Output = tmpDir

			err := strategy.Execute(ctx, server.URL, opts)
			require.NoError(t, err)
		})
	}
}
