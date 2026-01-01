package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractJSON_ValidMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"summary": "test", "tags": ["a"], "category": "guide"}`,
			expected: `{"summary": "test", "tags": ["a"], "category": "guide"}`,
		},
		{
			name:     "with markdown fence",
			input:    "```json\n{\"summary\": \"test\", \"tags\": [\"a\"], \"category\": \"guide\"}\n```",
			expected: `{"summary": "test", "tags": ["a"], "category": "guide"}`,
		},
		{
			name:     "with markdown fence no newline",
			input:    "```json{\"summary\": \"test\", \"tags\": [\"a\"], \"category\": \"guide\"}```",
			expected: `{"summary": "test", "tags": ["a"], "category": "guide"}`,
		},
		{
			name:     "with surrounding text",
			input:    "Here is the metadata:\n{\"summary\": \"test\", \"tags\": [\"a\"], \"category\": \"guide\"}\nDone.",
			expected: `{"summary": "test", "tags": ["a"], "category": "guide"}`,
		},
		{
			name:     "with uppercase JSON fence",
			input:    "```JSON\n{\"summary\": \"test\", \"tags\": [\"a\"], \"category\": \"guide\"}\n```",
			expected: `{"summary": "test", "tags": ["a"], "category": "guide"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

func TestExtractJSON_InvalidStructure(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "wrong keys - multilingual_messages",
			input: `{"multilingual_messages": {"en": "hello"}}`,
		},
		{
			name:  "missing summary",
			input: `{"tags": ["a"], "category": "guide"}`,
		},
		{
			name:  "missing tags",
			input: `{"summary": "test", "category": "guide"}`,
		},
		{
			name:  "missing category",
			input: `{"summary": "test", "tags": ["a"]}`,
		},
		{
			name:  "wrong type for summary",
			input: `{"summary": 123, "tags": ["a"], "category": "guide"}`,
		},
		{
			name:  "wrong type for tags",
			input: `{"summary": "test", "tags": "a", "category": "guide"}`,
		},
		{
			name:  "wrong type for category",
			input: `{"summary": "test", "tags": ["a"], "category": 123}`,
		},
		{
			name:  "extra fields only",
			input: `{"foo": "bar", "baz": 123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Empty(t, result, "should reject invalid structure")
		})
	}
}

func TestExtractJSON_MalformedJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "truncated",
			input: `{"summary": "test", "tags": ["a"], "category": "gu`,
		},
		{
			name:  "missing closing brace",
			input: `{"summary": "test", "tags": ["a"], "category": "guide"`,
		},
		{
			name:  "invalid syntax",
			input: `{"summary": test, "tags": ["a"], "category": "guide"}`,
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "just text",
			input: "This is not JSON at all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Empty(t, result, "should reject malformed JSON")
		})
	}
}

func TestStripMarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json fence with newlines",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON uppercase",
			input:    "```JSON\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "plain fence",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "no fence",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "fence with space",
			input:    "``` json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "inline fence no newlines",
			input:    "```json{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCodeBlocks(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTryExtractAndValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid metadata",
			input:    `{"summary": "test", "tags": ["a", "b"], "category": "guide"}`,
			expected: `{"summary": "test", "tags": ["a", "b"], "category": "guide"}`,
		},
		{
			name:     "empty tags array",
			input:    `{"summary": "test", "tags": [], "category": "guide"}`,
			expected: `{"summary": "test", "tags": [], "category": "guide"}`,
		},
		{
			name:     "missing required field",
			input:    `{"summary": "test", "tags": ["a"]}`,
			expected: "",
		},
		{
			name:     "not starting with brace",
			input:    `["summary", "test"]`,
			expected: "",
		},
		{
			name:     "invalid json",
			input:    `{"summary": "test"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tryExtractAndValidate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
