package llm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
)

func TestMetadataEnhancer_Enhance_SuccessFirstAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some documentation content",
	}

	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Return(&domain.LLMResponse{
			Content: `{"summary": "A summary", "tags": ["tag1", "tag2"], "category": "api"}`,
		}, nil).
		Times(1)

	err := enhancer.Enhance(ctx, doc)
	require.NoError(t, err)
	assert.Equal(t, "A summary", doc.Summary)
	assert.Equal(t, []string{"tag1", "tag2"}, doc.Tags)
	assert.Equal(t, "api", doc.Category)
}

func TestMetadataEnhancer_Enhance_RetryOnWrongStructure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some documentation content",
	}

	gomock.InOrder(
		mockProvider.EXPECT().
			Complete(ctx, gomock.Any()).
			Return(&domain.LLMResponse{
				Content: `{"multilingual_messages": {"en": "hello"}}`,
			}, nil),
		mockProvider.EXPECT().
			Complete(ctx, gomock.Any()).
			Return(&domain.LLMResponse{
				Content: `{"summary": "Retry success", "tags": ["test"], "category": "guide"}`,
			}, nil),
	)

	err := enhancer.Enhance(ctx, doc)
	require.NoError(t, err)
	assert.Equal(t, "Retry success", doc.Summary)
	assert.Equal(t, []string{"test"}, doc.Tags)
	assert.Equal(t, "guide", doc.Category)
}

func TestMetadataEnhancer_Enhance_FailAfterMaxRetries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some content",
	}

	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Return(&domain.LLMResponse{Content: `{"wrong": "structure"}`}, nil).
		Times(3)

	err := enhancer.Enhance(ctx, doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata enhancement failed after 3 attempts")
}

func TestMetadataEnhancer_Enhance_ProviderError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Content",
	}

	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).
		Times(3)

	err := enhancer.Enhance(ctx, doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM completion failed")
}

func TestMetadataEnhancer_Enhance_NilDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	err := enhancer.Enhance(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "document is nil")
}

func TestMetadataEnhancer_EnhanceAll_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	docs := []*domain.Document{
		{Title: "Doc 1", Content: "Content 1"},
		{Title: "Doc 2", Content: "Content 2"},
		{Title: "Doc 3", Content: "Content 3"},
	}

	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Return(&domain.LLMResponse{
			Content: `{"summary": "Test", "tags": ["test"], "category": "guide"}`,
		}, nil).
		Times(len(docs))

	err := enhancer.EnhanceAll(ctx, docs)
	require.NoError(t, err)

	for _, doc := range docs {
		assert.Equal(t, "Test", doc.Summary)
		assert.Equal(t, []string{"test"}, doc.Tags)
		assert.Equal(t, "guide", doc.Category)
	}
}

func TestMetadataEnhancer_EnhanceAll_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	docs := []*domain.Document{
		{Title: "Doc 1", Content: "Content 1"},
		{Title: "Doc 2", Content: "Content 2"},
	}

	err := enhancer.EnhanceAll(ctx, docs)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMetadataEnhancer_EnhanceAll_EmptyDocs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	// Should not call provider at all
	err := enhancer.EnhanceAll(ctx, []*domain.Document{})
	require.NoError(t, err)
}

func TestMetadataEnhancer_ContentTruncation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	// Create content longer than 8000 chars
	longContent := string(make([]byte, 10000))
	for i := range longContent {
		longContent = longContent[:i] + "a" + longContent[i+1:]
	}

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: longContent,
	}

	var capturedPrompt string
	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Do(func(_ context.Context, req *domain.LLMRequest) {
			// Capture the user prompt
			if len(req.Messages) > 1 {
				capturedPrompt = req.Messages[1].Content
			}
		}).
		Return(&domain.LLMResponse{
			Content: `{"summary": "Test", "tags": ["test"], "category": "guide"}`,
		}, nil)

	err := enhancer.Enhance(ctx, doc)
	require.NoError(t, err)

	// Verify content was truncated
	assert.Contains(t, capturedPrompt, "...[truncated]")
}

func TestMetadataEnhancer_MissingFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some content",
	}

	tests := []struct {
		name    string
		jsonResp string
	}{
		{
			name:     "missing_summary",
			jsonResp: `{"tags": ["test"], "category": "guide"}`,
		},
		{
			name:     "missing_tags",
			jsonResp: `{"summary": "Test", "category": "guide"}`,
		},
		{
			name:     "missing_category",
			jsonResp: `{"summary": "Test", "tags": ["test"]}`,
		},
		{
			name:     "wrong_type_tags",
			jsonResp: `{"summary": "Test", "tags": "not-array", "category": "guide"}`,
		},
		{
			name:     "wrong_type_summary",
			jsonResp: `{"summary": 123, "tags": ["test"], "category": "guide"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider.EXPECT().
				Complete(ctx, gomock.Any()).
				Return(&domain.LLMResponse{Content: tt.jsonResp}, nil).
				Times(3)

			err := enhancer.Enhance(ctx, doc)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "metadata enhancement failed")
		})
	}
}

func TestMetadataEnhancer_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some content",
	}

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "malformed_json",
			content: `{"summary": "test", "tags": ["test"]`,
		},
		{
			name:    "plain_text",
			content: `This is just plain text with no JSON structure`,
		},
		{
			name:    "empty_response",
			content: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider.EXPECT().
				Complete(ctx, gomock.Any()).
				Return(&domain.LLMResponse{Content: tt.content}, nil).
				Times(3)

			err := enhancer.Enhance(ctx, doc)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "metadata enhancement failed")
		})
	}
}

func TestMetadataEnhancer_Enhance_MarkdownFencedResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some documentation content",
	}

	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Return(&domain.LLMResponse{
			Content: "Here is the JSON:\n```json\n{\n\t\"summary\": \"A summary\",\n\t\"tags\": [\"tag1\"],\n\t\"category\": \"guide\"\n}\n```",
		}, nil).
		Times(1)

	err := enhancer.Enhance(ctx, doc)
	require.NoError(t, err)
	assert.Equal(t, "A summary", doc.Summary)
	assert.Equal(t, []string{"tag1"}, doc.Tags)
	assert.Equal(t, "guide", doc.Category)
}

func TestMetadataEnhancer_Enhance_EmbeddedJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)
	ctx := context.Background()

	doc := &domain.Document{
		Title:   "Test Doc",
		Content: "Some documentation content",
	}

	mockProvider.EXPECT().
		Complete(ctx, gomock.Any()).
		Return(&domain.LLMResponse{
			Content: "Sure, here it is: {\"summary\": \"Embed\", \"tags\": [], \"category\": \"other\"} hope this helps.",
		}, nil).
		Times(1)

	err := enhancer.Enhance(ctx, doc)
	require.NoError(t, err)
	assert.Equal(t, "Embed", doc.Summary)
	assert.Empty(t, doc.Tags)
	assert.Equal(t, "other", doc.Category)
}
