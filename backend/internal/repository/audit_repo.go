package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// Audit statuses as defined by the audit_status database enum.
const (
	StatusPending          = "PENDING"
	StatusParsing          = "PARSING"
	StatusAnalyzing        = "ANALYZING"
	StatusGeneratingReport = "GENERATING_REPORT"
	StatusCompleted        = "COMPLETED"
	StatusFailed           = "FAILED"
)

// AuditJob mirrors a row of the audit_jobs table.
type AuditJob struct {
	ID              string
	ContractName    string
	SourceCode      string
	SolidityVersion *string
	AnalysisDepth   string
	Status          string
	TotalChunks     int
	AnalyzedChunks  int
	ErrorMessage    *string
	LLMModel        *string
	RAGPatternsUsed int
	Warnings        []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
}

// CreateAuditParams holds the fields required to enqueue an audit.
type CreateAuditParams struct {
	ContractName    string
	SourceCode      string
	SolidityVersion *string
	AnalysisDepth   string
}

// AuditRepo persists audit jobs.
type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

const auditColumns = `
	id, contract_name, source_code, solidity_version, analysis_depth, status,
	total_chunks, analyzed_chunks, error_message, llm_model,
	rag_patterns_used, warnings,
	created_at, updated_at, completed_at
`

func scanAuditJob(row pgx.Row) (*AuditJob, error) {
	var job AuditJob
	err := row.Scan(
		&job.ID, &job.ContractName, &job.SourceCode, &job.SolidityVersion,
		&job.AnalysisDepth, &job.Status, &job.TotalChunks, &job.AnalyzedChunks,
		&job.ErrorMessage, &job.LLMModel, &job.RAGPatternsUsed, &job.Warnings,
		&job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan audit job: %w", err)
	}
	return &job, nil
}

// Create inserts a new audit job in PENDING state.
func (r *AuditRepo) Create(ctx context.Context, params CreateAuditParams) (*AuditJob, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO audit_jobs (contract_name, source_code, solidity_version, analysis_depth, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+auditColumns,
		params.ContractName, params.SourceCode, params.SolidityVersion, params.AnalysisDepth, StatusPending,
	)
	return scanAuditJob(row)
}

// GetByID fetches a single audit job.
func (r *AuditRepo) GetByID(ctx context.Context, id string) (*AuditJob, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+auditColumns+` FROM audit_jobs WHERE id = $1`, id)
	return scanAuditJob(row)
}

