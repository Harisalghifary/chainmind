package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"chainmind/backend/internal/config"
)

// EmbeddingDimensions matches the vector(1536) column used by pgvector.
const EmbeddingDimensions = 1536

// ErrNotConfigured is returned when no embedding provider can be built.
var ErrNotConfigured = errors.New("embedding provider is not configured")

// Embedder turns text into a fixed-size embedding vector.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Name() string
}

// EmbeddingError describes a failed embedding request.
type EmbeddingError struct {
	StatusCode int
	Retryable  bool
	Message    string
	Err        error
}

func (e *EmbeddingError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("embedding failed: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("embedding failed: %s", e.Message)
}

func (e *EmbeddingError) Unwrap() error { return e.Err }

// OpenAIEmbedder calls the OpenAI embeddings API.
type OpenAIEmbedder struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// NewOpenAIEmbedder builds an embedder for text-embedding-3-small (or another
// 1536-dimension OpenAI model).
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbedder{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    "https://api.openai.com",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *OpenAIEmbedder) Name() string { return e.Model }

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, &EmbeddingError{Message: "input text is empty"}
	}

	payload, err := json.Marshal(map[string]any{
		"model": e.Model,
		"input": text,
	})
	if err != nil {
		return nil, &EmbeddingError{Message: "encode request", Err: err}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/v1/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, &EmbeddingError{Message: "build request", Err: err}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+e.APIKey)

	response, err := e.HTTPClient.Do(request)
	if err != nil {
		return nil, &EmbeddingError{Message: "call embedding api", Retryable: true, Err: err}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return nil, &EmbeddingError{
			StatusCode: response.StatusCode,
			Retryable:  response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500,
			Message:    fmt.Sprintf("embedding api returned %d: %s", response.StatusCode, strings.TrimSpace(string(body))),
		}
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, &EmbeddingError{Message: "decode embedding api response", Err: err}
	}
	if len(result.Data) == 0 {
		return nil, &EmbeddingError{Message: "embedding api returned no data"}
	}
	if len(result.Data[0].Embedding) != EmbeddingDimensions {
		return nil, &EmbeddingError{
			Message: fmt.Sprintf("embedding has %d dimensions, want %d", len(result.Data[0].Embedding), EmbeddingDimensions),
		}
	}

	return result.Data[0].Embedding, nil
}

// MockEmbedder produces deterministic lexical embeddings. It exists so the
// full pipeline can be exercised in development without an OpenAI key; it is
// never selected in production.
type MockEmbedder struct {
	Dimensions int
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{Dimensions: EmbeddingDimensions}
}

func (m *MockEmbedder) Name() string { return "mock-embedding" }

func (m *MockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	dimensions := m.Dimensions
	if dimensions <= 0 {
		dimensions = EmbeddingDimensions
	}

	vector := make([]float32, dimensions)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return nil, &EmbeddingError{Message: "text has no embeddable tokens"}
	}

	for _, token := range tokens {
		vector[hashToken(token)%uint32(dimensions)] += 1
	}

	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}
	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] = float32(float64(vector[i]) / norm)
	}

	return vector, nil
}

var identifierRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// tokenize lowercases identifiers and splits camelCase / snake_case into
// searchable sub-tokens, which makes lexical similarity behave sensibly on
// Solidity source and prose.
func tokenize(text string) []string {
	tokens := []string{}
	for _, identifier := range identifierRe.FindAllString(text, -1) {
		for _, part := range splitIdentifier(identifier) {
			if len(part) >= 3 {
				tokens = append(tokens, strings.ToLower(part))
			}
		}
	}
	return tokens
}

func splitIdentifier(identifier string) []string {
	parts := []string{}
	current := []rune{}

	for index, char := range identifier {
		if char == '_' {
			if len(current) > 0 {
				parts = append(parts, string(current))
				current = nil
			}
			continue
		}
		if index > 0 && char >= 'A' && char <= 'Z' && len(current) > 0 {
			parts = append(parts, string(current))
			current = nil
		}
		current = append(current, char)
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}

	return parts
}

func hashToken(token string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(token); i++ {
		hash ^= uint32(token[i])
		hash *= prime32
	}
	return hash
}

// NewEmbedder selects the OpenAI embedder when a key exists, falling back to
// the mock embedder only in explicit development mock mode.
func NewEmbedder(cfg *config.Config) (Embedder, error) {
	if cfg.OpenAIAPIKey != "" {
		return NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.EmbeddingModel), nil
	}
	if cfg.UseMockProviders() {
		return NewMockEmbedder(), nil
	}
	return nil, ErrNotConfigured
}
