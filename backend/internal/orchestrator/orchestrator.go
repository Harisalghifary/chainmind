package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"chainmind/backend/internal/llm"
	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/rag"
	"chainmind/backend/internal/report"
	"chainmind/backend/internal/repository"
)

// Orchestrator runs audit jobs asynchronously and drives them through the
// PENDING -> PARSING -> ANALYZING -> GENERATING_REPORT -> COMPLETED lifecycle.
type Orchestrator struct {
	store        AuditStore
	retriever    PatternRetriever
	llmClient    Auditor
	log          zerolog.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	sem          chan struct{}
	chunkTimeout time.Duration
}

// New builds an orchestrator. Audits run in-process, one goroutine per job,
// with a bounded number of concurrent jobs.
func New(options Options) *Orchestrator {
	logger := options.Logger
	maxConcurrent := options.MaxConcurrentAudits
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	chunkTimeout := options.ChunkTimeout
	if chunkTimeout <= 0 {
		chunkTimeout = 90 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Orchestrator{
		store:        options.Store,
		retriever:    options.Retriever,
		llmClient:    options.LLM,
		log:          logger,
		ctx:          ctx,
		cancel:       cancel,
		sem:          make(chan struct{}, maxConcurrent),
		chunkTimeout: chunkTimeout,
	}
}

// Submit persists a PENDING audit and schedules it for background processing.
func (o *Orchestrator) Submit(ctx context.Context, params repository.CreateAuditParams) (*repository.AuditJob, error) {
	job, err := o.store.Create(ctx, params)
	if err != nil {
		return nil, err
	}

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()

		select {
		case o.sem <- struct{}{}:
			defer func() { <-o.sem }()
		case <-o.ctx.Done():
			o.fail(job.ID, "audit interrupted during server shutdown")
			return
		}

		o.run(job.ID, params.SourceCode, params.AnalysisDepth)
	}()

	return job, nil
}

// Close stops accepting new work and cancels running audits.
func (o *Orchestrator) Close() {
	o.cancel()
}

// Wait blocks until every submitted audit has finished.
func (o *Orchestrator) Wait() {
	o.wg.Wait()
}

