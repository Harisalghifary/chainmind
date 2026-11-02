package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds every runtime setting. Secrets are only ever sourced from the
// environment (optionally hydrated from a local, git-ignored .env file).
type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	CORSAllowedOrigins []string

	DefaultLLMProvider string
	AnthropicAPIKey    string
	AnthropicModel     string
	OpenAIAPIKey       string
	OpenAIModel        string
	EmbeddingModel     string

	RAGTopK                int
	RAGSimilarityThreshold float64
}

// Load reads configuration from environment variables, falling back to the
// local .env file when one exists. It returns an error when a required value
// is missing.
func Load() (*Config, error) {
	loadDotEnv()

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("port", "8080")
	v.SetDefault("env", "development")
	v.SetDefault("cors_allowed_origins", "http://localhost:3000")
	v.SetDefault("default_llm_provider", "anthropic")
	v.SetDefault("anthropic_model", "claude-sonnet-4-20250514")
	v.SetDefault("openai_model", "gpt-4o")
	v.SetDefault("embedding_model", "text-embedding-3-small")
	v.SetDefault("rag_top_k", 5)
	v.SetDefault("rag_similarity_threshold", 0.75)

	cfg := &Config{
		Port:                   v.GetString("port"),
		Env:                    v.GetString("env"),
		DatabaseURL:            v.GetString("database_url"),
		CORSAllowedOrigins:     splitCSV(v.GetString("cors_allowed_origins")),
		DefaultLLMProvider:     strings.ToLower(v.GetString("default_llm_provider")),
		AnthropicAPIKey:        v.GetString("anthropic_api_key"),
		AnthropicModel:         v.GetString("anthropic_model"),
		OpenAIAPIKey:           v.GetString("openai_api_key"),
		OpenAIModel:            v.GetString("openai_model"),
		EmbeddingModel:         v.GetString("embedding_model"),
		RAGTopK:                v.GetInt("rag_top_k"),
		RAGSimilarityThreshold: v.GetFloat64("rag_similarity_threshold"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.Port == "" {
		return nil, errors.New("PORT must not be empty")
	}
	if cfg.RAGTopK <= 0 {
		cfg.RAGTopK = 5
	}
	if cfg.RAGSimilarityThreshold < 0 || cfg.RAGSimilarityThreshold > 1 {
		return nil, fmt.Errorf("RAG_SIMILARITY_THRESHOLD must be between 0 and 1, got %v", cfg.RAGSimilarityThreshold)
	}
	if cfg.DefaultLLMProvider == "" {
		cfg.DefaultLLMProvider = "anthropic"
	}

	return cfg, nil
}

// Addr returns the listen address for the HTTP server.
func (c *Config) Addr() string {
	return ":" + c.Port
}

// IsDevelopment reports whether the server runs in a non-production mode.
func (c *Config) IsDevelopment() bool {
	return c.Env != "production"
}

// UseMockProviders reports whether keyless development providers are enabled.
// Mock mode requires both a non-production environment and an explicit opt-in.
func (c *Config) UseMockProviders() bool {
	return c.IsDevelopment() && c.DefaultLLMProvider == "mock"
}

// VectorSearchEnabled reports whether embedding generation is configured.
func (c *Config) VectorSearchEnabled() bool {
	return c.OpenAIAPIKey != ""
}

func loadDotEnv() {
	// Best effort: a missing .env is a valid deployment configuration.
	if err := godotenv.Load(".env"); err == nil {
		return
	}
	_ = godotenv.Load("../.env")
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
