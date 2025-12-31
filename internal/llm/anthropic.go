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

const anthropicVersion = "2023-06-01"

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type AnthropicProvider struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
}

func NewAnthropicProvider(cfg ProviderConfig, httpClient *http.Client) (*AnthropicProvider, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &AnthropicProvider{
		httpClient:  httpClient,
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       cfg.Model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
	var systemPrompt string
	messages := make([]anthropicMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == domain.RoleSystem {
			systemPrompt = msg.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}

	anthropicReq := anthropicRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    systemPrompt,
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, &domain.LLMError{
			Provider: "anthropic",
			Message:  fmt.Sprintf("request failed: %v", err),
			Err:      err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, &domain.LLMError{
			Provider:   "anthropic",
			StatusCode: resp.StatusCode,
			Message:    anthropicResp.Error.Message,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp.StatusCode, respBody)
	}

	var sb strings.Builder
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	content := sb.String()

	return &domain.LLMResponse{
		Content:      content,
		Model:        anthropicResp.Model,
		FinishReason: anthropicResp.StopReason,
		Usage: domain.LLMUsage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

func (p *AnthropicProvider) Close() error {
	return nil
}

func (p *AnthropicProvider) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return &domain.LLMError{
			Provider:   "anthropic",
			StatusCode: statusCode,
			Message:    "authentication failed",
			Err:        domain.ErrLLMAuthFailed,
		}
	case http.StatusTooManyRequests:
		return &domain.LLMError{
			Provider:   "anthropic",
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Err:        domain.ErrLLMRateLimited,
		}
	default:
		return &domain.LLMError{
			Provider:   "anthropic",
			StatusCode: statusCode,
			Message:    string(body),
		}
	}
}
