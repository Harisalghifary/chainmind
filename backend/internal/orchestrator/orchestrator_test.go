package orchestrator

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"chainmind/backend/internal/llm"
	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/rag"
	"chainmind/backend/internal/report"
	"chainmind/backend/internal/repository"
)

const demoContract = `pragma solidity ^0.8.0;

contract Demo {
    uint256 private value;

    function alpha() external { value = 1; }

    function beta() external { value = 2; }
}
`

type progressRecord struct {
	status   string
	total    int
	analyzed int
}

type fakeStore struct {
	mu sync.Mutex

	statuses       []string
	progress       []progressRecord
	findings       []repository.Finding
	completed      bool
	failMessage    string
	completedModel string
	ragPatterns    int
	warnings       []string
	replaceErr     error
}

func (f *fakeStore) Create(context.Context, repository.CreateAuditParams) (*repository.AuditJob, error) {
	return &repository.AuditJob{ID: "audit-1", Status: repository.StatusPending}, nil
}

func (f *fakeStore) UpdateStatus(_ context.Context, _ string, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statuses = append(f.statuses, status)
	return nil
}

func (f *fakeStore) UpdateProgress(_ context.Context, _ string, status string, total, analyzed int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.progress = append(f.progress, progressRecord{status: status, total: total, analyzed: analyzed})
	return nil
}

func (f *fakeStore) ReplaceFindings(_ context.Context, _ string, findings []repository.Finding) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.replaceErr != nil {
		return f.replaceErr
	}
	f.findings = append([]repository.Finding{}, findings...)
	return nil
}

func (f *fakeStore) Complete(_ context.Context, _ string, llmModel string, ragPatternsUsed int, warnings []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.completed = true
	f.completedModel = llmModel
	f.ragPatterns = ragPatternsUsed
	f.warnings = append([]string{}, warnings...)
	return nil
}

func (f *fakeStore) Fail(_ context.Context, _ string, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failMessage = message
	return nil
}

type fakeRetriever struct {
	mu      sync.Mutex
	results []rag.RetrievalResult
	err     error
	calls   int
}

func (f *fakeRetriever) Retrieve(context.Context, parser.CodeChunk) ([]rag.RetrievalResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.results, f.err
}

type fakeAuditor struct {
	mu    sync.Mutex
	name  string
	calls int
	fn    func(llm.AnalysisRequest) (*llm.AnalysisResponse, error)
}

func (f *fakeAuditor) Analyze(_ context.Context, request llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.fn(request)
}

func (f *fakeAuditor) Name() string { return f.name }

func newTestOrchestrator(store AuditStore, retriever PatternRetriever, auditor Auditor) *Orchestrator {
	return New(Options{
		Store:        store,
		Retriever:    retriever,
		LLM:          auditor,
		Logger:       zerolog.Nop(),
		ChunkTimeout: 2 * time.Second,
	})
}

func testPattern() rag.RetrievalResult {
	return rag.RetrievalResult{
		Pattern: rag.ExploitPattern{
			ID:            "pattern-1",
			Name:          "Pattern A",
			Category:      "reentrancy",
			Severity:      "critical",
			RealWorldCase: "The DAO Hack, 2016",
			References:    []string{"https://swcregistry.io/docs/SWC-107"},
		},
		SimilarityScore: 0.91,
	}
}

func TestOrchestratorHappyPath(t *testing.T) {
	store := &fakeStore{}
	retriever := &fakeRetriever{results: []rag.RetrievalResult{testPattern()}}
	auditor := &fakeAuditor{name: "fake-provider", fn: func(request llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
		if request.Chunk.FunctionName != "alpha" {
			return &llm.AnalysisResponse{Findings: []llm.LLMFinding{}, Model: "fake-model"}, nil
		}
		return &llm.AnalysisResponse{
			Model: "fake-model",
			Findings: []llm.LLMFinding{{
				VulnerabilityType: "reentrancy",
				Severity:          "critical",
				Confidence:        0.95,
				AffectedFunction:  "alpha",
				LineStart:         6,
				LineEnd:           6,
				Title:             "Reentrancy in alpha()",
				Description:       "External call before state update.",
				VulnerableCode:    "value = 1;",
				RecommendedFix:    "Update state first.",
				FixedCode:         "value = 1;",
				MatchedPattern:    "Pattern A",
			}},
		}, nil
	}}

	orchestrator := newTestOrchestrator(store, retriever, auditor)
	job, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Demo",
		SourceCode:    demoContract,
		AnalysisDepth: "standard",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if job.ID != "audit-1" {
		t.Errorf("job id = %q", job.ID)
	}

	orchestrator.Wait()

	if store.failMessage != "" {
		t.Fatalf("audit failed: %s", store.failMessage)
	}
	if !store.completed {
		t.Fatal("audit was not completed")
	}
	if store.completedModel != "fake-model" {
		t.Errorf("completed model = %q", store.completedModel)
	}
	if store.ragPatterns != 2 {
		t.Errorf("rag patterns = %d, want 2 (one per chunk)", store.ragPatterns)
	}
	if len(store.warnings) != 0 {
		t.Errorf("warnings = %v", store.warnings)
	}
	if len(store.findings) != 1 {
		t.Fatalf("findings = %+v", store.findings)
	}

	finding := store.findings[0]
	if finding.MatchedPatternID == nil || *finding.MatchedPatternID != "pattern-1" {
		t.Errorf("matched pattern id = %v", finding.MatchedPatternID)
	}
	if finding.LineStart == nil || *finding.LineStart != 6 {
		t.Errorf("line start = %v", finding.LineStart)
	}
	if len(finding.References) != 1 {
		t.Errorf("references = %v", finding.References)
	}
	if !containsStatus(store.statuses, repository.StatusParsing) ||
		!containsStatus(store.statuses, repository.StatusGeneratingReport) {
		t.Errorf("statuses = %v", store.statuses)
	}
	if store.progress[0].status != repository.StatusAnalyzing {
		t.Errorf("first progress status = %q, want ANALYZING", store.progress[0].status)
	}

	last := store.progress[len(store.progress)-1]
	if last.total != 2 || last.analyzed != 2 || last.status != repository.StatusAnalyzing {
		t.Errorf("final progress = %+v", last)
	}
}

