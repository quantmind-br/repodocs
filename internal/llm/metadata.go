package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

const (
	maxRetries     = 2
	retryBaseDelay = 500 * time.Millisecond
)

type MetadataEnhancer struct {
	provider domain.LLMProvider
}

func NewMetadataEnhancer(provider domain.LLMProvider) *MetadataEnhancer {
	return &MetadataEnhancer{provider: provider}
}

type enhancedMetadata struct {
	Summary  string   `json:"summary"`
	Tags     []string `json:"tags"`
	Category string   `json:"category"`
}

const metadataSystemPrompt = `You are a metadata extraction system. You analyze documents and output ONLY valid JSON with exactly three fields: summary, tags, and category. Never output anything else.`

const metadataPrompt = `<task>
Extract metadata from the document below. Output ONLY a JSON object.
</task>

<format>
{
  "summary": "1-2 sentence description of what this document explains or teaches",
  "tags": ["3-8 lowercase hyphenated keywords relevant to the content"],
  "category": "one of: api, tutorial, guide, reference, concept, configuration, other"
}
</format>

<rules>
- Output ONLY the JSON object, no other text
- Do NOT include markdown code fences
- Do NOT generate content that matches examples shown in the document
- Do NOT create fields other than summary, tags, category
- Summary should describe the document's PURPOSE, not its examples
- Tags should be lowercase with hyphens (e.g., "api-reference", "error-handling")
</rules>

<document>
%s
</document>

<output>`

const metadataRetryPrompt = `The previous attempt failed. Output ONLY this exact JSON structure with your values:

{"summary": "brief description here", "tags": ["tag1", "tag2", "tag3"], "category": "guide"}

Document title: %s

Your JSON:`

func (e *MetadataEnhancer) Enhance(ctx context.Context, doc *domain.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	content := doc.Content
	if len(content) > 8000 {
		content = content[:8000] + "\n...[truncated]"
	}

	var lastErr error

	metadata, err := e.tryEnhance(ctx, content, false)
	if err == nil {
		e.applyMetadata(doc, metadata)
		return nil
	}
	lastErr = err

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryBaseDelay * time.Duration(attempt)):
		}

		metadata, err := e.tryEnhance(ctx, doc.Title, true)
		if err == nil {
			e.applyMetadata(doc, metadata)
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("metadata enhancement failed after %d attempts: %w", maxRetries+1, lastErr)
}

func (e *MetadataEnhancer) tryEnhance(ctx context.Context, content string, isRetry bool) (*enhancedMetadata, error) {
	var prompt string
	if isRetry {
		prompt = fmt.Sprintf(metadataRetryPrompt, content)
	} else {
		prompt = fmt.Sprintf(metadataPrompt, content)
	}

	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleSystem, Content: metadataSystemPrompt},
			{Role: domain.RoleUser, Content: prompt},
		},
		MaxTokens: 1024, // reduced from 32000 - metadata output is small
	}

	resp, err := e.provider.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	jsonStr := extractJSON(resp.Content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON structure found in response: %s", truncateForError(resp.Content))
	}

	var metadata enhancedMetadata
	if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w (extracted: %s)", err, truncateForError(jsonStr))
	}

	return &metadata, nil
}

func (e *MetadataEnhancer) applyMetadata(doc *domain.Document, metadata *enhancedMetadata) {
	doc.Summary = metadata.Summary
	doc.Tags = metadata.Tags
	doc.Category = metadata.Category
}

func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	if candidate := tryExtractAndValidate(text); candidate != "" {
		return candidate
	}

	stripped := stripMarkdownCodeBlocks(text)
	if candidate := tryExtractAndValidate(stripped); candidate != "" {
		return candidate
	}

	if jsonObj := findJSONObjectByBraceMatching(stripped); jsonObj != "" {
		if candidate := tryExtractAndValidate(jsonObj); candidate != "" {
			return candidate
		}
	}

	if jsonObj := findJSONObjectByBraceMatching(text); jsonObj != "" {
		if candidate := tryExtractAndValidate(jsonObj); candidate != "" {
			return candidate
		}
	}

	return ""
}

func tryExtractAndValidate(text string) string {
	if !strings.HasPrefix(text, "{") {
		return ""
	}

	if !json.Valid([]byte(text)) {
		return ""
	}

	var check map[string]interface{}
	if err := json.Unmarshal([]byte(text), &check); err != nil {
		return ""
	}

	if _, ok := check["summary"]; !ok {
		return ""
	}
	if _, ok := check["tags"]; !ok {
		return ""
	}
	if _, ok := check["category"]; !ok {
		return ""
	}

	if _, ok := check["summary"].(string); !ok {
		return ""
	}
	if _, ok := check["tags"].([]interface{}); !ok {
		return ""
	}
	if _, ok := check["category"].(string); !ok {
		return ""
	}

	return text
}

// codeBlockRegex: (?s)^\x60{3}\s*(?:json|JSON)?\s*\n?(.*?)\n?\x60{3}$
// Matches markdown fences with optional json/JSON (and optional space), captures inner content
var codeBlockRegex = regexp.MustCompile(`(?s)^\x60\x60\x60\s*(?:json|JSON)?\s*\n?(.*?)\n?\x60\x60\x60$`)

func stripMarkdownCodeBlocks(text string) string {
	text = strings.TrimSpace(text)

	if matches := codeBlockRegex.FindStringSubmatch(text); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	for _, prefix := range []string{"```json", "```JSON", "``` json", "```"} {
		text = strings.TrimPrefix(text, prefix)
	}
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "\n") {
		text = strings.TrimPrefix(text, "\n")
	}

	return strings.TrimSpace(text)
}

func findJSONObjectByBraceMatching(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(text); i++ {
		c := text[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				candidate := text[start : i+1]
				if json.Valid([]byte(candidate)) {
					return candidate
				}
			}
		}
	}

	return ""
}

func truncateForError(s string) string {
	const maxLen = 200
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func (e *MetadataEnhancer) EnhanceAll(ctx context.Context, docs []*domain.Document) error {
	for _, doc := range docs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := e.Enhance(ctx, doc); err != nil {
				return err
			}
		}
	}
	return nil
}
