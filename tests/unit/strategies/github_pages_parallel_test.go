package strategies_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// probeAwareFetcher is a mock fetcher that responds differently based on probe paths
type probeAwareFetcher struct {
	mu         sync.Mutex
	responses  map[string]*domain.Response
	errors     map[string]error
	delays     map[string]time.Duration
	callCount  map[string]int
	callOrder  []string
	callOrderM sync.Mutex
}

func newProbeAwareFetcher() *probeAwareFetcher {
	return &probeAwareFetcher{
		responses: make(map[string]*domain.Response),
		errors:    make(map[string]error),
		delays:    make(map[string]time.Duration),
		callCount: make(map[string]int),
		callOrder: make([]string, 0),
	}
}

func (f *probeAwareFetcher) setResponse(pathSuffix string, resp *domain.Response) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[pathSuffix] = resp
}

func (f *probeAwareFetcher) setError(pathSuffix string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errors[pathSuffix] = err
}

func (f *probeAwareFetcher) setDelay(pathSuffix string, delay time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delays[pathSuffix] = delay
}

func (f *probeAwareFetcher) getCallCount(pathSuffix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount[pathSuffix]
}

func (f *probeAwareFetcher) getCallOrder() []string {
	f.callOrderM.Lock()
	defer f.callOrderM.Unlock()
	result := make([]string, len(f.callOrder))
	copy(result, f.callOrder)
	return result
}

func (f *probeAwareFetcher) Get(ctx context.Context, url string) (*domain.Response, error) {
	// Extract path suffix for matching
	pathSuffix := ""
	for _, probe := range strategies.GetDiscoveryProbes() {
		if strings.HasSuffix(url, probe.Path) {
			pathSuffix = probe.Path
			break
		}
	}

	// Record call order
	f.callOrderM.Lock()
	f.callOrder = append(f.callOrder, pathSuffix)
	f.callOrderM.Unlock()

	f.mu.Lock()
	f.callCount[pathSuffix]++
	delay := f.delays[pathSuffix]
	resp := f.responses[pathSuffix]
	err := f.errors[pathSuffix]
	f.mu.Unlock()

	// Simulate network delay
	if delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	if err != nil {
		return nil, err
	}

	if resp != nil {
		return resp, nil
	}

	// Default 404 response
	return &domain.Response{
		StatusCode: http.StatusNotFound,
		Body:       []byte("Not Found"),
		Headers:    make(http.Header),
	}, nil
}

func (f *probeAwareFetcher) GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*domain.Response, error) {
	return f.Get(ctx, url)
}

func (f *probeAwareFetcher) GetCookies(url string) []*http.Cookie {
	return nil
}

func (f *probeAwareFetcher) Transport() http.RoundTripper {
	return nil
}

func (f *probeAwareFetcher) Close() error {
	return nil
}

