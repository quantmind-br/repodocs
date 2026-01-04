package converter

import (
	"fmt"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// Pre-compiled regex patterns for markdown stripping
var (
	linkRegex              = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	imageRegex             = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	boldAsterisksRegex     = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicAsterisksRegex   = regexp.MustCompile(`\*([^*]+)\*`)
	boldUnderscoresRegex   = regexp.MustCompile(`__([^_]+)__`)
	italicUnderscoresRegex = regexp.MustCompile(`_([^_]+)_`)
	headersRegex           = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	horizontalRuleRegex    = regexp.MustCompile(`(?m)^[\-*_]{3,}$`)
	blockquoteRegex        = regexp.MustCompile(`(?m)^>\s+`)
	unorderedListRegex     = regexp.MustCompile(`(?m)^[\s]*[\-*+]\s+`)
	orderedListRegex       = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`)
	fencedCodeBlockRegex   = regexp.MustCompile(`(?s)\`\`\`[^\`]*\`\`\``)
	indentedCodeBlockRegex = regexp.MustCompile(`(?m)^(    |\t).*$`)
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

// GenerateFrontmatter generates YAML frontmatter for a document
func GenerateFrontmatter(doc *domain.Document) (string, error) {
	fm := doc.ToFrontmatter()
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
	markdown = linkRegex.ReplaceAllString(markdown, "$1")

	// Remove images: ![alt](url) -> alt
	markdown = imageRegex.ReplaceAllString(markdown, "$1")

	// Remove emphasis: **bold** -> bold, *italic* -> italic
	markdown = boldAsterisksRegex.ReplaceAllString(markdown, "$1")
	markdown = italicAsterisksRegex.ReplaceAllString(markdown, "$1")
	markdown = boldUnderscoresRegex.ReplaceAllString(markdown, "$1")
	markdown = italicUnderscoresRegex.ReplaceAllString(markdown, "$1")

	// Remove headers: # Header -> Header
	markdown = headersRegex.ReplaceAllString(markdown, "")

	// Remove horizontal rules
	markdown = horizontalRuleRegex.ReplaceAllString(markdown, "")

	// Remove blockquotes
	markdown = blockquoteRegex.ReplaceAllString(markdown, "")

	// Remove list markers
	markdown = unorderedListRegex.ReplaceAllString(markdown, "")
	markdown = orderedListRegex.ReplaceAllString(markdown, "")

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
