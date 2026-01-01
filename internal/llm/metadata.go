package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/quantmind-br/repodocs-go/internal/domain"
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

const metadataSystemPrompt = `You are a strict JSON generator. Output valid JSON only. No text. No thinking.`

const metadataPrompt = `Generate a JSON object for this document.

Format:
{"summary": "string", "tags": ["string"], "category": "string"}

Categories: api, tutorial, guide, reference, concept, configuration, other

Document Content:
%s

JSON Output:`

func (e *MetadataEnhancer) Enhance(ctx context.Context, doc *domain.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	content := doc.Content
	if len(content) > 8000 {
		content = content[:8000] + "\n...[truncated]"
	}

	prompt := fmt.Sprintf(metadataPrompt, content)

	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleSystem, Content: metadataSystemPrompt},
			{Role: domain.RoleUser, Content: prompt},
		},
		MaxTokens: 4096,
	}

	resp, err := e.provider.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM completion failed: %w", err)
	}

	var metadata enhancedMetadata
	jsonStr := extractJSON(resp.Content)
	if jsonStr == "" {
		return fmt.Errorf("failed to extract JSON from LLM response: no valid JSON found in: %s", truncateForError(resp.Content))
	}

	if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
		return fmt.Errorf("failed to parse LLM response JSON: %w (extracted: %s)", err, truncateForError(jsonStr))
	}

	doc.Summary = metadata.Summary
	doc.Tags = metadata.Tags
	doc.Category = metadata.Category

	return nil
}

func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	if json.Valid([]byte(text)) && strings.HasPrefix(text, "{") {
		return text
	}

	text = stripMarkdownCodeBlocks(text)
	if json.Valid([]byte(text)) && strings.HasPrefix(text, "{") {
		return text
	}

	if jsonObj := findJSONObjectByBraceMatching(text); jsonObj != "" {
		return jsonObj
	}

	re := regexp.MustCompile(`\{[^{}]*"summary"[^{}]*"tags"[^{}]*"category"[^{}]*\}`)
	if match := re.FindString(text); match != "" && json.Valid([]byte(match)) {
		return match
	}

	return ""
}

func stripMarkdownCodeBlocks(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
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