func (o *Orchestrator) run(auditID, source, depth string) {
	log := o.log.With().Str("audit_id", auditID).Logger()
	started := time.Now()

	log.Info().Str("stage", "queued").Msg("audit_started")

	if o.llmClient == nil {
		o.fail(auditID, "LLM provider is not configured")
		return
	}

	if err := o.store.UpdateStatus(o.ctx, auditID, repository.StatusParsing); err != nil {
		log.Error().Err(err).Str("stage", "parsing").Msg("audit_status_update_failed")
		o.fail(auditID, "failed to update audit status")
		return
	}

	contract, err := parser.Parse(source)
	if err != nil {
		log.Warn().Err(err).Str("stage", "parsing").Msg("parsing_failed")
		o.fail(auditID, "unable to parse Solidity contract: "+err.Error())
		return
	}

	chunks := parser.Chunk(contract)
	log.Info().
		Str("stage", "parsing").
		Int("chunks", len(chunks)).
		Msg("parsing_complete")

	if err := o.store.UpdateProgress(o.ctx, auditID, repository.StatusAnalyzing, len(chunks), 0); err != nil {
		log.Error().Err(err).Str("stage", "analyzing").Msg("audit_progress_update_failed")
		o.fail(auditID, "failed to update audit progress")
		return
	}

	warnings := []string{}
	if o.retriever == nil {
		warnings = append(warnings, "RAG is not configured; analysis ran without exploit patterns.")
	}

	ragPatternsUsed := 0
	failures := 0
	model := ""
	rawFindings := make([]report.Finding, 0, len(chunks))

	for index, chunk := range chunks {
		if o.ctx.Err() != nil {
			o.fail(auditID, "audit interrupted by server shutdown")
			return
		}

		patterns := []rag.RetrievalResult{}
		if o.retriever != nil {
			chunkCtx, cancel := context.WithTimeout(o.ctx, o.chunkTimeout)
			found, retrieveErr := o.retriever.Retrieve(chunkCtx, chunk)
			cancel()

			if retrieveErr != nil {
				if o.ctx.Err() != nil {
					o.fail(auditID, "audit interrupted by server shutdown")
					return
				}
				log.Warn().
					Err(retrieveErr).
					Str("stage", "rag").
					Str("function_name", chunk.FunctionName).
					Msg("rag_retrieval_failed")
				warnings = appendWarning(warnings, "RAG retrieval failed for "+chunk.FunctionName+"()")
			} else {
				patterns = found
				ragPatternsUsed += len(found)
				log.Info().
					Str("stage", "rag").
					Str("function_name", chunk.FunctionName).
					Int("patterns", len(found)).
					Msg("rag_retrieval_complete")
			}
		}

		chunkCtx, cancel := context.WithTimeout(o.ctx, o.chunkTimeout)
		response, analyzeErr := o.llmClient.Analyze(chunkCtx, llm.AnalysisRequest{
			Chunk:    chunk,
			Patterns: patterns,
			Depth:    depth,
		})
		cancel()

		if analyzeErr != nil {
			if o.ctx.Err() != nil {
				o.fail(auditID, "audit interrupted by server shutdown")
				return
			}
			failures++
			log.Error().
				Err(analyzeErr).
				Str("stage", "analysis").
				Str("function_name", chunk.FunctionName).
				Str("llm_provider", o.llmClient.Name()).
				Msg("llm_analysis_failed")
			warnings = appendWarning(warnings, "Analysis failed for "+chunk.FunctionName+"()")
		} else {
			if model == "" && response.Model != "" {
				model = response.Model
			}
			converted := convertFindings(chunk, response.Findings, patterns)
			rawFindings = append(rawFindings, converted...)
			log.Info().
				Str("stage", "analysis").
				Str("function_name", chunk.FunctionName).
				Str("llm_provider", o.llmClient.Name()).
				Int("findings", len(response.Findings)).
				Msg("llm_analysis_complete")
		}

		if err := o.store.UpdateProgress(o.ctx, auditID, repository.StatusAnalyzing, len(chunks), index+1); err != nil {
			log.Error().Err(err).Str("stage", "analyzing").Msg("audit_progress_update_failed")
			o.fail(auditID, "failed to update audit progress")
			return
		}
	}

	if o.ctx.Err() != nil {
		o.fail(auditID, "audit interrupted by server shutdown")
		return
	}

	if len(chunks) > 0 && failures == len(chunks) {
		o.fail(auditID, fmt.Sprintf("all %d chunk analyses failed; check the LLM provider configuration", len(chunks)))
		return
	}

	if err := o.store.UpdateStatus(o.ctx, auditID, repository.StatusGeneratingReport); err != nil {
		log.Error().Err(err).Str("stage", "report").Msg("audit_status_update_failed")
		o.fail(auditID, "failed to update audit status")
		return
	}

	finalFindings := report.SortFindings(report.Deduplicate(rawFindings))
	if err := o.store.ReplaceFindings(o.ctx, auditID, toRepositoryFindings(finalFindings)); err != nil {
		log.Error().Err(err).Str("stage", "report").Msg("finding_persist_failed")
		o.fail(auditID, "failed to persist audit findings")
		return
	}

	if model == "" {
		model = o.llmClient.Name()
	}
	if err := o.store.Complete(o.ctx, auditID, model, ragPatternsUsed, warnings); err != nil {
		log.Error().Err(err).Str("stage", "report").Msg("audit_completion_failed")
		o.fail(auditID, "failed to complete audit")
		return
	}

	log.Info().
		Str("stage", "completed").
		Int64("duration_ms", time.Since(started).Milliseconds()).
		Int("findings", len(finalFindings)).
		Int("rag_patterns", ragPatternsUsed).
		Msg("audit_complete")
}

func (o *Orchestrator) fail(auditID, message string) {
	// Failure updates use an independent context so cancellation or a dead
	// request context cannot leave a job stuck in a non-terminal state.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := o.store.Fail(ctx, auditID, message); err != nil {
		o.log.Error().
			Err(err).
			Str("audit_id", auditID).
			Str("stage", "failed").
			Msg("audit_failure_persist_failed")
		return
	}

	o.log.Error().
		Str("audit_id", auditID).
		Str("stage", "failed").
		Str("error", message).
		Msg("audit_failed")
}

