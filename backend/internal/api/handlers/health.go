package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"chainmind/backend/internal/config"
)

// Version is the service version reported by the health endpoints.
const Version = "0.1.0"

// DBHealth is the subset of the database pool used by the health check.
type DBHealth interface {
	Ping(ctx context.Context) error
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// HealthHandler reports the status of the API and its dependencies.
type HealthHandler struct {
	cfg *config.Config
	db  DBHealth
}

func NewHealthHandler(cfg *config.Config, db DBHealth) *HealthHandler {
	return &HealthHandler{cfg: cfg, db: db}
}

type serviceStatus struct {
	API      string `json:"api"`
	Database string `json:"database"`
	VectorDB string `json:"vector_db"`
	LLMAPI   string `json:"llm_api"`
}

type healthResponse struct {
	Status   string        `json:"status"`
	Version  string        `json:"version"`
	Services serviceStatus `json:"services"`
}

// Health returns 200 when the API and database are reachable, 503 otherwise.
// A missing LLM key degrades (but does not fail) the service.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	services := serviceStatus{API: "up", Database: "down", VectorDB: "down", LLMAPI: "down"}

	if h.db != nil && h.db.Ping(ctx) == nil {
		services.Database = "up"
		if h.vectorExtensionPresent(ctx) {
			services.VectorDB = "up"
		}
	}

	if h.llmConfigured() {
		services.LLMAPI = "up"
	}

	status := "healthy"
	httpStatus := http.StatusOK
	switch {
	case services.Database != "up":
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	case services.VectorDB != "up" || services.LLMAPI != "up":
		status = "degraded"
	}

	c.JSON(httpStatus, healthResponse{
		Status:   status,
		Version:  Version,
		Services: services,
	})
}

func (h *HealthHandler) vectorExtensionPresent(ctx context.Context) bool {
	var present bool
	err := h.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')`,
	).Scan(&present)
	return err == nil && present
}

func (h *HealthHandler) llmConfigured() bool {
	switch h.cfg.DefaultLLMProvider {
	case "anthropic":
		return h.cfg.AnthropicAPIKey != ""
	case "openai":
		return h.cfg.OpenAIAPIKey != ""
	case "mock":
		return h.cfg.IsDevelopment()
	default:
		return false
	}
}
