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

func TestMetadataEnhancer_Enhance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := mocks.NewMockLLMProvider(ctrl)
	enhancer := llm.NewMetadataEnhancer(mockProvider)

	ctx := context.Background()

	tests := []struct {
		name          string
		docContent    string
		mockResponse  string
		mockError     error
		expectedError bool
		expectedMeta  func(*domain.Document)
	}{
		{
			name:       "Success - Clean JSON",
			docContent: "Some documentation content",
			mockResponse: `{
				"summary": "A summary",
				"tags": ["tag1", "tag2"],
				"category": "api"
			}`,
			expectedError: false,
			expectedMeta: func(d *domain.Document) {
				assert.Equal(t, "A summary", d.Summary)
				assert.Equal(t, []string{"tag1", "tag2"}, d.Tags)
				assert.Equal(t, "api", d.Category)
			},
		},
		{
			name:          "Success - Markdown Block",
			docContent:    "Some documentation content",
			mockResponse:  "Here is the JSON:\n```json\n{\n\t\"summary\": \"A summary\",\n\t\"tags\": [\"tag1\"],\n\t\"category\": \"guide\"\n}\n```",
			expectedError: false,
			expectedMeta: func(d *domain.Document) {
				assert.Equal(t, "A summary", d.Summary)
				assert.Equal(t, []string{"tag1"}, d.Tags)
				assert.Equal(t, "guide", d.Category)
			},
		},
		{
			name:          "Success - Embedded JSON",
			docContent:    "Some documentation content",
			mockResponse:  "Sure, here it is: {\"summary\": \"Embed\", \"tags\": [], \"category\": \"other\"} hope this helps.",
			expectedError: false,
			expectedMeta: func(d *domain.Document) {
				assert.Equal(t, "Embed", d.Summary)
				assert.Empty(t, d.Tags)
				assert.Equal(t, "other", d.Category)
			},
		},
		{
			name:          "Failure - Provider Error",
			docContent:    "Content",
			mockError:     fmt.Errorf("provider failed"),
			expectedError: true,
		},
		{
			name:          "Failure - Invalid JSON",
			docContent:    "Content",
			mockResponse:  "This is not JSON",
			expectedError: true,
		},
		{
			name:          "Failure - Empty Response",
			docContent:    "Content",
			mockResponse:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &domain.Document{
				Title:   "Test Doc",
				Content: tt.docContent,
			}

			if tt.mockError != nil {
				mockProvider.EXPECT().
					Complete(ctx, gomock.Any()).
					Return(nil, tt.mockError)
			} else {
				mockProvider.EXPECT().
					Complete(ctx, gomock.Any()).
					Return(&domain.LLMResponse{Content: tt.mockResponse}, nil)
			}

			err := enhancer.Enhance(ctx, doc)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedMeta != nil {
					tt.expectedMeta(doc)
				}
			}
		})
	}
}
