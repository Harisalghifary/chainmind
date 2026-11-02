package config

import "testing"

const testDatabaseURL = "postgres://chainmind:chainmind@localhost:5433/chainmind?sslmode=disable"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is missing")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("PORT", "")
	t.Setenv("DEFAULT_LLM_PROVIDER", "")
	t.Setenv("RAG_TOP_K", "")
	t.Setenv("RAG_SIMILARITY_THRESHOLD", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.DefaultLLMProvider != "anthropic" {
		t.Errorf("DefaultLLMProvider = %q, want anthropic", cfg.DefaultLLMProvider)
	}
	if cfg.RAGTopK != 5 {
		t.Errorf("RAGTopK = %d, want 5", cfg.RAGTopK)
	}
	if cfg.RAGSimilarityThreshold != 0.75 {
		t.Errorf("RAGSimilarityThreshold = %v, want 0.75", cfg.RAGSimilarityThreshold)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if cfg.VectorSearchEnabled() {
		t.Error("VectorSearchEnabled() = true without an OpenAI key")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("PORT", "9090")
	t.Setenv("ENV", "production")
	t.Setenv("DEFAULT_LLM_PROVIDER", "OpenAI")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("RAG_TOP_K", "3")
	t.Setenv("RAG_SIMILARITY_THRESHOLD", "0.5")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.test, https://b.test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr() != ":9090" {
		t.Errorf("Addr() = %q, want :9090", cfg.Addr())
	}
	if cfg.IsDevelopment() {
		t.Error("IsDevelopment() = true for ENV=production")
	}
	if cfg.DefaultLLMProvider != "openai" {
		t.Errorf("DefaultLLMProvider = %q, want openai", cfg.DefaultLLMProvider)
	}
	if cfg.RAGTopK != 3 || cfg.RAGSimilarityThreshold != 0.5 {
		t.Errorf("RAG config = %d/%v", cfg.RAGTopK, cfg.RAGSimilarityThreshold)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if !cfg.VectorSearchEnabled() {
		t.Error("VectorSearchEnabled() = false with an OpenAI key")
	}
}

func TestLoadRejectsInvalidThreshold(t *testing.T) {
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("RAG_SIMILARITY_THRESHOLD", "1.5")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for an out-of-range similarity threshold")
	}
}
