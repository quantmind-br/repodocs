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

type googleRequest struct {
	Contents          []googleContent  `json:"contents"`
	SystemInstruction *googleContent   `json:"systemInstruction,omitempty"`
	GenerationConfig  *googleGenConfig `json:"generationConfig,omitempty"`
}

type googleContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []googlePart `json:"parts"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

type GoogleProvider struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
}

func NewGoogleProvider(cfg ProviderConfig, httpClient *http.Client) (*GoogleProvider, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	return &GoogleProvider{
		httpClient:  httpClient,
		apiKey:      cfg.APIKey,
		baseURL:     baseURL,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (p *GoogleProvider) Name() string {
	return "google"
}

func (p *GoogleProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
	var systemInstruction *googleContent
	contents := make([]googleContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		switch msg.Role {
		case domain.RoleSystem:
			systemInstruction = &googleContent{
				Parts: []googlePart{{Text: msg.Content}},
			}
		case domain.RoleUser:
			contents = append(contents, googleContent{
				Role:  "user",
				Parts: []googlePart{{Text: msg.Content}},
			})
		case domain.RoleAssistant:
			contents = append(contents, googleContent{
				Role:  "model",
				Parts: []googlePart{{Text: msg.Content}},
			})
		}
	}

	googleReq := googleRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}

	temp := p.temperature
	if req.Temperature != nil {
		temp = *req.Temperature
	}

	if maxTokens > 0 || temp > 0 {
		googleReq.GenerationConfig = &googleGenConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     temp,
		}
	}

	body, err := json.Marshal(googleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", p.baseURL, p.model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, &domain.LLMError{
			Provider: "google",
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
		var googleResp googleResponse
		if json.Unmarshal(respBody, &googleResp) == nil && googleResp.Error != nil {
			if resp.StatusCode == http.StatusTooManyRequests || googleResp.Error.Status == "RESOURCE_EXHAUSTED" {
				return nil, &domain.LLMError{
					Provider:   "google",
					StatusCode: googleResp.Error.Code,
					Message:    googleResp.Error.Message,
					Err:        domain.ErrLLMRateLimited,
				}
			}
			return nil, &domain.LLMError{
				Provider:   "google",
				StatusCode: googleResp.Error.Code,
				Message:    googleResp.Error.Message,
				Err:        domain.ErrLLMRequestFailed,
			}
		}
		return nil, p.handleHTTPError(resp.StatusCode, respBody)
	}

	var googleResp googleResponse
	if err := json.Unmarshal(respBody, &googleResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if googleResp.Error != nil {
		return nil, &domain.LLMError{
			Provider:   "google",
			StatusCode: googleResp.Error.Code,
			Message:    googleResp.Error.Message,
			Err:        domain.ErrLLMRequestFailed,
		}
	}

	if len(googleResp.Candidates) == 0 {
		return nil, &domain.LLMError{
			Provider: "google",
			Message:  "no candidates in response",
		}
	}

	candidate := googleResp.Candidates[0]

	var sb strings.Builder
	for _, part := range candidate.Content.Parts {
		sb.WriteString(part.Text)
	}
	content := sb.String()

	return &domain.LLMResponse{
		Content:      content,
		Model:        p.model,
		FinishReason: candidate.FinishReason,
		Usage: domain.LLMUsage{
			PromptTokens:     googleResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: googleResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      googleResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func (p *GoogleProvider) Close() error {
	return nil
}

func (p *GoogleProvider) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &domain.LLMError{
			Provider:   "google",
			StatusCode: statusCode,
			Message:    "authentication failed",
			Err:        domain.ErrLLMAuthFailed,
		}
	case http.StatusTooManyRequests:
		return &domain.LLMError{
			Provider:   "google",
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Err:        domain.ErrLLMRateLimited,
		}
	default:
		return &domain.LLMError{
			Provider:   "google",
			StatusCode: statusCode,
			Message:    string(body),
		}
	}
}
