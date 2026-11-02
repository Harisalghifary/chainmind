package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"chainmind/backend/internal/api/handlers"
	"chainmind/backend/internal/config"
	"chainmind/backend/internal/orchestrator"
	"chainmind/backend/internal/repository"
)

// NewRouter wires middleware, handlers and routes for the API.
func NewRouter(
	cfg *config.Config,
	pool *pgxpool.Pool,
	auditRepo *repository.AuditRepo,
	exploitRepo *repository.ExploitRepo,
	worker *orchestrator.Orchestrator,
	log zerolog.Logger,
) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(RequestLogger(log))
	router.Use(Recovery(log))
	router.Use(CORS(cfg.CORSAllowedOrigins))
	router.Use(ErrorHandler(log))

	healthHandler := handlers.NewHealthHandler(cfg, pool)
	auditHandler := handlers.NewAuditHandler(worker, auditRepo)
	reportHandler := handlers.NewReportHandler(auditRepo)
	exploitsHandler := handlers.NewExploitsHandler(exploitRepo)

	router.GET("/health", healthHandler.Health)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.Health)
		v1.POST("/audit", auditHandler.Submit)
		v1.GET("/audit/:id", auditHandler.Get)
		v1.GET("/audit/:id/report", reportHandler.Get)
		v1.GET("/exploits", exploitsHandler.List)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NOT_FOUND",
			"message": "route not found",
		})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":   "METHOD_NOT_ALLOWED",
			"message": "method not allowed",
		})
	})

	return router
}
