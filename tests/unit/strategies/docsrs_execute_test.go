package strategies_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFetcher is a simple mock for testing
type MockFetcher struct {
	responses      map[string]*domain.Response
	errorResponses map[string]error
}

func (m *MockFetcher) Get(ctx context.Context, url string) (*domain.Response, error) {
	if m.errorResponses != nil {
		if err, ok := m.errorResponses[url]; ok {
			return nil, err
		}
	}
	if m.responses != nil {
		if resp, ok := m.responses[url]; ok {
			return resp, nil
		}
	}
	return nil, assert.AnError
}

func (m *MockFetcher) GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*domain.Response, error) {
	return m.Get(ctx, url)
}

func (m *MockFetcher) GetCookies(url string) []*http.Cookie {
	return nil
}

func (m *MockFetcher) Transport() http.RoundTripper {
	return nil
}

func (m *MockFetcher) Close() error {
	return nil
}

// TestDocsRSStrategy_Execute_ValidCrate tests the happy path
func TestDocsRSStrategy_Execute_ValidCrate(t *testing.T) {
	ctx := context.Background()

	jsonData := helpers.LoadFixture(t, "docsrs/json/std_valid.json")

	// Create mock fetcher
	fetcher := &MockFetcher{
		responses: map[string]*domain.Response{
			"https://docs.rs/crate/std/latest/json": {
				StatusCode:  200,
				ContentType: "application/json",
				Body:        jsonData,
			},
		},
	}

	// Create temporary output directory
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/std", strategies.Options{
		Concurrency: 2,
		Limit:       10,
	})
	require.NoError(t, err)
}

// TestDocsRSStrategy_Execute_FetcherError tests error handling when fetcher fails
func TestDocsRSStrategy_Execute_FetcherError(t *testing.T) {
	ctx := context.Background()

	// Create mock fetcher that returns an error
	fetcher := &MockFetcher{
		errorResponses: map[string]error{
			"https://docs.rs/crate/std/latest/json": assert.AnError,
		},
	}

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	// Execute the strategy
	err := strategy.Execute(ctx, "https://docs.rs/std", strategies.Options{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch rustdoc JSON")
}

// TestDocsRSStrategy_Execute_JSONParseError tests error handling for malformed JSON
func TestDocsRSStrategy_Execute_JSONParseError(t *testing.T) {
	ctx := context.Background()

	// Read the malformed JSON fixture
	jsonData := helpers.LoadFixture(t, "docsrs/json/std_malformed.json")

	// Create mock fetcher
	fetcher := &MockFetcher{
		responses: map[string]*domain.Response{
			"https://docs.rs/crate/std/latest/json": {
				StatusCode:  200,
				ContentType: "application/json",
				Body:        jsonData,
			},
		},
	}

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/std", strategies.Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse rustdoc JSON")
}

// TestDocsRSStrategy_Execute_Limit tests the limit option
func TestDocsRSStrategy_Execute_Limit(t *testing.T) {
	ctx := context.Background()

	jsonData := helpers.LoadFixture(t, "docsrs/json/crate_minimal.json")

	fetcher := &MockFetcher{
		responses: map[string]*domain.Response{
			"https://docs.rs/crate/mycrate/latest/json": {
				StatusCode:  200,
				ContentType: "application/json",
				Body:        jsonData,
			},
		},
	}

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/mycrate", strategies.Options{
		Limit: 1,
	})
	require.NoError(t, err)
}

// TestDocsRSStrategy_Execute_DryRun tests the dry-run option
func TestDocsRSStrategy_Execute_DryRun(t *testing.T) {
	ctx := context.Background()

	jsonData := helpers.LoadFixture(t, "docsrs/json/crate_minimal.json")

	fetcher := &MockFetcher{
		responses: map[string]*domain.Response{
			"https://docs.rs/crate/mycrate/latest/json": {
				StatusCode:  200,
				ContentType: "application/json",
				Body:        jsonData,
			},
		},
	}

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
		DryRun:  true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/mycrate", strategies.Options{})
	require.NoError(t, err)
}

// TestDocsRSStrategy_Execute_ContextCancellation tests context cancellation
func TestDocsRSStrategy_Execute_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	jsonData := helpers.LoadFixture(t, "docsrs/json/crate_minimal.json")

	fetcher := &MockFetcher{
		responses: map[string]*domain.Response{
			"https://docs.rs/crate/mycrate/latest/json": {
				StatusCode:  200,
				ContentType: "application/json",
				Body:        jsonData,
			},
		},
	}

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)
	_ = strategy.Execute(ctx, "https://docs.rs/mycrate", strategies.Options{})
}

// TestDocsRSStrategy_Execute_OldFormatVersion tests format version validation
func TestDocsRSStrategy_Execute_OldFormatVersion(t *testing.T) {
	ctx := context.Background()

	jsonData := helpers.LoadFixture(t, "docsrs/json/old_format_version.json")

	fetcher := &MockFetcher{
		responses: map[string]*domain.Response{
			"https://docs.rs/crate/oldcrate/latest/json": {
				StatusCode:  200,
				ContentType: "application/json",
				Body:        jsonData,
			},
		},
	}

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/oldcrate", strategies.Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "format version")
}

// TestDocsRSStrategy_Execute_NilFetcher tests error handling with nil fetcher
func TestDocsRSStrategy_Execute_NilFetcher(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
		// Fetcher is nil
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/std", strategies.Options{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetcher is nil")
}

// TestDocsRSStrategy_Execute_NilWriter tests error handling with nil writer
func TestDocsRSStrategy_Execute_NilWriter(t *testing.T) {
	ctx := context.Background()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	deps := &strategies.Dependencies{
		Fetcher: &MockFetcher{},
		Logger:  logger,
		// Writer is nil
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://docs.rs/std", strategies.Options{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer is nil")
}

// TestDocsRSStrategy_Execute_InvalidURL tests error handling with invalid URL
func TestDocsRSStrategy_Execute_InvalidURL(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: &MockFetcher{},
		Logger:  logger,
		Writer:  writer,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "not-a-valid-url", strategies.Options{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid docs.rs URL")
}

// TestDocsRSStrategy_Execute_NonDocsrsURL tests error handling with non-docs.rs URL
func TestDocsRSStrategy_Execute_NonDocsrsURL(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher: &MockFetcher{},
		Logger:  logger,
		Writer:  writer,
	}

	strategy := strategies.NewDocsRSStrategy(deps)

	err := strategy.Execute(ctx, "https://github.com/rust-lang/rust", strategies.Options{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a docs.rs URL")
}
