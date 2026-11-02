package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"chainmind/backend/internal/api"
	"chainmind/backend/internal/api/handlers"
	"chainmind/backend/internal/dto"
	"chainmind/backend/internal/report"
	"chainmind/backend/internal/repository"
)

const minimalContract = `pragma solidity ^0.8.0;

contract Demo {
    uint256 private value;

    function setValue(uint256 next) external {
        value = next;
    }
}
`

type fakeSubmitter struct {
	params repository.CreateAuditParams
	job    *repository.AuditJob
	err    error
	calls  int
}

func (f *fakeSubmitter) Submit(_ context.Context, params repository.CreateAuditParams) (*repository.AuditJob, error) {
	f.calls++
	f.params = params
	if f.err != nil {
		return nil, f.err
	}
	return f.job, nil
}

type fakeReader struct {
	job      *repository.AuditJob
	findings []repository.Finding
	getErr   error
	listErr  error
}

func (f *fakeReader) GetByID(context.Context, string) (*repository.AuditJob, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.job, nil
}

func (f *fakeReader) ListFindings(context.Context, string) ([]repository.Finding, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.findings, nil
}

type fakeLister struct {
	patterns []repository.ExploitPattern
	total    int
	filter   repository.ExploitFilter
	err      error
}

func (f *fakeLister) List(_ context.Context, filter repository.ExploitFilter) ([]repository.ExploitPattern, int, error) {
	f.filter = filter
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.patterns, f.total, nil
}

func newTestEngine(register func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(api.ErrorHandler(zerolog.Nop()))
	register(engine)
	return engine
}

func perform(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}

	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v (%s)", err, recorder.Body.String())
	}
	return body.Error, body.Message
}

func newAuditEngine(submitter handlers.AuditSubmitter, reader handlers.AuditReader) *gin.Engine {
	handler := handlers.NewAuditHandler(submitter, reader)
	reporter := handlers.NewReportHandler(reader)
	return newTestEngine(func(engine *gin.Engine) {
		engine.POST("/api/v1/audit", handler.Submit)
		engine.GET("/api/v1/audit/:id", handler.Get)
		engine.GET("/api/v1/audit/:id/report", reporter.Get)
	})
}

func TestSubmitAuditAcceptsValidContract(t *testing.T) {
	now := time.Now().UTC()
	submitter := &fakeSubmitter{job: &repository.AuditJob{
		ID:            "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42",
		Status:        repository.StatusPending,
		AnalysisDepth: "standard",
		CreatedAt:     now,
	}}
	engine := newAuditEngine(submitter, &fakeReader{})

	body := `{"contract_name":"Demo","source_code":` + quote(minimalContract) + `}`
	recorder := perform(engine, http.MethodPost, "/api/v1/audit", body)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response dto.AuditCreatedResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AuditID != "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42" || response.Status != "PENDING" {
		t.Errorf("response = %+v", response)
	}
	if response.EstimatedSeconds != 30 {
		t.Errorf("estimated seconds = %d", response.EstimatedSeconds)
	}
	if submitter.params.AnalysisDepth != "standard" {
		t.Errorf("depth = %q, want default standard", submitter.params.AnalysisDepth)
	}
	if submitter.params.SolidityVersion != nil {
		t.Errorf("solidity version = %v, want nil", submitter.params.SolidityVersion)
	}
}

func TestSubmitAuditRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "missing contract name",
			body:    `{"source_code":"contract C {}"}`,
			message: "contract_name is required",
		},
		{
			name:    "missing source",
			body:    `{"contract_name":"C"}`,
			message: "source_code is required",
		},
		{
			name:    "invalid depth",
			body:    `{"contract_name":"C","source_code":"contract C {}","analysis_depth":"slow"}`,
			message: "analysis_depth must be one of quick, standard, deep",
		},
		{
			name:    "malformed json",
			body:    `{"contract_name":`,
			message: "request body must be valid JSON",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			submitter := &fakeSubmitter{}
			engine := newAuditEngine(submitter, &fakeReader{})

			recorder := perform(engine, http.MethodPost, "/api/v1/audit", testCase.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}

			code, message := decodeError(t, recorder)
			if code != "INVALID_INPUT" || message != testCase.message {
				t.Errorf("error = %s / %s", code, message)
			}
			if submitter.calls != 0 {
				t.Error("submitter must not be called for invalid input")
			}
		})
	}
}

func TestSubmitAuditRejectsUnparseableSource(t *testing.T) {
	submitter := &fakeSubmitter{}
	engine := newAuditEngine(submitter, &fakeReader{})

	recorder := perform(engine, http.MethodPost, "/api/v1/audit",
		`{"contract_name":"Broken","source_code":"this is not solidity"}`)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	code, message := decodeError(t, recorder)
	if code != "PARSE_ERROR" || message != "Unable to parse Solidity contract" {
		t.Errorf("error = %s / %s", code, message)
	}
	if submitter.calls != 0 {
		t.Error("submitter must not be called for unparseable source")
	}
}

