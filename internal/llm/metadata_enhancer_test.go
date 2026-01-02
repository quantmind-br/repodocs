package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetadataEnhancer tests creating a metadata enhancer
func TestNewMetadataEnhancer(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test"}
	enhancer := NewMetadataEnhancer(mockProvider)

	assert.NotNil(t, enhancer)
	assert.Equal(t, mockProvider, enhancer.provider)
}

// TestMetadataEnhancer_Enhance_NilDocument tests enhancing with nil document
func TestMetadataEnhancer_Enhance_NilDocument(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test"}
	enhancer := NewMetadataEnhancer(mockProvider)

	ctx := context.Background()
	err := enhancer.Enhance(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document is nil")
}

// TestMetadataEnhancer_Enhance_Success tests successful enhancement
func TestMetadataEnhancer_Enhance_Success(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name: "test",
		response: &domain.LLMResponse{
			Content: `{"summary": "Test summary", "tags": ["tag1", "tag2"], "category": "guide"}`,
		},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Test content",
	}

	ctx := context.Background()
	err := enhancer.Enhance(ctx, doc)

	require.NoError(t, err)
	assert.Equal(t, "Test summary", doc.Summary)
	assert.Equal(t, []string{"tag1", "tag2"}, doc.Tags)
	assert.Equal(t, "guide", doc.Category)
}

// TestMetadataEnhancer_Enhance_LongContent truncates content
func TestMetadataEnhancer_Enhance_LongContent(t *testing.T) {
	longBytes := make([]byte, 9000)
	for i := range longBytes {
		longBytes[i] = 'a'
	}
	longContent := string(longBytes)

	mockProvider := &mockLLMProvider{
		name: "test",
		response: &domain.LLMResponse{
			Content: `{"summary": "Test", "tags": ["tag1"], "category": "guide"}`,
		},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	doc := &domain.Document{
		Title:   "Test Doc",
		Content: longContent,
	}

	ctx := context.Background()
	err := enhancer.Enhance(ctx, doc)

	// Should succeed with long content
	require.NoError(t, err)
	assert.Equal(t, "Test", doc.Summary)
}

// TestMetadataEnhancer_Enhance_RetryOnFailure tests retry on LLM failure
func TestMetadataEnhancer_Enhance_RetryOnFailure(t *testing.T) {
	calls := 0
	mockProvider := &mockLLMProvider{
		name: "test",
		fn: func() (*domain.LLMResponse, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("first attempt fails")
			}
			return &domain.LLMResponse{
				Content: `{"summary": "Test", "tags": ["tag1"], "category": "guide"}`,
			}, nil
		},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Test content",
	}

	ctx := context.Background()
	err := enhancer.Enhance(ctx, doc)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 2)
}

// TestMetadataEnhancer_Enhance_ContextCancellation tests context cancellation
func TestMetadataEnhancer_Enhance_ContextCancellation(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name:     "test",
		response: &domain.LLMResponse{Content: "test"},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Test content",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := enhancer.Enhance(ctx, doc)

	assert.Error(t, err)
}

// TestMetadataEnhancer_EnhanceAll tests enhancing multiple documents
func TestMetadataEnhancer_EnhanceAll(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name: "test",
		response: &domain.LLMResponse{
			Content: `{"summary": "Test", "tags": ["tag1"], "category": "guide"}`,
		},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	docs := []*domain.Document{
		{Title: "Doc 1", Content: "Content 1"},
		{Title: "Doc 2", Content: "Content 2"},
		{Title: "Doc 3", Content: "Content 3"},
	}

	ctx := context.Background()
	err := enhancer.EnhanceAll(ctx, docs)

	require.NoError(t, err)

	for _, doc := range docs {
		assert.Equal(t, "Test", doc.Summary)
		assert.Equal(t, []string{"tag1"}, doc.Tags)
		assert.Equal(t, "guide", doc.Category)
	}
}

