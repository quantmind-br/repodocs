package converter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// MarkdownConverter converts HTML to Markdown
type MarkdownConverter struct {
	domain string
}

// MarkdownOptions contains options for Markdown conversion
type MarkdownOptions struct {
	Domain          string
	CodeBlockStyle  string // "fenced" or "indented"
	HeadingStyle    string // "atx" or "setext"
	BulletListStyle string // "-", "*", or "+"
}

// DefaultMarkdownOptions returns default Markdown options
func DefaultMarkdownOptions() MarkdownOptions {
	return MarkdownOptions{
		CodeBlockStyle:  "fenced",
		HeadingStyle:    "atx",
		BulletListStyle: "-",
	}
}

// NewMarkdownConverter creates a new Markdown converter
func NewMarkdownConverter(opts MarkdownOptions) *MarkdownConverter {
	return &MarkdownConverter{
		domain: opts.Domain,
	}
}

// Convert converts HTML to Markdown
func (c *MarkdownConverter) Convert(html string) (string, error) {
	// html-to-markdown v2 uses ConvertString directly
	markdown, err := md.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	// Clean up the markdown
	markdown = c.cleanMarkdown(markdown)

	return markdown, nil
}

// cleanMarkdown cleans up the converted markdown
func (c *MarkdownConverter) cleanMarkdown(markdown string) string {
	// Remove excessive blank lines (more than 2 consecutive)
	for strings.Contains(markdown, "\n\n\n\n") {
		markdown = strings.ReplaceAll(markdown, "\n\n\n\n", "\n\n\n")
	}

	// Trim leading/trailing whitespace
	markdown = strings.TrimSpace(markdown)

	return markdown
}

// Frontmatter represents YAML frontmatter
type Frontmatter struct {
	Title      string    `yaml:"title"`
	URL        string    `yaml:"url"`
	Source     string    `yaml:"source"`
	FetchedAt  time.Time `yaml:"fetched_at"`
	RenderedJS bool      `yaml:"rendered_js"`
	WordCount  int       `yaml:"word_count"`
}

// GenerateFrontmatter generates YAML frontmatter for a document
func GenerateFrontmatter(doc *domain.Document) (string, error) {
	fm := Frontmatter{
		Title:      doc.Title,
		URL:        doc.URL,
		Source:     doc.SourceStrategy,
		FetchedAt:  doc.FetchedAt,
		RenderedJS: doc.RenderedWithJS,
		WordCount:  doc.WordCount,
	}

	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("---\n%s---\n\n", string(data)), nil
}

// AddFrontmatter adds YAML frontmatter to markdown content
func AddFrontmatter(markdown string, doc *domain.Document) (string, error) {
	frontmatter, err := GenerateFrontmatter(doc)
	if err != nil {
		return "", err
	}

	return frontmatter + markdown, nil
}

// CountWords counts words in text
func CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// CountChars counts characters in text
func CountChars(text string) int {
	return len(text)
}

// StripMarkdown removes markdown formatting to get plain text
func StripMarkdown(markdown string) string {
	// Remove code blocks
	markdown = removeCodeBlocks(markdown)

	// Remove links but keep text: [text](url) -> text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	markdown = linkRegex.ReplaceAllString(markdown, "$1")

	// Remove images: ![alt](url) -> alt
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	markdown = imageRegex.ReplaceAllString(markdown, "$1")

	// Remove emphasis: **bold** -> bold, *italic* -> italic
	markdown = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(markdown, "$1")
	markdown = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(markdown, "$1")
	markdown = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(markdown, "$1")
	markdown = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(markdown, "$1")

	// Remove headers: # Header -> Header
	markdown = regexp.MustCompile(`(?m)^#{1,6}\s+`).ReplaceAllString(markdown, "")

	// Remove horizontal rules
	markdown = regexp.MustCompile(`(?m)^[\-*_]{3,}$`).ReplaceAllString(markdown, "")

	// Remove blockquotes
	markdown = regexp.MustCompile(`(?m)^>\s+`).ReplaceAllString(markdown, "")

	// Remove list markers
	markdown = regexp.MustCompile(`(?m)^[\s]*[\-*+]\s+`).ReplaceAllString(markdown, "")
	markdown = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`).ReplaceAllString(markdown, "")

	return strings.TrimSpace(markdown)
}

// removeCodeBlocks removes fenced code blocks
func removeCodeBlocks(markdown string) string {
	// Remove fenced code blocks
	fenced := regexp.MustCompile("(?s)```[^`]*```")
	markdown = fenced.ReplaceAllString(markdown, "")

	// Remove indented code blocks (lines starting with 4 spaces or tab)
	indented := regexp.MustCompile(`(?m)^(    |\t).*$`)
	markdown = indented.ReplaceAllString(markdown, "")

	return markdown
}