func TestSubmitAuditMapsStoreFailureTo503(t *testing.T) {
	submitter := &fakeSubmitter{err: errors.New("database down")}
	engine := newAuditEngine(submitter, &fakeReader{})

	body := `{"contract_name":"Demo","source_code":` + quote(minimalContract) + `}`
	recorder := perform(engine, http.MethodPost, "/api/v1/audit", body)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", recorder.Code)
	}
	code, _ := decodeError(t, recorder)
	if code != "SERVICE_UNAVAILABLE" {
		t.Errorf("error = %q", code)
	}
}

func TestGetAuditStatus(t *testing.T) {
	now := time.Now().UTC()
	job := &repository.AuditJob{
		ID:             "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42",
		Status:         repository.StatusAnalyzing,
		TotalChunks:    5,
		AnalyzedChunks: 2,
		CreatedAt:      now.Add(-10 * time.Second),
		UpdatedAt:      now,
	}
	engine := newAuditEngine(&fakeSubmitter{}, &fakeReader{job: job})

	recorder := perform(engine, http.MethodGet, "/api/v1/audit/"+job.ID, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response dto.AuditStatusResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Progress.TotalChunks != 5 || response.Progress.AnalyzedChunks != 2 ||
		response.Progress.CurrentStage != "analyzing" {
		t.Errorf("progress = %+v", response.Progress)
	}
	if response.ErrorMessage != nil || response.CompletedAt != nil {
		t.Errorf("nullable fields = %+v / %+v", response.ErrorMessage, response.CompletedAt)
	}
}

func TestGetAuditStatusNotFound(t *testing.T) {
	engine := newAuditEngine(&fakeSubmitter{}, &fakeReader{getErr: repository.ErrNotFound})

	recorder := perform(engine, http.MethodGet, "/api/v1/audit/"+uuid.NewString(), "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
	if code, _ := decodeError(t, recorder); code != "AUDIT_NOT_FOUND" {
		t.Errorf("error = %q", code)
	}

	recorder = perform(engine, http.MethodGet, "/api/v1/audit/not-a-uuid", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("malformed id status = %d", recorder.Code)
	}
}

func TestGetAuditReport(t *testing.T) {
	now := time.Now().UTC()
	started := now.Add(-20 * time.Second)
	job := &repository.AuditJob{
		ID:              "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42",
		ContractName:    "VulnerableVault",
		Status:          repository.StatusCompleted,
		AnalysisDepth:   "standard",
		RAGPatternsUsed: 5,
		LLMModel:        strPtr("claude-test"),
		Warnings:        []string{"rag degraded"},
		CreatedAt:       started,
		UpdatedAt:       now,
		CompletedAt:     &now,
	}

	lineStart, lineEnd := 12, 16
	confidence := float32(0.95)
	similarity := 0.94
	patternID := "f2d1c9e4-8f7b-4c1d-9e42-8f14e45fceea"
	patternName := "Reentrancy via external call before state update"
	realCase := "The DAO Hack, 2016"
	reader := &fakeReader{
		job: job,
		findings: []repository.Finding{{
			ID:                          "f2d1c9e4-8f7b-4c1d-9e42-8f14e45fceea",
			AuditID:                     job.ID,
			VulnerabilityType:           "reentrancy",
			Severity:                    "critical",
			Confidence:                  confidence,
			AffectedFunction:            "withdraw",
			LineStart:                   &lineStart,
			LineEnd:                     &lineEnd,
			Title:                       "Reentrancy vulnerability in withdraw()",
			Description:                 "External call before state update.",
			VulnerableCode:              strPtr("function withdraw() external {}"),
			RecommendedFix:              strPtr("Update state first."),
			FixedCode:                   strPtr("balances[msg.sender] = 0;"),
			MatchedPatternID:            &patternID,
			MatchedPatternName:          &patternName,
			SimilarityScore:             &similarity,
			References:                  []string{"https://swcregistry.io/docs/SWC-107"},
			MatchedPatternRealWorldCase: &realCase,
		}},
	}

	engine := newAuditEngine(&fakeSubmitter{}, reader)
	recorder := perform(engine, http.MethodGet, "/api/v1/audit/"+job.ID+"/report", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response report.AuditReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if response.Summary.TotalFindings != 1 || response.Summary.Critical != 1 {
		t.Errorf("summary = %+v", response.Summary)
	}
	if len(response.Findings) != 1 {
		t.Fatalf("findings = %+v", response.Findings)
	}

	finding := response.Findings[0]
	if finding.MatchedPattern == nil || finding.MatchedPattern.Name != patternName ||
		finding.MatchedPattern.RealWorldCase != realCase {
		t.Errorf("matched pattern = %+v", finding.MatchedPattern)
	}
	if finding.LineStart != 12 || finding.LineEnd != 16 || finding.Confidence != 0.95 {
		t.Errorf("finding = %+v", finding)
	}
	if len(finding.References) != 1 {
		t.Errorf("references = %v", finding.References)
	}
	if response.Metadata.LLMModel != "claude-test" || response.Metadata.RAGPatternsUsed != 5 ||
		response.Metadata.ProcessingTimeMS < 20000 {
		t.Errorf("metadata = %+v", response.Metadata)
	}
	if len(response.Metadata.Warnings) != 1 {
		t.Errorf("warnings = %v", response.Metadata.Warnings)
	}
}

func TestGetAuditReportNotReady(t *testing.T) {
	job := &repository.AuditJob{ID: uuid.NewString(), Status: repository.StatusAnalyzing}
	engine := newAuditEngine(&fakeSubmitter{}, &fakeReader{job: job})

	recorder := perform(engine, http.MethodGet, "/api/v1/audit/"+job.ID+"/report", "")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d", recorder.Code)
	}
	if code, _ := decodeError(t, recorder); code != "AUDIT_NOT_READY" {
		t.Errorf("error = %q", code)
	}
}

func TestGetAuditReportNotFound(t *testing.T) {
	engine := newAuditEngine(&fakeSubmitter{}, &fakeReader{getErr: repository.ErrNotFound})

	recorder := perform(engine, http.MethodGet, "/api/v1/audit/"+uuid.NewString()+"/report", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestListExploits(t *testing.T) {
	swc := "SWC-107"
	realCase := "The DAO Hack, 2016"
	lister := &fakeLister{
		total: 21,
		patterns: []repository.ExploitPattern{{
			ID:            "pattern-1",
			Name:          "Reentrancy via external call before state update",
			Category:      "reentrancy",
			SWCID:         &swc,
			Severity:      "critical",
			Description:   "desc",
			RealWorldCase: &realCase,
		}},
	}

	handler := handlers.NewExploitsHandler(lister)
	engine := newTestEngine(func(engine *gin.Engine) {
		engine.GET("/api/v1/exploits", handler.List)
	})

	recorder := perform(engine, http.MethodGet, "/api/v1/exploits?category=reentrancy&severity=critical&page=2&limit=5", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response dto.ExploitListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Total != 21 || response.Page != 2 || response.Limit != 5 {
		t.Errorf("pagination = %+v", response)
	}
	if len(response.Patterns) != 1 || response.Patterns[0].SWCID != "SWC-107" {
		t.Errorf("patterns = %+v", response.Patterns)
	}
	if lister.filter.Category != "reentrancy" || lister.filter.Severity != "critical" ||
		lister.filter.Page != 2 || lister.filter.Limit != 5 {
		t.Errorf("filter = %+v", lister.filter)
	}
}

func TestListExploitsDefaultsAndValidation(t *testing.T) {
	lister := &fakeLister{patterns: []repository.ExploitPattern{}}
	handler := handlers.NewExploitsHandler(lister)
	engine := newTestEngine(func(engine *gin.Engine) {
		engine.GET("/api/v1/exploits", handler.List)
	})

	recorder := perform(engine, http.MethodGet, "/api/v1/exploits", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if lister.filter.Page != 1 || lister.filter.Limit != 20 {
		t.Errorf("defaults = %+v", lister.filter)
	}

	var response dto.ExploitListResponse
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	if response.Patterns == nil {
		t.Error("patterns must be an empty array, not null")
	}

	for _, path := range []string{
		"/api/v1/exploits?severity=severe",
		"/api/v1/exploits?page=0",
		"/api/v1/exploits?limit=abc",
	} {
		recorder = perform(engine, http.MethodGet, path, "")
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s status = %d", path, recorder.Code)
		}
		if code, _ := decodeError(t, recorder); code != "INVALID_INPUT" {
			t.Errorf("%s error = %q", path, code)
		}
	}
}

func TestListExploitsStoreFailure(t *testing.T) {
	lister := &fakeLister{err: errors.New("database down")}
	handler := handlers.NewExploitsHandler(lister)
	engine := newTestEngine(func(engine *gin.Engine) {
		engine.GET("/api/v1/exploits", handler.List)
	})

	recorder := perform(engine, http.MethodGet, "/api/v1/exploits", "")
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func quote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func strPtr(value string) *string {
	return &value
}
