package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OpenAIClient calls the OpenAI Chat Completions API in JSON mode.
type OpenAIClient struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// NewOpenAIClient builds a GPT client.
func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIClient{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    "https://api.openai.com",
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAIClient) Name() string { return "openai" }

type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *OpenAIClient) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	systemPrompt, userPrompt := BuildPrompt(request)

	payload := openAIRequest{
		Model:       c.Model,
		Temperature: 0,
		MaxTokens:   4096,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	payload.ResponseFormat.Type = "json_object"

	body, status, err := doJSONRequest(ctx, c.HTTPClient, c.BaseURL+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + c.APIKey,
	}, payload)
	if err != nil {
		return nil, &ProviderError{Provider: c.Name(), Retryable: true, Err: err}
	}
	if status != http.StatusOK {
		return nil, providerFailure(c.Name(), status, body, nil)
	}

	var response openAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, &ProviderError{Provider: c.Name(), Err: fmt.Errorf("decode response: %w", err)}
	}
	if len(response.Choices) == 0 {
		return nil, &ProviderError{Provider: c.Name(), Err: fmt.Errorf("%w: no choices", ErrInvalidResponse)}
	}

	parsed, err := ParseResponse(response.Choices[0].Message.Content)
	if err != nil {
		return nil, &ProviderError{Provider: c.Name(), Err: err}
	}

	if response.Model != "" {
		parsed.Model = response.Model
	} else {
		parsed.Model = c.Model
	}
	parsed.TokensUsed = response.Usage.TotalTokens

	return parsed, nil
}
