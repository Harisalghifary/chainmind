package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const validAnalysisJSON = `{"findings":[{"vulnerability_type":"reentrancy","severity":"critical","confidence":0.95,"affected_function":"withdraw","line_start":14,"line_end":17,"title":"Reentrancy in withdraw()","description":"External call before state update.","vulnerable_code":"msg.sender.call{value: amount}(\"\");","recommended_fix":"Reset first.","fixed_code":"balances[msg.sender] = 0;","matched_exploit_pattern":"Reentrancy via external call before state update"}]}`

func TestAnthropicClientAnalyze(t *testing.T) {
	var gotModel, gotSystem string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}

		var body struct {
			Model    string `json:"model"`
			System   string `json:"system"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotModel = body.Model
		gotSystem = body.System
		if len(body.Messages) != 1 {
			t.Errorf("messages = %d", len(body.Messages))
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "claude-response-model",
			"content": []map[string]any{{"type": "text", "text": validAnalysisJSON}},
			"usage":   map[string]int{"input_tokens": 100, "output_tokens": 50},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key", "claude-test-model")
	client.BaseURL = server.URL

	response, err := client.Analyze(context.Background(), AnalysisRequest{})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if gotModel != "claude-test-model" {
		t.Errorf("request model = %q", gotModel)
	}
	if gotSystem != SystemPrompt {
		t.Error("request did not use the auditor system prompt")
	}
	if response.Model != "claude-response-model" || response.TokensUsed != 150 {
		t.Errorf("response metadata = %s/%d", response.Model, response.TokensUsed)
	}
	if len(response.Findings) != 1 || response.Findings[0].AffectedFunction != "withdraw" {
		t.Errorf("findings = %+v", response.Findings)
	}
}

func TestAnthropicClientRetryableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "overloaded", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key", "claude-test-model")
	client.BaseURL = server.URL

	_, err := client.Analyze(context.Background(), AnalysisRequest{})
	if !IsRetryable(err) {
		t.Fatalf("error = %v, want retryable", err)
	}
}

func TestAnthropicClientInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "I could not analyze this."}},
		})
	}))
	defer server.Close()

	client := NewAnthropicClient("test-key", "claude-test-model")
	client.BaseURL = server.URL

	_, err := client.Analyze(context.Background(), AnalysisRequest{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if IsRetryable(err) {
		t.Error("invalid JSON should not be marked retryable")
	}
	if !errors.Is(err, ErrInvalidResponse) {
		t.Errorf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestOpenAIClientAnalyze(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}

		var body struct {
			Model          string `json:"model"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.ResponseFormat.Type != "json_object" {
			t.Errorf("response_format = %q", body.ResponseFormat.Type)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "gpt-response-model",
			"choices": []map[string]any{{
				"message": map[string]string{"content": validAnalysisJSON},
			}},
			"usage": map[string]int{"total_tokens": 200},
		})
	}))
	defer server.Close()

	client := NewOpenAIClient("test-key", "gpt-test-model")
	client.BaseURL = server.URL

	response, err := client.Analyze(context.Background(), AnalysisRequest{})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if response.Model != "gpt-response-model" || response.TokensUsed != 200 {
		t.Errorf("response metadata = %s/%d", response.Model, response.TokensUsed)
	}
	if len(response.Findings) != 1 {
		t.Errorf("findings = %+v", response.Findings)
	}
}

func TestOpenAIClientRateLimitIsRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewOpenAIClient("test-key", "gpt-test-model")
	client.BaseURL = server.URL

	_, err := client.Analyze(context.Background(), AnalysisRequest{})
	if !IsRetryable(err) {
		t.Fatalf("error = %v, want retryable", err)
	}
}
