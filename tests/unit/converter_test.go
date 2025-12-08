package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "testdata", "fixtures", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to load fixture: %s", name)
	return string(data)
}

func loadGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "testdata", "golden", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to load golden file: %s", name)
	return string(data)
}

func TestPipeline_BasicHTML(t *testing.T) {
	html := loadFixture(t, "basic.html")

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/test")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.Equal(t, "https://example.com/test", doc.URL)
	assert.Equal(t, "Test Page - Basic HTML", doc.Title)
	assert.Contains(t, doc.Content, "# Welcome to Test Page")
	assert.Contains(t, doc.Content, "## Features")
	assert.Contains(t, doc.Content, "- Feature one")
	assert.Contains(t, doc.Content, "**bold text**")
	assert.Contains(t, doc.Content, "*italic text*")
	assert.Greater(t, doc.WordCount, 0)
	assert.Greater(t, doc.CharCount, 0)
	assert.NotEmpty(t, doc.ContentHash)
}

func TestPipeline_WithTables(t *testing.T) {
	html := loadFixture(t, "with_tables.html")

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/api")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	// Content should contain table data (format may vary by converter)
	assert.Contains(t, doc.Content, "API Reference")
	assert.Contains(t, doc.Content, "Endpoints")
	assert.Contains(t, doc.Content, "/api/users")
	assert.Contains(t, doc.Content, "GET")
	assert.Contains(t, doc.Content, "POST")
}

func TestPipeline_WithCodeBlocks(t *testing.T) {
	html := loadFixture(t, "with_code_blocks.html")

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/examples")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	// Code blocks should be preserved (with fenced code blocks)
	assert.Contains(t, doc.Content, "```")
	assert.Contains(t, doc.Content, "package main")
	assert.Contains(t, doc.Content, "fmt.Println")
	// Inline code should be preserved
	assert.Contains(t, doc.Content, "`fmt.Println()`")
}

func TestPipeline_SPADetection(t *testing.T) {
	html := loadFixture(t, "spa_react.html")

	// SPA pages have minimal content
	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/app")
	require.NoError(t, err)

	// SPA pages typically have minimal extractable content
	assert.NotNil(t, doc)
	// Word count should be very low for SPA placeholder pages
	assert.Less(t, doc.WordCount, 50)
}

func TestPipeline_ExtractLinks(t *testing.T) {
	html := loadFixture(t, "basic.html")

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/test")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.NotEmpty(t, doc.Links)
}

func TestPipeline_ExtractHeaders(t *testing.T) {
	html := loadFixture(t, "basic.html")

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/test")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.NotEmpty(t, doc.Headers)
	// Should have h1 and h2 headers
	if h1s, ok := doc.Headers["h1"]; ok {
		assert.Contains(t, strings.Join(h1s, " "), "Welcome")
	}
}

func TestPipeline_ContentSelector(t *testing.T) {
	html := `
	<!DOCTYPE html>
	<html>
	<body>
		<nav>Navigation content</nav>
		<article class="main-content">
			<h1>Main Article</h1>
			<p>This is the main content.</p>
		</article>
		<aside>Sidebar content</aside>
	</body>
	</html>
	`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: "article.main-content",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/article")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.Contains(t, doc.Content, "Main Article")
	assert.Contains(t, doc.Content, "main content")
}

func TestPipeline_EncodingDetection(t *testing.T) {
	// UTF-8 encoded content
	html := `<!DOCTYPE html>
	<html>
	<head><meta charset="UTF-8"></head>
	<body><p>Olá Mundo! Привет мир! こんにちは世界!</p></body>
	</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/unicode")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.Contains(t, doc.Content, "Olá Mundo")
	assert.Contains(t, doc.Content, "Привет мир")
	assert.Contains(t, doc.Content, "こんにちは世界")
}

func TestPipeline_EmptyHTML(t *testing.T) {
	html := `<!DOCTYPE html><html><body></body></html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/empty")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.Equal(t, 0, doc.WordCount)
}

func TestConvertHTML_Convenience(t *testing.T) {
	html := `<html><body><h1>Test</h1><p>Content here.</p></body></html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/test")
	require.NoError(t, err)

	assert.NotNil(t, doc)
	assert.Contains(t, doc.Content, "# Test")
}
