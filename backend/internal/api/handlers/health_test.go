package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"chainmind/backend/internal/config"
)

type fakeDB struct {
	pingErr   error
	hasVector bool
	vectorErr error
}

func (f *fakeDB) Ping(context.Context) error { return f.pingErr }

func (f *fakeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{value: f.hasVector, err: f.vectorErr}
}

type fakeRow struct {
	value bool
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	target, ok := dest[0].(*bool)
	if !ok {
		return errors.New("unexpected scan destination")
	}
	*target = r.value
	return nil
}

func doHealth(t *testing.T, cfg *config.Config, db DBHealth) (*httptest.ResponseRecorder, healthResponse) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", NewHealthHandler(cfg, db).Health)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(recorder, request)

	var body healthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	return recorder, body
}

func TestHealthHealthy(t *testing.T) {
	cfg := &config.Config{
		Env:                "development",
		DefaultLLMProvider: "anthropic",
		AnthropicAPIKey:    "test-key",
	}

	recorder, body := doHealth(t, cfg, &fakeDB{hasVector: true})

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
	if body.Status != "healthy" || body.Version != Version {
		t.Errorf("body = %+v", body)
	}
	if body.Services.Database != "up" || body.Services.VectorDB != "up" || body.Services.LLMAPI != "up" {
		t.Errorf("services = %+v", body.Services)
	}
}

func TestHealthDegradedWithoutLLMKey(t *testing.T) {
	cfg := &config.Config{Env: "development", DefaultLLMProvider: "anthropic"}

	recorder, body := doHealth(t, cfg, &fakeDB{hasVector: true})

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
	if body.Status != "degraded" {
		t.Errorf("Status = %q, want degraded", body.Status)
	}
	if body.Services.LLMAPI != "down" {
		t.Errorf("LLMAPI = %q, want down", body.Services.LLMAPI)
	}
}

func TestHealthDegradedWithoutVectorExtension(t *testing.T) {
	cfg := &config.Config{
		Env:                "development",
		DefaultLLMProvider: "openai",
		OpenAIAPIKey:       "test-key",
	}

	_, body := doHealth(t, cfg, &fakeDB{hasVector: false})

	if body.Status != "degraded" || body.Services.VectorDB != "down" {
		t.Errorf("body = %+v", body)
	}
}

func TestHealthUnhealthyWhenDatabaseIsDown(t *testing.T) {
	cfg := &config.Config{Env: "development", DefaultLLMProvider: "anthropic"}

	recorder, body := doHealth(t, cfg, &fakeDB{pingErr: errors.New("connection refused")})

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", recorder.Code)
	}
	if body.Status != "unhealthy" || body.Services.Database != "down" {
		t.Errorf("body = %+v", body)
	}
}

func TestHealthMockProviderOnlyInDevelopment(t *testing.T) {
	devCfg := &config.Config{Env: "development", DefaultLLMProvider: "mock"}
	if _, body := doHealth(t, devCfg, &fakeDB{hasVector: true}); body.Services.LLMAPI != "up" {
		t.Errorf("mock provider in development: LLMAPI = %q, want up", body.Services.LLMAPI)
	}

	prodCfg := &config.Config{Env: "production", DefaultLLMProvider: "mock"}
	if _, body := doHealth(t, prodCfg, &fakeDB{hasVector: true}); body.Services.LLMAPI != "down" {
		t.Errorf("mock provider in production: LLMAPI = %q, want down", body.Services.LLMAPI)
	}
}