func appendWarning(warnings []string, warning string) []string {
	for _, existing := range warnings {
		if existing == warning {
			return warnings
		}
	}
	return append(warnings, warning)
}

func convertFindings(chunk parser.CodeChunk, findings []llm.LLMFinding, patterns []rag.RetrievalResult) []report.Finding {
	converted := make([]report.Finding, 0, len(findings))

	for _, finding := range findings {
		lineStart, lineEnd := normalizeLines(finding.LineStart, finding.LineEnd, chunk)
		vulnerableCode := finding.VulnerableCode
		if strings.TrimSpace(vulnerableCode) == "" {
			vulnerableCode = chunk.Code
		}

		result := report.Finding{
			ID:                uuid.NewString(),
			VulnerabilityType: finding.VulnerabilityType,
			Severity:          finding.Severity,
			Confidence:        finding.Confidence,
			AffectedFunction:  finding.AffectedFunction,
			LineStart:         lineStart,
			LineEnd:           lineEnd,
			Title:             finding.Title,
			Description:       finding.Description,
			VulnerableCode:    vulnerableCode,
			RecommendedFix:    finding.RecommendedFix,
			FixedCode:         finding.FixedCode,
			References:        []string{},
		}

		if pattern, ok := matchPattern(finding.MatchedPattern, patterns); ok {
			result.MatchedPattern = &report.MatchedPattern{
				Name:            pattern.Pattern.Name,
				RealWorldCase:   pattern.Pattern.RealWorldCase,
				SimilarityScore: pattern.SimilarityScore,
			}
			result.MatchedPatternID = pattern.Pattern.ID
			result.References = append([]string{}, pattern.Pattern.References...)
		}

		converted = append(converted, result)
	}

	return converted
}

func matchPattern(name string, patterns []rag.RetrievalResult) (rag.RetrievalResult, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return rag.RetrievalResult{}, false
	}

	for _, pattern := range patterns {
		if strings.EqualFold(strings.TrimSpace(pattern.Pattern.Name), name) {
			return pattern, true
		}
	}

	return rag.RetrievalResult{}, false
}

func normalizeLines(start, end int, chunk parser.CodeChunk) (int, int) {
	offset := 0
	if start <= 0 {
		start = chunk.LineStart
	} else if start < chunk.LineStart || start > chunk.LineEnd {
		// Some models answer with lines relative to the snippet; shift both
		// ends of the range by the same amount.
		offset = chunk.LineStart - 1
		start += offset
	}
	if end > 0 && offset != 0 {
		end += offset
	}

	if start < chunk.LineStart {
		start = chunk.LineStart
	}
	if start > chunk.LineEnd {
		start = chunk.LineEnd
	}

	if end <= 0 || end < start {
		end = start
	}
	if end > chunk.LineEnd {
		end = chunk.LineEnd
	}
	if end < chunk.LineStart {
		end = chunk.LineStart
	}

	return start, end
}

func toRepositoryFindings(findings []report.Finding) []repository.Finding {
	stored := make([]repository.Finding, 0, len(findings))

	for _, finding := range findings {
		record := repository.Finding{
			ID:                finding.ID,
			VulnerabilityType: finding.VulnerabilityType,
			Severity:          finding.Severity,
			Confidence:        finding.Confidence,
			AffectedFunction:  finding.AffectedFunction,
			Title:             finding.Title,
			Description:       finding.Description,
			VulnerableCode:    optionalString(finding.VulnerableCode),
			RecommendedFix:    optionalString(finding.RecommendedFix),
			FixedCode:         optionalString(finding.FixedCode),
			References:        finding.References,
			MatchedPatternID:  optionalString(finding.MatchedPatternID),
		}

		if finding.LineStart > 0 {
			start := finding.LineStart
			record.LineStart = &start
		}
		if finding.LineEnd > 0 {
			end := finding.LineEnd
			record.LineEnd = &end
		}
		if finding.MatchedPattern != nil {
			name := finding.MatchedPattern.Name
			record.MatchedPatternName = &name
			similarity := float64(finding.MatchedPattern.SimilarityScore)
			record.SimilarityScore = &similarity
		}

		stored = append(stored, record)
	}

	return stored
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
