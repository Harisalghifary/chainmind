package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AnthropicClient calls the Anthropic Messages API.
type AnthropicClient struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// NewAnthropicClient builds a Claude client using the Messages API.
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicClient{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    "https://api.anthropic.com",
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *AnthropicClient) Name() string { return "anthropic" }

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *AnthropicClient) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	systemPrompt, userPrompt := BuildPrompt(request)

	payload := anthropicRequest{
		Model:       c.Model,
		MaxTokens:   4096,
		System:      systemPrompt,
		Messages:    []anthropicMessage{{Role: "user", Content: userPrompt}},
		Temperature: 0,
	}

	body, status, err := doJSONRequest(ctx, c.HTTPClient, c.BaseURL+"/v1/messages", map[string]string{
		"x-api-key":         c.APIKey,
		"anthropic-version": "2023-06-01",
	}, payload)
	if err != nil {
		return nil, &ProviderError{Provider: c.Name(), Retryable: true, Err: err}
	}
	if status != http.StatusOK {
		return nil, providerFailure(c.Name(), status, body, nil)
	}

	var response anthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, &ProviderError{Provider: c.Name(), Err: fmt.Errorf("decode response: %w", err)}
	}

	text := ""
	for _, block := range response.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	if text == "" {
		return nil, &ProviderError{Provider: c.Name(), Err: fmt.Errorf("%w: empty content", ErrInvalidResponse)}
	}

	parsed, err := ParseResponse(text)
	if err != nil {
		return nil, &ProviderError{Provider: c.Name(), Err: err}
	}

	if response.Model != "" {
		parsed.Model = response.Model
	} else {
		parsed.Model = c.Model
	}
	parsed.TokensUsed = response.Usage.InputTokens + response.Usage.OutputTokens

	return parsed, nil
}
