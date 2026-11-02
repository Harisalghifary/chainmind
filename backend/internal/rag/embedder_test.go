package rag

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"chainmind/backend/internal/config"
)

func cosine(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func TestMockEmbedderRanksRelatedTextHigher(t *testing.T) {
	embedder := NewMockEmbedder()
	ctx := context.Background()

	query, err := embedder.Embed(ctx, `function withdraw() external {
		uint256 amount = balances[msg.sender];
		(bool success, ) = msg.sender.call{value: amount}("");
		require(success);
		balances[msg.sender] = 0;
	}`)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	related, err := embedder.Embed(ctx, "Reentrancy via external call before state update. The withdraw function calls msg.sender before resetting balances.")
	if err != nil {
		t.Fatalf("Embed(related) error = %v", err)
	}

	unrelated, err := embedder.Embed(ctx, "Missing owner check allows any caller to mint unlimited tokens or destroy the contract.")
	if err != nil {
		t.Fatalf("Embed(unrelated) error = %v", err)
	}

	if len(query) != EmbeddingDimensions {
		t.Fatalf("dimensions = %d", len(query))
	}

	relatedScore := cosine(query, related)
	unrelatedScore := cosine(query, unrelated)
	if relatedScore <= unrelatedScore {
		t.Errorf("related score %.3f <= unrelated score %.3f", relatedScore, unrelatedScore)
	}
}

func TestMockEmbedderRejectsEmptyText(t *testing.T) {
	if _, err := NewMockEmbedder().Embed(context.Background(), "!!! ***"); err == nil {
		t.Fatal("expected an error for text without tokens")
	}
}

func TestOpenAIEmbedderSuccess(t *testing.T) {
	embedding := make([]float32, EmbeddingDimensions)
	embedding[0] = 1

	var gotModel, gotInput, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")

		var body struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotModel = body.Model
		gotInput = body.Input

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"embedding": embedding}},
		})
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small")
	embedder.BaseURL = server.URL

	result, err := embedder.Embed(context.Background(), "function withdraw() external {}")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(result) != EmbeddingDimensions || result[0] != 1 {
		t.Errorf("embedding = %v...", result[:2])
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotModel != "text-embedding-3-small" {
		t.Errorf("model = %q", gotModel)
	}
	if gotInput != "function withdraw() external {}" {
		t.Errorf("input = %q", gotInput)
	}
}

func TestOpenAIEmbedderMarksServerErrorsRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small")
	embedder.BaseURL = server.URL

	_, err := embedder.Embed(context.Background(), "solidity")
	var embeddingErr *EmbeddingError
	if !errors.As(err, &embeddingErr) {
		t.Fatalf("error = %v, want *EmbeddingError", err)
	}
	if !embeddingErr.Retryable || embeddingErr.StatusCode != http.StatusBadGateway {
		t.Errorf("error = %+v", embeddingErr)
	}
}

func TestOpenAIEmbedderRejectsUnexpectedDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"embedding": []float32{1, 2, 3}}},
		})
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small")
	embedder.BaseURL = server.URL

	if _, err := embedder.Embed(context.Background(), "solidity"); err == nil {
		t.Fatal("expected a dimension error")
	}
}

func TestNewEmbedderSelection(t *testing.T) {
	if _, err := NewEmbedder(&config.Config{Env: "development", DefaultLLMProvider: "anthropic"}); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("error = %v, want ErrNotConfigured", err)
	}

	mock, err := NewEmbedder(&config.Config{Env: "development", DefaultLLMProvider: "mock"})
	if err != nil || mock.Name() != "mock-embedding" {
		t.Errorf("mock embedder = %v, %v", mock, err)
	}

	if _, err := NewEmbedder(&config.Config{Env: "production", DefaultLLMProvider: "mock"}); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("production mock error = %v, want ErrNotConfigured", err)
	}

	openai, err := NewEmbedder(&config.Config{
		Env:                "development",
		DefaultLLMProvider: "anthropic",
		OpenAIAPIKey:       "sk-test",
		EmbeddingModel:     "text-embedding-3-small",
	})
	if err != nil || openai.Name() != "text-embedding-3-small" {
		t.Errorf("openai embedder = %v, %v", openai, err)
	}
}
