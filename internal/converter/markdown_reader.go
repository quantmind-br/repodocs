package converter

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"gopkg.in/yaml.v3"
)

// MarkdownReader reads and extracts metadata from markdown content
// without using HTML parsing (avoids the 512 node limit issue).
type MarkdownReader struct{}

// NewMarkdownReader creates a new markdown reader.
func NewMarkdownReader() *MarkdownReader {
	return &MarkdownReader{}
}

// Frontmatter represents YAML frontmatter commonly found in markdown files.
type Frontmatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Summary     string   `yaml:"summary"`
	Author      string   `yaml:"author"`
	Date        string   `yaml:"date"`
	Tags        []string `yaml:"tags"`
	Category    string   `yaml:"category"`
}

// Read processes markdown content and returns a Document.
func (r *MarkdownReader) Read(content, sourceURL string) (*domain.Document, error) {
	frontmatter, body := r.parseFrontmatter(content)
	title := r.extractTitle(frontmatter, body)
	description := r.extractDescription(frontmatter, body)
	headers := r.extractHeaders(body)
	links := r.extractLinks(body, sourceURL)

	plainText := StripMarkdown(body)
	wordCount := CountWords(plainText)
	charCount := CountChars(plainText)
	contentHash := r.calculateHash(body)

	return &domain.Document{
		URL:            sourceURL,
		Title:          title,
		Description:    description,
		Content:        body,
		HTMLContent:    "",
		FetchedAt:      time.Now(),
		ContentHash:    contentHash,
		WordCount:      wordCount,
		CharCount:      charCount,
		Links:          links,
		Headers:        headers,
		RenderedWithJS: false,
		SourceStrategy: "",
		CacheHit:       false,
	}, nil
}

func (r *MarkdownReader) parseFrontmatter(content string) (*Frontmatter, string) {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "---") {
		return nil, content
	}

	rest := content[3:]

	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	}

	lines := strings.Split(rest, "\n")
	yamlLines := []string{}
	closingIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "---" {
			closingIdx = i
			break
		}
		yamlLines = append(yamlLines, line)
	}

	if closingIdx == -1 {
		return nil, content
	}

	yamlContent := strings.Join(yamlLines, "\n")
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, content
	}

	bodyLines := lines[closingIdx+1:]
	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return &fm, body
}

func (r *MarkdownReader) extractTitle(fm *Frontmatter, body string) string {
	if fm != nil && fm.Title != "" {
		return fm.Title
	}

	inCodeBlock := false
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			title = strings.TrimRight(title, "#")
			return strings.TrimSpace(title)
		}
	}

	return ""
}

var numberedListRegex = regexp.MustCompile(`^\d+\.\s`)
var horizontalRuleRegex = regexp.MustCompile(`^[-*_]{3,}$`)

func (r *MarkdownReader) extractDescription(fm *Frontmatter, body string) string {
	if fm != nil {
		if fm.Description != "" {
			return fm.Description
		}
		if fm.Summary != "" {
			return fm.Summary
		}
	}

	inCodeBlock := false
	lines := strings.Split(body, "\n")
	var paragraphLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "+ ") ||
			numberedListRegex.MatchString(trimmed) {
			continue
		}

		if trimmed == "" {
			if len(paragraphLines) > 0 {
				break
			}
			continue
		}

		if horizontalRuleRegex.MatchString(trimmed) {
			continue
		}

		paragraphLines = append(paragraphLines, trimmed)
	}

	if len(paragraphLines) > 0 {
		desc := strings.Join(paragraphLines, " ")
		if len(desc) > 300 {
			desc = desc[:297] + "..."
		}
		return desc
	}

	return ""
}

var headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

func (r *MarkdownReader) extractHeaders(body string) map[string][]string {
	headers := make(map[string][]string)

	inCodeBlock := false
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		matches := headingRegex.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			level := len(matches[1])
			text := strings.TrimSpace(matches[2])

			text = strings.TrimRight(text, "#")
			text = strings.TrimSpace(text)

			if text != "" {
				key := "h" + string(rune('0'+level))
				headers[key] = append(headers[key], text)
			}
		}
	}

	return headers
}

var markdownLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

func (r *MarkdownReader) extractLinks(body, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)

	base, _ := url.Parse(baseURL)

	inCodeBlock := false
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		matches := markdownLinkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				href := strings.TrimSpace(match[2])

				if href == "" ||
					strings.HasPrefix(href, "#") ||
					strings.HasPrefix(href, "javascript:") ||
					strings.HasPrefix(href, "mailto:") ||
					strings.HasPrefix(href, "tel:") {
					continue
				}

				if base != nil && !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
					if refURL, err := url.Parse(href); err == nil {
						href = base.ResolveReference(refURL).String()
					}
				}

				if !seen[href] {
					seen[href] = true
					links = append(links, href)
				}
			}
		}
	}

	return links
}

func (r *MarkdownReader) calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