func TestOrchestratorContinuesWithoutRAG(t *testing.T) {
	store := &fakeStore{}
	retriever := &fakeRetriever{err: errors.New("vector db unavailable")}
	auditor := &fakeAuditor{name: "fake-provider", fn: func(llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
		return &llm.AnalysisResponse{Findings: []llm.LLMFinding{}, Model: "fake-model"}, nil
	}}

	orchestrator := newTestOrchestrator(store, retriever, auditor)
	if _, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Demo",
		SourceCode:    demoContract,
		AnalysisDepth: "quick",
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	orchestrator.Wait()

	if !store.completed {
		t.Fatalf("audit should complete without RAG, fail message = %q", store.failMessage)
	}
	if store.ragPatterns != 0 {
		t.Errorf("rag patterns = %d, want 0", store.ragPatterns)
	}
	if len(store.warnings) != 2 {
		t.Errorf("warnings = %v, want one per chunk", store.warnings)
	}
	for _, warning := range store.warnings {
		if !strings.Contains(warning, "RAG retrieval failed") {
			t.Errorf("unexpected warning %q", warning)
		}
	}
}

func TestOrchestratorWarnsWhenRetrieverNotConfigured(t *testing.T) {
	store := &fakeStore{}
	auditor := &fakeAuditor{name: "fake-provider", fn: func(llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
		return &llm.AnalysisResponse{Findings: []llm.LLMFinding{}, Model: "fake-model"}, nil
	}}

	orchestrator := newTestOrchestrator(store, nil, auditor)
	if _, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Demo",
		SourceCode:    demoContract,
		AnalysisDepth: "standard",
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	orchestrator.Wait()

	if !store.completed || len(store.warnings) != 1 {
		t.Fatalf("completed = %v, warnings = %v", store.completed, store.warnings)
	}
	if !strings.Contains(store.warnings[0], "RAG is not configured") {
		t.Errorf("warning = %q", store.warnings[0])
	}
}

func TestOrchestratorToleratesSingleChunkFailure(t *testing.T) {
	store := &fakeStore{}
	retriever := &fakeRetriever{results: []rag.RetrievalResult{testPattern()}}
	auditor := &fakeAuditor{name: "fake-provider", fn: func(request llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
		if request.Chunk.FunctionName == "alpha" {
			return nil, errors.New("invalid JSON from provider")
		}
		return &llm.AnalysisResponse{Findings: []llm.LLMFinding{}, Model: "fake-model"}, nil
	}}

	orchestrator := newTestOrchestrator(store, retriever, auditor)
	if _, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Demo",
		SourceCode:    demoContract,
		AnalysisDepth: "standard",
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	orchestrator.Wait()

	if !store.completed {
		t.Fatalf("audit should complete with one failed chunk: %q", store.failMessage)
	}
	if len(store.warnings) != 1 || !strings.Contains(store.warnings[0], "alpha") {
		t.Errorf("warnings = %v", store.warnings)
	}
}

func TestOrchestratorFailsWhenAllChunksFail(t *testing.T) {
	store := &fakeStore{}
	auditor := &fakeAuditor{name: "fake-provider", fn: func(llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
		return nil, errors.New("provider down")
	}}

	orchestrator := newTestOrchestrator(store, &fakeRetriever{}, auditor)
	if _, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Demo",
		SourceCode:    demoContract,
		AnalysisDepth: "standard",
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	orchestrator.Wait()

	if store.completed {
		t.Fatal("audit must not complete when every chunk fails")
	}
	if !strings.Contains(store.failMessage, "all 2 chunk analyses failed") {
		t.Errorf("fail message = %q", store.failMessage)
	}
}