// TestMetadataEnhancer_EnhanceAll_ContextCancellation tests context cancellation in EnhanceAll
func TestMetadataEnhancer_EnhanceAll_ContextCancellation(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test"}

	enhancer := NewMetadataEnhancer(mockProvider)
	docs := []*domain.Document{
		{Title: "Doc 1", Content: "Content 1"},
		{Title: "Doc 2", Content: "Content 2"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := enhancer.EnhanceAll(ctx, docs)

	assert.Error(t, err)
}

// TestMetadataEnhancer_EnhanceAll_EmptySlice tests enhancing empty document slice
func TestMetadataEnhancer_EnhanceAll_EmptySlice(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test"}
	enhancer := NewMetadataEnhancer(mockProvider)

	ctx := context.Background()
	err := enhancer.EnhanceAll(ctx, []*domain.Document{})

	assert.NoError(t, err)
}

// TestMetadataEnhancer_applyMetadata tests applying metadata to document
func TestMetadataEnhancer_applyMetadata(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test"}
	enhancer := NewMetadataEnhancer(mockProvider)

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Test content",
	}

	metadata := &enhancedMetadata{
		Summary:  "Test summary",
		Tags:     []string{"tag1", "tag2", "tag3"},
		Category: "tutorial",
	}

	enhancer.applyMetadata(doc, metadata)

	assert.Equal(t, "Test summary", doc.Summary)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, doc.Tags)
	assert.Equal(t, "tutorial", doc.Category)
}

// TestMetadataEnhancer_tryEnhance tests tryEnhance with different prompts
func TestMetadataEnhancer_tryEnhance(t *testing.T) {
	tests := []struct {
		name      string
		isRetry   bool
		response  string
		wantErr   bool
		validate  func(t *testing.T, metadata *enhancedMetadata)
	}{
		{
			name:     "initial attempt",
			isRetry:  false,
			response: `{"summary": "Test summary", "tags": ["tag1"], "category": "guide"}`,
			wantErr:  false,
			validate: func(t *testing.T, metadata *enhancedMetadata) {
				assert.Equal(t, "Test summary", metadata.Summary)
			},
		},
		{
			name:     "retry attempt",
			isRetry:  true,
			response: `{"summary": "Retry summary", "tags": ["tag2"], "category": "api"}`,
			wantErr:  false,
			validate: func(t *testing.T, metadata *enhancedMetadata) {
				assert.Equal(t, "Retry summary", metadata.Summary)
			},
		},
		{
			name:     "invalid response",
			isRetry:  false,
			response: `not json`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &mockLLMProvider{
				name:     "test",
				response: &domain.LLMResponse{Content: tt.response},
			}

			enhancer := NewMetadataEnhancer(mockProvider)
			ctx := context.Background()

			metadata, err := enhancer.tryEnhance(ctx, "test content", tt.isRetry)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, metadata)
				if tt.validate != nil {
					tt.validate(t, metadata)
				}
			}
		})
	}
}

// TestTruncateForError tests truncation for error messages
func TestTruncateForError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		maxLen int
	}{
		{
			name:  "short string",
			input: "short",
			maxLen: 5,
		},
		{
			name:  "exactly max length",
			input: "abcde",
			maxLen: 5,
		},
		{
			name:  "exceeds max length",
			input: "abcdefghijk",
			maxLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateForError(tt.input)
			if len(tt.input) <= 200 {
				assert.Equal(t, tt.input, result)
			} else {
				assert.Equal(t, 203, len(result)) // 200 + "..."
				assert.Contains(t, result, "...")
			}
		})
	}
}

// TestMetadataEnhancer_Enhance_WithMarkdownCodeBlockInResponse tests handling markdown code blocks
func TestMetadataEnhancer_Enhance_WithMarkdownCodeBlockInResponse(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name: "test",
		response: &domain.LLMResponse{
			Content: "```json\n{\"summary\": \"Test\", \"tags\": [\"tag1\"], \"category\": \"guide\"}\n```",
		},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Test content",
	}

	ctx := context.Background()
	err := enhancer.Enhance(ctx, doc)

	require.NoError(t, err)
	assert.Equal(t, "Test", doc.Summary)
}

// TestMetadataEnhancer_Enhance_WithRetrySuccessAfterInitialFailure tests retry flow
func TestMetadataEnhancer_Enhance_WithRetrySuccessAfterInitialFailure(t *testing.T) {
	attempts := 0
	mockProvider := &mockLLMProvider{
		name: "test",
		fn: func() (*domain.LLMResponse, error) {
			attempts++
			if attempts == 1 {
				return &domain.LLMResponse{Content: "invalid response"}, nil
			}
			return &domain.LLMResponse{
				Content: `{"summary": "Test", "tags": ["tag1"], "category": "guide"}`,
			}, nil
		},
	}

	enhancer := NewMetadataEnhancer(mockProvider)
	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Test content",
	}

	// Note: This test uses the actual retry delay from metadata.go (500ms)
	// We'll use a context with a longer timeout to allow for retries
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := enhancer.Enhance(ctx, doc)

	// Should succeed after retry
	require.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 2)
}
