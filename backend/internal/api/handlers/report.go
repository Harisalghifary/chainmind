package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"chainmind/backend/internal/report"
	"chainmind/backend/internal/repository"
	apperrors "chainmind/backend/pkg/errors"
)

// ReportHandler serves completed audit reports.
type ReportHandler struct {
	reader AuditReader
}

func NewReportHandler(reader AuditReader) *ReportHandler {
	return &ReportHandler{reader: reader}
}

// Get handles GET /api/v1/audit/:id/report.
func (h *ReportHandler) Get(c *gin.Context) {
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

	if job.Status != repository.StatusCompleted {
		c.Error(apperrors.AuditNotReady())
		return
	}

	records, err := h.reader.ListFindings(c.Request.Context(), id)
	if err != nil {
		c.Error(apperrors.Wrap(err, "SERVICE_UNAVAILABLE", "audit service is unavailable", http.StatusServiceUnavailable))
		return
	}

	findings := make([]report.Finding, 0, len(records))
	for _, record := range records {
		findings = append(findings, findingFromRecord(record))
	}

	var processingTimeMS int64
	if job.CompletedAt != nil {
		processingTimeMS = job.CompletedAt.Sub(job.CreatedAt).Milliseconds()
		if processingTimeMS < 0 {
			processingTimeMS = 0
		}
	}

	model := ""
	if job.LLMModel != nil {
		model = *job.LLMModel
	}

	c.JSON(http.StatusOK, report.Generate(report.GenerateInput{
		AuditID:          job.ID,
		ContractName:     job.ContractName,
		LLMModel:         model,
		AnalysisDepth:    job.AnalysisDepth,
		RAGPatternsUsed:  job.RAGPatternsUsed,
		ProcessingTimeMS: processingTimeMS,
		Warnings:         job.Warnings,
		Findings:         findings,
	}))
}

func findingFromRecord(record repository.Finding) report.Finding {
	finding := report.Finding{
		ID:                record.ID,
		VulnerabilityType: record.VulnerabilityType,
		Severity:          record.Severity,
		Confidence:        record.Confidence,
		AffectedFunction:  record.AffectedFunction,
		Title:             record.Title,
		Description:       record.Description,
		VulnerableCode:    derefString(record.VulnerableCode),
		RecommendedFix:    derefString(record.RecommendedFix),
		FixedCode:         derefString(record.FixedCode),
		References:        record.References,
		MatchedPatternID:  derefString(record.MatchedPatternID),
	}
	if record.LineStart != nil {
		finding.LineStart = *record.LineStart
	}
	if record.LineEnd != nil {
		finding.LineEnd = *record.LineEnd
	}
	if finding.References == nil {
		finding.References = []string{}
	}

	if record.MatchedPatternName != nil {
		similarity := float32(0)
		if record.SimilarityScore != nil {
			similarity = float32(*record.SimilarityScore)
		}
		finding.MatchedPattern = &report.MatchedPattern{
			Name:            *record.MatchedPatternName,
			RealWorldCase:   derefString(record.MatchedPatternRealWorldCase),
			SimilarityScore: similarity,
		}
	}

	return finding
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
