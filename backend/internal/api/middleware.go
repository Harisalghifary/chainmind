package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	apperrors "chainmind/backend/pkg/errors"
)

// RequestIDHeader is returned on every response for correlation.
const RequestIDHeader = "X-Request-ID"

// RequestLogger emits one structured log line per request.
func RequestLogger(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()

		event := log.Info()
		if c.Writer.Status() >= http.StatusInternalServerError {
			event = log.Error()
		} else if c.Writer.Status() >= http.StatusBadRequest {
			event = log.Warn()
		}

		event.
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", c.FullPath()).
			Int("status", c.Writer.Status()).
			Int64("duration_ms", time.Since(start).Milliseconds()).
			Msg("http_request")
	}
}

// CORS allows the configured frontend origins to call the API.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, "+RequestIDHeader)
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Recovery converts panics into structured JSON errors.
func Recovery(log zerolog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Error().
			Interface("panic", recovered).
			Str("request_id", c.GetString("request_id")).
			Str("path", c.Request.URL.Path).
			Msg("panic_recovered")

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": "internal server error",
		})
	})
}

// ErrorHandler renders errors attached with c.Error as structured JSON.
func ErrorHandler(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		appErr := apperrors.As(c.Errors.Last().Err)
		if appErr.HTTPStatus >= http.StatusInternalServerError {
			log.Error().
				Str("request_id", c.GetString("request_id")).
				Str("error", appErr.Code).
				Err(appErr.Err).
				Msg("request_failed")
		}

		c.AbortWithStatusJSON(appErr.HTTPStatus, gin.H{
			"error":   appErr.Code,
			"message": appErr.Message,
		})
	}
}

// NormalizePath trims trailing slashes so /api/v1/audit/ maps to /api/v1/audit.
func NormalizePath(path string) string {
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		return strings.TrimSuffix(path, "/")
	}
	return path
}
