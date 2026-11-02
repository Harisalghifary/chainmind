package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"chainmind/backend/internal/config"
)

type scriptedClient struct {
	name    string
	results []scriptedResult
	calls   int
}

type scriptedResult struct {
	response *AnalysisResponse
	err      error
}

func (s *scriptedClient) Analyze(context.Context, AnalysisRequest) (*AnalysisResponse, error) {
	index := s.calls
	if index >= len(s.results) {
		index = len(s.results) - 1
	}
	s.calls++

	result := s.results[index]
	return result.response, result.err
}

func (s *scriptedClient) Name() string { return s.name }

func TestRetryClientRetriesTransientFailure(t *testing.T) {
	client := &scriptedClient{
		name: "primary",
		results: []scriptedResult{
			{err: &ProviderError{Provider: "primary", Retryable: true, Err: errors.New("timeout")}},
			{response: &AnalysisResponse{Findings: []LLMFinding{}, Model: "ok"}},
		},
	}

	retry := NewRetryClient(client, 2, time.Millisecond, zerolog.Nop())
	response, err := retry.Analyze(context.Background(), AnalysisRequest{})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if response.Model != "ok" {
		t.Errorf("response = %+v", response)
	}
	if client.calls != 2 {
		t.Errorf("calls = %d, want 2", client.calls)
	}
}

func TestRetryClientDoesNotRetryFatalFailure(t *testing.T) {
	client := &scriptedClient{
		name:    "primary",
		results: []scriptedResult{{err: &ProviderError{Provider: "primary", StatusCode: 401, Err: errors.New("unauthorized")}}},
	}

	retry := NewRetryClient(client, 3, time.Millisecond, zerolog.Nop())
	if _, err := retry.Analyze(context.Background(), AnalysisRequest{}); err == nil {
		t.Fatal("expected an error")
	}
	if client.calls != 1 {
		t.Errorf("calls = %d, want 1", client.calls)
	}
}

func TestRetryClientStopsAfterAttempts(t *testing.T) {
	client := &scriptedClient{
		name: "primary",
		results: []scriptedResult{
			{err: &ProviderError{Provider: "primary", Retryable: true, Err: errors.New("timeout")}},
		},
	}

	retry := NewRetryClient(client, 3, time.Millisecond, zerolog.Nop())
	if _, err := retry.Analyze(context.Background(), AnalysisRequest{}); err == nil {
		t.Fatal("expected an error")
	}
	if client.calls != 3 {
		t.Errorf("calls = %d, want 3", client.calls)
	}
}

func TestFallbackClientUsesSecondaryProvider(t *testing.T) {
	primary := NewRetryClient(&scriptedClient{
		name:    "anthropic",
		results: []scriptedResult{{err: &ProviderError{Provider: "anthropic", StatusCode: 401, Err: errors.New("unauthorized")}}},
	}, 1, time.Millisecond, zerolog.Nop())

	secondaryClient := &scriptedClient{
		name:    "openai",
		results: []scriptedResult{{response: &AnalysisResponse{Model: "gpt", Findings: []LLMFinding{}}}},
	}
	secondary := NewRetryClient(secondaryClient, 1, time.Millisecond, zerolog.Nop())

	chain := NewFallbackClient([]Client{primary, secondary}, zerolog.Nop())
	response, err := chain.Analyze(context.Background(), AnalysisRequest{})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if response.Model != "gpt" || secondaryClient.calls != 1 {
		t.Errorf("response = %+v, secondary calls = %d", response, secondaryClient.calls)
	}
}

func TestFallbackClientAllProvidersFail(t *testing.T) {
	failing := func(name string) Client {
		return NewRetryClient(&scriptedClient{
			name:    name,
			results: []scriptedResult{{err: errors.New("down")}},
		}, 1, time.Millisecond, zerolog.Nop())
	}

	chain := NewFallbackClient([]Client{failing("anthropic"), failing("openai")}, zerolog.Nop())
	if _, err := chain.Analyze(context.Background(), AnalysisRequest{}); err == nil {
		t.Fatal("expected a combined error")
	}
}

func TestNewClientSelection(t *testing.T) {
	log := zerolog.Nop()

	mock, err := NewClient(&config.Config{Env: "development", DefaultLLMProvider: "mock"}, log)
	if err != nil || mock.Name() != "mock" {
		t.Errorf("mock client = %v, %v", mock, err)
	}

	if _, err := NewClient(&config.Config{Env: "production", DefaultLLMProvider: "mock"}, log); err == nil {
		t.Error("mock provider must not be available in production")
	}

	if _, err := NewClient(&config.Config{Env: "development", DefaultLLMProvider: "anthropic"}, log); err == nil {
		t.Error("expected an error without any provider key")
	}

	anthropic, err := NewClient(&config.Config{
		Env:                "development",
		DefaultLLMProvider: "anthropic",
		AnthropicAPIKey:    "test-key",
		AnthropicModel:     "claude-test",
	}, log)
	if err != nil || anthropic.Name() != "anthropic" {
		t.Errorf("anthropic client = %v, %v", anthropic, err)
	}

	both, err := NewClient(&config.Config{
		Env:                "development",
		DefaultLLMProvider: "anthropic",
		AnthropicAPIKey:    "test-key",
		AnthropicModel:     "claude-test",
		OpenAIAPIKey:       "test-key",
		OpenAIModel:        "gpt-test",
	}, log)
	if err != nil || both.Name() != "anthropic->openai" {
		t.Errorf("fallback chain = %v, %v", both, err)
	}

	fallbackOnly, err := NewClient(&config.Config{
		Env:                "development",
		DefaultLLMProvider: "openai",
		AnthropicAPIKey:    "test-key",
		AnthropicModel:     "claude-test",
	}, log)
	if err != nil || fallbackOnly.Name() != "anthropic" {
		t.Errorf("fallback-only client = %v, %v", fallbackOnly, err)
	}
}
