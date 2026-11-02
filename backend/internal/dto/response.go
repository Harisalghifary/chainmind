package dto

import (
	"time"
)

// AuditCreatedResponse is the 202 response for a submitted audit.
type AuditCreatedResponse struct {
	AuditID          string    `json:"audit_id"`
	Status           string    `json:"status"`
	EstimatedSeconds int       `json:"estimated_seconds"`
	CreatedAt        time.Time `json:"created_at"`
}

// Progress is the chunk-level progress of an audit.
type Progress struct {
	TotalChunks    int    `json:"total_chunks"`
	AnalyzedChunks int    `json:"analyzed_chunks"`
	CurrentStage   string `json:"current_stage"`
}

// AuditStatusResponse is the GET /api/v1/audit/:id payload.
type AuditStatusResponse struct {
	AuditID      string     `json:"audit_id"`
	Status       string     `json:"status"`
	Progress     Progress   `json:"progress"`
	ErrorMessage *string    `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

// ExploitPatternSummary is one row of GET /api/v1/exploits.
type ExploitPatternSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	SWCID         string `json:"swc_id"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
	RealWorldCase string `json:"real_world_case"`
}

// ExploitListResponse is the GET /api/v1/exploits payload.
type ExploitListResponse struct {
	Patterns []ExploitPatternSummary `json:"patterns"`
	Total    int                     `json:"total"`
	Page     int                     `json:"page"`
	Limit    int                     `json:"limit"`
}

// CurrentStage maps a job status onto the progress stage vocabulary.
func CurrentStage(status string) string {
	switch status {
	case "PENDING":
		return "pending"
	case "PARSING":
		return "parsing"
	case "ANALYZING":
		return "analyzing"
	case "GENERATING_REPORT":
		return "generating_report"
	case "COMPLETED":
		return "completed"
	case "FAILED":
		return "failed"
	default:
		return "unknown"
	}
}

// EstimatedSecondsForDepth returns the rough duration shown after submission.
func EstimatedSecondsForDepth(depth string) int {
	switch depth {
	case "quick":
		return 15
	case "deep":
		return 45
	default:
		return 30
	}
}
