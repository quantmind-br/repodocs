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

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            ollamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration"`
	LoadDuration       int64         `json:"load_duration"`
	PromptEvalCount    int64         `json:"prompt_eval_count"`
	PromptEvalDuration int64         `json:"prompt_eval_duration"`
	EvalCount          int64         `json:"eval_count"`
	EvalDuration       int64         `json:"eval_duration"`
	Error              string        `json:"error,omitempty"`
}

type OllamaProvider struct {
	httpClient  *http.Client
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
}

func NewOllamaProvider(cfg ProviderConfig, httpClient *http.Client) (*OllamaProvider, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	return &OllamaProvider{
		httpClient:  httpClient,
		baseURL:     baseURL,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
	messages := make([]ollamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ollamaMessage{
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

	ollamaReq := ollamaRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
	}

	if maxTokens > 0 || temp > 0 {
		ollamaReq.Options = &ollamaOptions{
			Temperature: temp,
			NumPredict:  maxTokens,
		}
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, &domain.LLMError{
			Provider: "ollama",
			Message:  fmt.Sprintf("request failed: %v", err),
			Err:      err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var ollamaResp ollamaResponse
		if json.Unmarshal(respBody, &ollamaResp) == nil && ollamaResp.Error != "" {
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, &domain.LLMError{
					Provider:   "ollama",
					StatusCode: resp.StatusCode,
					Message:    ollamaResp.Error,
					Err:        domain.ErrLLMRateLimited,
				}
			}
			return nil, &domain.LLMError{
				Provider:   "ollama",
				StatusCode: resp.StatusCode,
				Message:    ollamaResp.Error,
				Err:        domain.ErrLLMRequestFailed,
			}
		}
		return nil, p.handleHTTPError(resp.StatusCode, respBody)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, &domain.LLMError{
			Provider: "ollama",
			Message:  ollamaResp.Error,
			Err:      domain.ErrLLMRequestFailed,
		}
	}

	finishReason := "stop"
	if !ollamaResp.Done {
		finishReason = "length"
	}

	return &domain.LLMResponse{
		Content:      ollamaResp.Message.Content,
		Model:        ollamaResp.Model,
		FinishReason: finishReason,
		Usage: domain.LLMUsage{
			PromptTokens:     int(ollamaResp.PromptEvalCount),
			CompletionTokens: int(ollamaResp.EvalCount),
			TotalTokens:      int(ollamaResp.PromptEvalCount + ollamaResp.EvalCount),
		},
	}, nil
}

func (p *OllamaProvider) Close() error {
	return nil
}

func (p *OllamaProvider) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusTooManyRequests:
		return &domain.LLMError{
			Provider:   "ollama",
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Err:        domain.ErrLLMRateLimited,
		}
	default:
		return &domain.LLMError{
			Provider:   "ollama",
			StatusCode: statusCode,
			Message:    string(body),
		}
	}
}
