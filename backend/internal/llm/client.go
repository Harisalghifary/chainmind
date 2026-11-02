package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"chainmind/backend/internal/config"
)

// Client analyzes a single code chunk with one provider.
type Client interface {
	Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error)
	Name() string
}

// ProviderError describes a provider failure and whether a retry can help.
type ProviderError struct {
	Provider   string
	StatusCode int
	Retryable  bool
	Err        error
}

func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s provider error (status %d): %v", e.Provider, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s provider error: %v", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// IsRetryable reports whether the error is transient (timeout, 429, 5xx).
func IsRetryable(err error) bool {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Retryable
	}
	return false
}

// RetryClient retries transient provider failures with linear backoff.
type RetryClient struct {
	client    Client
	attempts  int
	baseDelay time.Duration
	log       zerolog.Logger
}

func NewRetryClient(client Client, attempts int, baseDelay time.Duration, log zerolog.Logger) *RetryClient {
	if attempts < 1 {
		attempts = 1
	}
	return &RetryClient{client: client, attempts: attempts, baseDelay: baseDelay, log: log}
}

func (r *RetryClient) Name() string { return r.client.Name() }

func (r *RetryClient) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= r.attempts; attempt++ {
		response, err := r.client.Analyze(ctx, request)
		if err == nil {
			return response, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !IsRetryable(err) || attempt == r.attempts {
			break
		}

		r.log.Warn().
			Err(err).
			Str("llm_provider", r.client.Name()).
			Int("attempt", attempt).
			Msg("llm_request_retry")

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(r.baseDelay * time.Duration(attempt)):
		}
	}

	return nil, lastErr
}

// FallbackClient tries providers in order until one succeeds.
type FallbackClient struct {
	clients []Client
	log     zerolog.Logger
}

func NewFallbackClient(clients []Client, log zerolog.Logger) *FallbackClient {
	return &FallbackClient{clients: clients, log: log}
}

func (f *FallbackClient) Name() string {
	names := make([]string, 0, len(f.clients))
	for _, client := range f.clients {
		names = append(names, client.Name())
	}
	return strings.Join(names, "->")
}

func (f *FallbackClient) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	if len(f.clients) == 0 {
		return nil, errors.New("no llm providers configured")
	}

	errs := make([]error, 0, len(f.clients))
	for index, client := range f.clients {
		response, err := client.Analyze(ctx, request)
		if err == nil {
			return response, nil
		}

		errs = append(errs, fmt.Errorf("%s: %w", client.Name(), err))
		if index < len(f.clients)-1 {
			f.log.Warn().
				Err(err).
				Str("llm_provider", client.Name()).
				Msg("llm_provider_fallback")
		}
	}

	return nil, fmt.Errorf("all llm providers failed: %w", errors.Join(errs...))
}

// NewClient builds the configured provider chain. Mock mode is only available
// in development and never falls back to a paid provider.
func NewClient(cfg *config.Config, log zerolog.Logger) (Client, error) {
	if cfg.UseMockProviders() {
		return NewMockClient(), nil
	}

	providers := []string{cfg.DefaultLLMProvider}
	for _, candidate := range []string{"anthropic", "openai"} {
		if candidate != cfg.DefaultLLMProvider && providerConfigured(cfg, candidate) {
			providers = append(providers, candidate)
		}
	}

	clients := make([]Client, 0, len(providers))
	for index, provider := range providers {
		client, err := buildClient(provider, cfg)
		if err != nil {
			if index == 0 {
				log.Warn().Err(err).Str("llm_provider", provider).Msg("llm_primary_unavailable")
				continue
			}
			log.Warn().Err(err).Str("llm_provider", provider).Msg("llm_fallback_unavailable")
			continue
		}
		clients = append(clients, NewRetryClient(client, 2, 500*time.Millisecond, log))
	}

	if len(clients) == 0 {
		return nil, errors.New("no llm provider is configured; set ANTHROPIC_API_KEY or OPENAI_API_KEY")
	}

	return NewFallbackClient(clients, log), nil
}

func providerConfigured(cfg *config.Config, provider string) bool {
	switch provider {
	case "anthropic":
		return cfg.AnthropicAPIKey != ""
	case "openai":
		return cfg.OpenAIAPIKey != ""
	default:
		return false
	}
}

func buildClient(provider string, cfg *config.Config) (Client, error) {
	switch provider {
	case "anthropic":
		if cfg.AnthropicAPIKey == "" {
			return nil, errors.New("ANTHROPIC_API_KEY is not configured")
		}
		return NewAnthropicClient(cfg.AnthropicAPIKey, cfg.AnthropicModel), nil
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return nil, errors.New("OPENAI_API_KEY is not configured")
		}
		return NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", provider)
	}
}

// doJSONRequest posts a JSON payload and returns the raw response body. A
// non-2xx status is returned to the caller for provider-specific handling.
func doJSONRequest(ctx context.Context, httpClient *http.Client, url string, headers map[string]string, payload any) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("encode request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, response.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return responseBody, response.StatusCode, nil
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func providerFailure(provider string, status int, body []byte, err error) *ProviderError {
	message := strings.TrimSpace(string(body))
	if len(message) > 500 {
		message = message[:500]
	}
	if message == "" && err != nil {
		message = err.Error()
	}
	return &ProviderError{
		Provider:   provider,
		StatusCode: status,
		Retryable:  retryableStatus(status),
		Err:        errors.New(message),
	}
}
