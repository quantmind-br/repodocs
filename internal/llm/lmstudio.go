package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// LMStudioProvider implements the LLMProvider interface for LM Studio.
// LM Studio uses the OpenAI chat completions wire format at http://localhost:1234/v1.
// Authentication is optional — no Authorization header is sent when apiKey is empty.
type LMStudioProvider struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
}

// NewLMStudioProvider creates a new LM Studio provider.
func NewLMStudioProvider(cfg ProviderConfig, httpClient *http.Client) (*LMStudioProvider, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	return &LMStudioProvider{
		httpClient:  httpClient,
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

// Name returns the provider name.
func (p *LMStudioProvider) Name() string {
	return "lmstudio"
}

// Complete sends a chat completion request to LM Studio using the OpenAI wire format.
func (p *LMStudioProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
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
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, &domain.LLMError{
			Provider: "lmstudio",
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
		// LM Studio may return non-JSON on 503 (no model loaded)
		if resp.StatusCode == http.StatusServiceUnavailable {
			bodyStr := string(respBody)
			if strings.Contains(strings.ToLower(bodyStr), "no model") {
				return nil, &domain.LLMError{
					Provider:   "lmstudio",
					StatusCode: resp.StatusCode,
					Message:    "no model is loaded in LM Studio — load a model and retry",
					Err:        domain.ErrLLMRequestFailed,
				}
			}
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, &domain.LLMError{
				Provider:   "lmstudio",
				StatusCode: resp.StatusCode,
				Message:    openAIResp.Error.Message,
				Err:        domain.ErrLLMRateLimited,
			}
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, &domain.LLMError{
				Provider:   "lmstudio",
				StatusCode: resp.StatusCode,
				Message:    openAIResp.Error.Message,
				Err:        domain.ErrLLMAuthFailed,
			}
		}
		return nil, &domain.LLMError{
			Provider:   "lmstudio",
			StatusCode: resp.StatusCode,
			Message:    openAIResp.Error.Message,
			Err:        domain.ErrLLMRequestFailed,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp.StatusCode, respBody)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, &domain.LLMError{
			Provider: "lmstudio",
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

// Close releases resources.
func (p *LMStudioProvider) Close() error {
	return nil
}

func (p *LMStudioProvider) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return &domain.LLMError{
			Provider:   "lmstudio",
			StatusCode: statusCode,
			Message:    "authentication failed",
			Err:        domain.ErrLLMAuthFailed,
		}
	case http.StatusTooManyRequests:
		return &domain.LLMError{
			Provider:   "lmstudio",
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Err:        domain.ErrLLMRateLimited,
		}
	case http.StatusServiceUnavailable:
		bodyStr := string(body)
		if strings.Contains(strings.ToLower(bodyStr), "no model") {
			return &domain.LLMError{
				Provider:   "lmstudio",
				StatusCode: statusCode,
				Message:    "no model is loaded in LM Studio — load a model and retry",
				Err:        domain.ErrLLMRequestFailed,
			}
		}
		return &domain.LLMError{
			Provider:   "lmstudio",
			StatusCode: statusCode,
			Message:    string(body),
		}
	default:
		return &domain.LLMError{
			Provider:   "lmstudio",
			StatusCode: statusCode,
			Message:    string(body),
		}
	}
}
