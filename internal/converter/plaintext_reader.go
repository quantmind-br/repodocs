package converter

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

type PlainTextReader struct{}

func NewPlainTextReader() *PlainTextReader {
	return &PlainTextReader{}
}

func (r *PlainTextReader) Read(content, sourceURL string) (*domain.Document, error) {
	content = strings.TrimSpace(content)

	title := r.extractTitle(content, sourceURL)
	description := r.extractDescription(content)
	links := r.extractLinks(content, sourceURL)

	wordCount := CountWords(content)
	charCount := CountChars(content)
	contentHash := r.calculateHash(content)

	return &domain.Document{
		URL:            sourceURL,
		Title:          title,
		Description:    description,
		Content:        content,
		HTMLContent:    "",
		FetchedAt:      time.Now(),
		ContentHash:    contentHash,
		WordCount:      wordCount,
		CharCount:      charCount,
		Links:          links,
		Headers:        make(map[string][]string),
		RenderedWithJS: false,
		SourceStrategy: "",
		CacheHit:       false,
	}, nil
}

func (r *PlainTextReader) extractTitle(content, sourceURL string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			return strings.TrimSpace(title)
		}

		if len(trimmed) > 100 {
			return trimmed[:97] + "..."
		}
		return trimmed
	}

	parsed, err := url.Parse(sourceURL)
	if err == nil {
		filename := path.Base(parsed.Path)
		if filename != "" && filename != "." && filename != "/" {
			return strings.TrimSuffix(filename, ".txt")
		}
	}

	return ""
}

func (r *PlainTextReader) extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	var descLines []string
	started := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !started {
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			started = true
		}

		if trimmed == "" && started {
			break
		}

		descLines = append(descLines, trimmed)
	}

	if len(descLines) > 0 {
		desc := strings.Join(descLines, " ")
		if len(desc) > 300 {
			return desc[:297] + "..."
		}
		return desc
	}

	return ""
}

var plainTextLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

func (r *PlainTextReader) extractLinks(content, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)
	base, _ := url.Parse(baseURL)

	matches := plainTextLinkRegex.FindAllStringSubmatch(content, -1)
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

	return links
}

func (r *PlainTextReader) calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