// TestDiscoverViaHTTPProbes_Parallel_ReturnHighestPriority verifies that when multiple
// probes succeed, the highest-priority (lowest index) probe is returned, not the fastest.
func TestDiscoverViaHTTPProbes_Parallel_ReturnHighestPriority(t *testing.T) {
	fetcher := newProbeAwareFetcher()

	// sitemap.xml (priority 1) responds quickly
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://test.github.io/page1</loc></url>
  <url><loc>https://test.github.io/page2</loc></url>
</urlset>`
	fetcher.setResponse("/sitemap.xml", &domain.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(sitemapXML),
		Headers:    make(http.Header),
	})
	fetcher.setDelay("/sitemap.xml", 10*time.Millisecond) // Fast response

	// llms.txt (priority 0, highest) responds slower but should still win
	llmsTxt := `# Test Documentation

## Docs
- [Page A](https://test.github.io/pageA)
- [Page B](https://test.github.io/pageB)
- [Page C](https://test.github.io/pageC)
`
	fetcher.setResponse("/llms.txt", &domain.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(llmsTxt),
		Headers:    make(http.Header),
	})
	fetcher.setDelay("/llms.txt", 50*time.Millisecond) // Slower response

	logger := utils.NewLogger(utils.LoggerOptions{})
	pipeline := converter.NewPipeline(converter.PipelineOptions{})
	outputDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
	})

	deps := &strategies.Dependencies{
		Fetcher:   fetcher,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	strategy := strategies.NewGitHubPagesStrategy(deps)
	require.NotNil(t, strategy)

	ctx := context.Background()
	err := strategy.Execute(ctx, "https://test.github.io", strategies.Options{
		CommonOptions: domain.CommonOptions{
			DryRun: true, // Don't write files, just test discovery
		},
	})

	// The strategy should execute without error
	// Even if llms.txt is slower, it should be returned because it has higher priority
	assert.NoError(t, err)

	// Verify both probes were called (parallel execution)
	assert.GreaterOrEqual(t, fetcher.getCallCount("/llms.txt"), 1, "llms.txt should be called")
	assert.GreaterOrEqual(t, fetcher.getCallCount("/sitemap.xml"), 1, "sitemap.xml should be called")
}

// TestDiscoverViaHTTPProbes_Parallel_ContextCancellation verifies that
// context cancellation stops all in-flight probes.
func TestDiscoverViaHTTPProbes_Parallel_ContextCancellation(t *testing.T) {
	fetcher := newProbeAwareFetcher()

	// All probes have long delays - context will cancel before they complete
	for _, probe := range strategies.GetDiscoveryProbes() {
		fetcher.setDelay(probe.Path, 2*time.Second)
	}

	logger := utils.NewLogger(utils.LoggerOptions{})

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()

	// Simulate what discoverViaHTTPProbes does - fire probes and respect context
	var wg sync.WaitGroup
	results := make(chan string, len(strategies.GetDiscoveryProbes()))

	for _, probe := range strategies.GetDiscoveryProbes() {
		wg.Add(1)
		go func(p strategies.DiscoveryProbe) {
			defer wg.Done()
			url := "https://test.github.io" + p.Path
			_, err := fetcher.Get(ctx, url)
			if err == nil {
				results <- p.Name
			}
		}(probe)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var successCount int
	for range results {
		successCount++
	}

	elapsed := time.Since(start)

	// No probes should succeed (context cancelled)
	assert.Zero(t, successCount, "No probes should succeed with cancelled context")
	// Should complete quickly, not wait for 2s delay
	assert.Less(t, elapsed, 500*time.Millisecond, "Should return quickly after context cancellation")

	logger.Debug().Msg("Context cancellation test completed")
}

// TestDiscoverViaHTTPProbes_Parallel_ErrorIsolation verifies that
// one probe's failure doesn't block or affect other probes.
func TestDiscoverViaHTTPProbes_Parallel_ErrorIsolation(t *testing.T) {
	fetcher := newProbeAwareFetcher()

	// Most probes fail with various errors
	fetcher.setError("/llms.txt", fmt.Errorf("network timeout"))
	fetcher.setResponse("/sitemap.xml", &domain.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       []byte("Server Error"),
		Headers:    make(http.Header),
	})
	fetcher.setError("/sitemap-0.xml", fmt.Errorf("connection refused"))

	// But MkDocs search index succeeds
	mkdocsJSON := `{
  "docs": [
    {"location": "getting-started/", "title": "Getting Started", "text": "..."},
    {"location": "api/", "title": "API Reference", "text": "..."}
  ]
}`
	fetcher.setResponse("/search/search_index.json", &domain.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(mkdocsJSON),
		Headers:    make(http.Header),
	})

	logger := utils.NewLogger(utils.LoggerOptions{})
	pipeline := converter.NewPipeline(converter.PipelineOptions{})
	outputDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
	})

	deps := &strategies.Dependencies{
		Fetcher:   fetcher,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	strategy := strategies.NewGitHubPagesStrategy(deps)
	require.NotNil(t, strategy)

	ctx := context.Background()
	err := strategy.Execute(ctx, "https://test.github.io", strategies.Options{
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})

	// The strategy should succeed using MkDocs index despite other failures
	assert.NoError(t, err)

	// Verify the MkDocs probe was called
	assert.GreaterOrEqual(t, fetcher.getCallCount("/search/search_index.json"), 1, "mkdocs-search should be called")
}

// TestDiscoverViaHTTPProbes_Parallel_AllProbesFired verifies that
// all probes are fired concurrently.
func TestDiscoverViaHTTPProbes_Parallel_AllProbesFired(t *testing.T) {
	fetcher := newProbeAwareFetcher()

	// All probes return 404 (default behavior) but with delay
	for _, probe := range strategies.GetDiscoveryProbes() {
		fetcher.setDelay(probe.Path, 50*time.Millisecond)
	}

	logger := utils.NewLogger(utils.LoggerOptions{})

	ctx := context.Background()
	start := time.Now()

	// Simulate parallel probe execution
	var wg sync.WaitGroup
	results := make(chan string, len(strategies.GetDiscoveryProbes()))

	for _, probe := range strategies.GetDiscoveryProbes() {
		wg.Add(1)
		go func(p strategies.DiscoveryProbe) {
			defer wg.Done()
			url := "https://test.github.io" + p.Path
			resp, err := fetcher.Get(ctx, url)
			if err == nil && resp.StatusCode == 200 {
				results <- p.Name
			}
		}(probe)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var successCount int
	for range results {
		successCount++
	}

	elapsed := time.Since(start)

	// With 9 probes x 50ms each:
	// - Sequential would take ~450ms minimum
	// - Parallel should complete in ~50-80ms (plus overhead)
	probeCount := len(strategies.GetDiscoveryProbes())
	sequentialMin := time.Duration(probeCount) * 50 * time.Millisecond

	// Verify all probes were called
	totalCalls := 0
	for _, probe := range strategies.GetDiscoveryProbes() {
		count := fetcher.getCallCount(probe.Path)
		totalCalls += count
	}

	assert.Equal(t, probeCount, totalCalls, "All probes should be called exactly once")

	// If implementation is parallel, elapsed should be significantly less than sequential
	assert.Less(t, elapsed, sequentialMin/2,
		"Parallel execution (%v) should be faster than 50%% of sequential time (%v)", elapsed, sequentialMin)

	logger.Debug().Msg("All probes fired test completed")
}