func TestOrchestratorFailsOnParseError(t *testing.T) {
	store := &fakeStore{}
	auditor := &fakeAuditor{name: "fake-provider", fn: func(llm.AnalysisRequest) (*llm.AnalysisResponse, error) {
		return &llm.AnalysisResponse{}, nil
	}}

	orchestrator := newTestOrchestrator(store, nil, auditor)
	if _, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Broken",
		SourceCode:    "this is not solidity",
		AnalysisDepth: "standard",
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	orchestrator.Wait()

	if store.completed {
		t.Fatal("invalid source must not complete")
	}
	if !strings.Contains(store.failMessage, "unable to parse Solidity contract") {
		t.Errorf("fail message = %q", store.failMessage)
	}
}

func TestOrchestratorFailsWithoutLLM(t *testing.T) {
	store := &fakeStore{}
	orchestrator := newTestOrchestrator(store, nil, nil)

	if _, err := orchestrator.Submit(context.Background(), repository.CreateAuditParams{
		ContractName:  "Demo",
		SourceCode:    demoContract,
		AnalysisDepth: "standard",
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	orchestrator.Wait()

	if !strings.Contains(store.failMessage, "LLM provider is not configured") {
		t.Errorf("fail message = %q", store.failMessage)
	}
}

func TestNormalizeLines(t *testing.T) {
	chunk := parser.CodeChunk{LineStart: 11, LineEnd: 20}

	cases := []struct {
		start, end         int
		wantStart, wantEnd int
	}{
		{14, 16, 14, 16}, // absolute lines inside the chunk
		{1, 3, 11, 13},   // relative lines
		{0, 0, 11, 11},   // missing
		{25, 30, 20, 20}, // beyond the chunk
		{18, 12, 18, 18}, // inverted range
	}

	for _, testCase := range cases {
		start, end := normalizeLines(testCase.start, testCase.end, chunk)
		if start != testCase.wantStart || end != testCase.wantEnd {
			t.Errorf("normalizeLines(%d,%d) = %d,%d; want %d,%d",
				testCase.start, testCase.end, start, end, testCase.wantStart, testCase.wantEnd)
		}
	}
}

func TestConvertFindingsMapsMatchedPattern(t *testing.T) {
	chunk := parser.CodeChunk{LineStart: 10, LineEnd: 15}
	findings := []llm.LLMFinding{{
		VulnerabilityType: "reentrancy",
		Severity:          "critical",
		Confidence:        0.9,
		AffectedFunction:  "withdraw",
		LineStart:         12,
		LineEnd:           14,
		Title:             "Reentrancy",
		Description:       "desc",
		MatchedPattern:    "Pattern A",
	}}

	converted := convertFindings(chunk, findings, []rag.RetrievalResult{testPattern()})
	if len(converted) != 1 {
		t.Fatalf("converted = %+v", converted)
	}
	if converted[0].MatchedPattern == nil || converted[0].MatchedPattern.Name != "Pattern A" {
		t.Errorf("matched pattern = %+v", converted[0].MatchedPattern)
	}
	if converted[0].MatchedPatternID != "pattern-1" {
		t.Errorf("matched pattern id = %q", converted[0].MatchedPatternID)
	}
	if len(converted[0].References) != 1 {
		t.Errorf("references = %v", converted[0].References)
	}
	if converted[0].VulnerableCode != chunk.Code {
		t.Errorf("vulnerable code should fall back to the chunk, got %q", converted[0].VulnerableCode)
	}
}

func TestRepositoryFindingMapping(t *testing.T) {
	findings := []report.Finding{{
		ID:                "finding-1",
		VulnerabilityType: "reentrancy",
		Severity:          "critical",
		Confidence:        0.9,
		AffectedFunction:  "withdraw",
		LineStart:         12,
		LineEnd:           14,
		Title:             "Reentrancy",
		Description:       "desc",
		MatchedPattern:    &report.MatchedPattern{Name: "Pattern A", SimilarityScore: 0.9},
		MatchedPatternID:  "pattern-1",
		References:        []string{"ref"},
	}}

	stored := toRepositoryFindings(findings)
	if len(stored) != 1 {
		t.Fatalf("stored = %+v", stored)
	}
	if stored[0].MatchedPatternID == nil || *stored[0].MatchedPatternID != "pattern-1" {
		t.Errorf("pattern id = %v", stored[0].MatchedPatternID)
	}
	if stored[0].LineStart == nil || *stored[0].LineStart != 12 {
		t.Errorf("line start = %v", stored[0].LineStart)
	}
	if stored[0].SimilarityScore == nil || math.Abs(*stored[0].SimilarityScore-0.9) > 0.001 {
		t.Errorf("similarity = %v", stored[0].SimilarityScore)
	}
}

func containsStatus(statuses []string, status string) bool {
	for _, value := range statuses {
		if value == status {
			return true
		}
	}
	return false
}
