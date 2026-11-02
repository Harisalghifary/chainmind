package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"chainmind/backend/internal/dto"
	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/repository"
	apperrors "chainmind/backend/pkg/errors"
	"chainmind/backend/pkg/validator"
)

// AuditSubmitter enqueues an audit for background processing.
type AuditSubmitter interface {
	Submit(ctx context.Context, params repository.CreateAuditParams) (*repository.AuditJob, error)
}

// AuditReader reads persisted audits and findings.
type AuditReader interface {
	GetByID(ctx context.Context, id string) (*repository.AuditJob, error)
	ListFindings(ctx context.Context, auditID string) ([]repository.Finding, error)
}

// AuditHandler serves audit submission and status requests.
type AuditHandler struct {
	submitter AuditSubmitter
	reader    AuditReader
}

func NewAuditHandler(submitter AuditSubmitter, reader AuditReader) *AuditHandler {
	return &AuditHandler{submitter: submitter, reader: reader}
}

// Submit handles POST /api/v1/audit.
func (h *AuditHandler) Submit(c *gin.Context) {
	var request dto.CreateAuditRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Error(apperrors.InvalidInput("request body must be valid JSON"))
		return
	}

	request.Normalize()
	if err := request.Validate(); err != nil {
		c.Error(err)
		return
	}

	// Reject clearly unparseable source synchronously so the client gets the
	// documented 422 instead of a job that fails immediately in the worker.
	if err := validator.ValidateSource(request.SourceCode); err != nil {
		c.Error(apperrors.ParseError("Unable to parse Solidity contract"))
		return
	}
	if _, err := parser.Parse(request.SourceCode); err != nil {
		c.Error(apperrors.ParseError("Unable to parse Solidity contract"))
		return
	}

	var solidityVersion *string
	if request.SolidityVersion != "" {
		version := request.SolidityVersion
		solidityVersion = &version
	}

	job, err := h.submitter.Submit(c.Request.Context(), repository.CreateAuditParams{
		ContractName:    request.ContractName,
		SourceCode:      request.SourceCode,
		SolidityVersion: solidityVersion,
		AnalysisDepth:   request.AnalysisDepth,
	})
	if err != nil {
		c.Error(apperrors.Wrap(err, "SERVICE_UNAVAILABLE", "audit service is unavailable", http.StatusServiceUnavailable))
		return
	}

	c.JSON(http.StatusAccepted, dto.AuditCreatedResponse{
		AuditID:          job.ID,
		Status:           job.Status,
		EstimatedSeconds: dto.EstimatedSecondsForDepth(job.AnalysisDepth),
		CreatedAt:        job.CreatedAt,
	})
}

// Get handles GET /api/v1/audit/:id.
func (h *AuditHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.Error(apperrors.AuditNotFound())
		return
	}

	job, err := h.reader.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.Error(apperrors.AuditNotFound())
			return
		}
		c.Error(apperrors.Wrap(err, "SERVICE_UNAVAILABLE", "audit service is unavailable", http.StatusServiceUnavailable))
		return
	}

	c.JSON(http.StatusOK, dto.AuditStatusResponse{
		AuditID: job.ID,
		Status:  job.Status,
		Progress: dto.Progress{
			TotalChunks:    job.TotalChunks,
			AnalyzedChunks: job.AnalyzedChunks,
			CurrentStage:   dto.CurrentStage(job.Status),
		},
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
		CompletedAt:  job.CompletedAt,
	})
}