// UpdateStatus transitions a job to a new status.
func (r *AuditRepo) UpdateStatus(ctx context.Context, id, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE audit_jobs
		SET status = $2::audit_status, updated_at = NOW()
		WHERE id = $1
	`, id, status)
	if err != nil {
		return fmt.Errorf("update audit status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateProgress records chunk progress for a running audit.
func (r *AuditRepo) UpdateProgress(ctx context.Context, id, status string, totalChunks, analyzedChunks int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE audit_jobs
		SET status = $2::audit_status,
		    total_chunks = $3,
		    analyzed_chunks = $4,
		    updated_at = NOW()
		WHERE id = $1
	`, id, status, totalChunks, analyzedChunks)
	if err != nil {
		return fmt.Errorf("update audit progress: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Complete marks a job as COMPLETED, recording the model, retrieved pattern
// count and any degradation warnings.
func (r *AuditRepo) Complete(ctx context.Context, id, llmModel string, ragPatternsUsed int, warnings []string) error {
	if warnings == nil {
		warnings = []string{}
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE audit_jobs
		SET status = $2::audit_status,
		    llm_model = NULLIF($3, ''),
		    rag_patterns_used = $4,
		    warnings = $5,
		    analyzed_chunks = total_chunks,
		    error_message = NULL,
		    updated_at = NOW(),
		    completed_at = NOW()
		WHERE id = $1
	`, id, StatusCompleted, llmModel, ragPatternsUsed, warnings)
	if err != nil {
		return fmt.Errorf("complete audit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Fail marks a job as FAILED and stores the reason.
func (r *AuditRepo) Fail(ctx context.Context, id, message string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE audit_jobs
		SET status = $2::audit_status,
		    error_message = $3,
		    updated_at = NOW(),
		    completed_at = NOW()
		WHERE id = $1
	`, id, StatusFailed, message)
	if err != nil {
		return fmt.Errorf("fail audit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FailStale marks every non-terminal job as FAILED. It is called on startup
// because audits run in-process and cannot survive a restart.
func (r *AuditRepo) FailStale(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE audit_jobs
		SET status = $1::audit_status,
		    error_message = 'audit interrupted by server restart',
		    updated_at = NOW(),
		    completed_at = NOW()
		WHERE status NOT IN ($1::audit_status, $2::audit_status)
	`, StatusFailed, StatusCompleted)
	if err != nil {
		return 0, fmt.Errorf("fail stale audits: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Finding mirrors a row of the findings table. Optional columns are pointers so
// the API can distinguish "not set" from a zero value; MatchedPatternRealWorldCase
// is joined from exploit_patterns when a pattern matched.
type Finding struct {
	ID                          string
	AuditID                     string
	VulnerabilityType           string
	Severity                    string
	Confidence                  float32
	AffectedFunction            string
	LineStart                   *int
	LineEnd                     *int
	Title                       string
	Description                 string
	VulnerableCode              *string
	RecommendedFix              *string
	FixedCode                   *string
	MatchedPatternID            *string
	MatchedPatternName          *string
	SimilarityScore             *float64
	References                  []string
	MatchedPatternRealWorldCase *string
	CreatedAt                   time.Time
}

// ReplaceFindings atomically replaces every finding stored for an audit.
func (r *AuditRepo) ReplaceFindings(ctx context.Context, auditID string, findings []Finding) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin findings transaction: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM findings WHERE audit_id = $1`, auditID); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("clear findings: %w", err)
	}

	for _, finding := range findings {
		id := finding.ID
		if id == "" {
			id = uuid.NewString()
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO findings
				(id, audit_id, vulnerability_type, severity, confidence, affected_function,
				 line_start, line_end, title, description, vulnerable_code, recommended_fix,
				 fixed_code, matched_pattern_id, matched_pattern_name, similarity_score, "references")
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`,
			id, auditID, finding.VulnerabilityType, finding.Severity, finding.Confidence,
			finding.AffectedFunction, finding.LineStart, finding.LineEnd, finding.Title,
			finding.Description, finding.VulnerableCode, finding.RecommendedFix,
			finding.FixedCode, finding.MatchedPatternID, finding.MatchedPatternName,
			finding.SimilarityScore, finding.References,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("insert finding %q: %w", finding.Title, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit findings: %w", err)
	}
	return nil
}

// ListFindings returns every finding for an audit, in insertion order.
func (r *AuditRepo) ListFindings(ctx context.Context, auditID string) ([]Finding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.audit_id, f.vulnerability_type, f.severity, f.confidence,
		       f.affected_function, f.line_start, f.line_end, f.title, f.description,
		       f.vulnerable_code, f.recommended_fix, f.fixed_code,
		       f.matched_pattern_id, f.matched_pattern_name, f.similarity_score,
		       f."references", f.created_at, p.real_world_case
		FROM findings f
		LEFT JOIN exploit_patterns p ON p.id = f.matched_pattern_id
		WHERE f.audit_id = $1
		ORDER BY f.created_at, f.id
	`, auditID)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	findings := []Finding{}
	for rows.Next() {
		var finding Finding
		if err := rows.Scan(
			&finding.ID, &finding.AuditID, &finding.VulnerabilityType, &finding.Severity,
			&finding.Confidence, &finding.AffectedFunction, &finding.LineStart, &finding.LineEnd,
			&finding.Title, &finding.Description, &finding.VulnerableCode, &finding.RecommendedFix,
			&finding.FixedCode, &finding.MatchedPatternID, &finding.MatchedPatternName,
			&finding.SimilarityScore, &finding.References, &finding.CreatedAt,
			&finding.MatchedPatternRealWorldCase,
		); err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		findings = append(findings, finding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings: %w", err)
	}

	return findings, nil
}
