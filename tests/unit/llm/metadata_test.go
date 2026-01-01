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
