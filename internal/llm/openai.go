package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type OpenAIProvider struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
}

func NewOpenAIProvider(cfg ProviderConfig, httpClient *http.Client) (*OpenAIProvider, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	return &OpenAIProvider{
		httpClient:  httpClient,
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
	messages := make([]openAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}

	temp := p.temperature
	if req.Temperature != nil {
		temp = *req.Temperature
	}

	openAIReq := openAIRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temp,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, &domain.LLMError{
			Provider: "openai",
			Message:  fmt.Sprintf("request failed: %v", err),
			Err:      err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, &domain.LLMError{
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			Message:    openAIResp.Error.Message,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp.StatusCode, respBody)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, &domain.LLMError{
			Provider: "openai",
			Message:  "no choices in response",
		}
	}

	choice := openAIResp.Choices[0]

	return &domain.LLMResponse{
		Content:      choice.Message.Content,
		Model:        openAIResp.Model,
		FinishReason: choice.FinishReason,
		Usage: domain.LLMUsage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *OpenAIProvider) Close() error {
	return nil
}

func (p *OpenAIProvider) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return &domain.LLMError{
			Provider:   "openai",
			StatusCode: statusCode,
			Message:    "authentication failed",
			Err:        domain.ErrLLMAuthFailed,
		}
	case http.StatusTooManyRequests:
		return &domain.LLMError{
			Provider:   "openai",
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Err:        domain.ErrLLMRateLimited,
		}
	default:
		return &domain.LLMError{
			Provider:   "openai",
			StatusCode: statusCode,
			Message:    string(body),
		}
	}
}
