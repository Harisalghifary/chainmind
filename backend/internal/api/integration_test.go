package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"chainmind/backend/internal/api"
	"chainmind/backend/internal/config"
	"chainmind/backend/internal/llm"
	"chainmind/backend/internal/orchestrator"
	"chainmind/backend/internal/report"
	"chainmind/backend/internal/repository"
)

// setupIntegration boots the real router against TEST_DATABASE_URL with the
// deterministic mock LLM and no RAG. It skips when no database is configured.
func setupIntegration(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping API integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := repository.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	if err := repository.Migrate(ctx, pool, zerolog.Nop()); err != nil {
		pool.Close()
		t.Fatalf("Migrate() error = %v", err)
	}
	t.Cleanup(pool.Close)

	auditRepo := repository.NewAuditRepo(pool)
	exploitRepo := repository.NewExploitRepo(pool)

	worker := orchestrator.New(orchestrator.Options{
		Store:     auditRepo,
		Retriever: nil,
		LLM:       llm.NewMockClient(),
		Logger:    zerolog.Nop(),
	})
	t.Cleanup(worker.Close)

	router := api.NewRouter(&config.Config{
		Env:                "test",
		DefaultLLMProvider: "mock",
		CORSAllowedOrigins: []string{"http://localhost:3000"},
	}, pool, auditRepo, exploitRepo, worker, zerolog.Nop())

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, pool
}

func readTestContract(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", path))
	if err != nil {
		t.Fatalf("read test contract: %v", err)
	}
	return string(content)
}

func submitAudit(t *testing.T, serverURL, contractName, source string) string {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"contract_name":  contractName,
		"source_code":    source,
		"analysis_depth": "standard",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	response, err := http.Post(serverURL+"/api/v1/audit", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /audit error = %v", err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /audit status = %d, body = %s", response.StatusCode, body)
	}

	var created struct {
		AuditID string `json:"audit_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	if created.AuditID == "" || created.Status != "PENDING" {
		t.Fatalf("submit response = %+v", created)
	}

	return created.AuditID
}

func waitForCompletion(t *testing.T, serverURL, auditID string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	status := ""

	for time.Now().Before(deadline) {
		response, err := http.Get(fmt.Sprintf("%s/api/v1/audit/%s", serverURL, auditID))
		if err != nil {
			t.Fatalf("GET /audit/%s error = %v", auditID, err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()

		var current struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &current); err != nil {
			t.Fatalf("decode status: %v", err)
		}

		status = current.Status
		if status == "COMPLETED" || status == "FAILED" {
			if status == "FAILED" {
				t.Fatalf("audit failed: %s", body)
			}
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("audit %s did not finish, last status %q", auditID, status)
}

func fetchReport(t *testing.T, serverURL, auditID string) report.AuditReport {
	t.Helper()

	response, err := http.Get(fmt.Sprintf("%s/api/v1/audit/%s/report", serverURL, auditID))
	if err != nil {
		t.Fatalf("GET report error = %v", err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET report status = %d, body = %s", response.StatusCode, body)
	}

	var auditReport report.AuditReport
	if err := json.Unmarshal(body, &auditReport); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return auditReport
}

func TestAPIAuditsVulnerableAndSafeContracts(t *testing.T) {
	server, pool := setupIntegration(t)

	tests := []struct {
		name           string
		path           string
		contractName   string
		wantType       string
		wantCritical   int
		wantNoHighCrit bool
	}{
		{
			name:         "reentrancy",
			path:         "vulnerable/reentrancy.sol",
			contractName: "VulnerableVault",
			wantType:     "reentrancy",
			wantCritical: 1,
		},
		{
			name:         "access control",
			path:         "vulnerable/access_control.sol",
			contractName: "UnprotectedAdmin",
			wantType:     "access_control",
			wantCritical: 2,
		},
		{
			name:           "safe vault",
			path:           "safe/safe_vault.sol",
			contractName:   "SafeVault",
			wantNoHighCrit: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			auditID := submitAudit(t, server.URL, testCase.contractName, readTestContract(t, testCase.path))
			t.Cleanup(func() {
				_, _ = pool.Exec(context.Background(), `DELETE FROM audit_jobs WHERE id = $1`, auditID)
			})

			waitForCompletion(t, server.URL, auditID)
			auditReport := fetchReport(t, server.URL, auditID)

			if auditReport.ContractName != testCase.contractName {
				t.Errorf("contract name = %q", auditReport.ContractName)
			}
			if auditReport.Metadata.LLMModel != "mock-auditor" {
				t.Errorf("model = %q", auditReport.Metadata.LLMModel)
			}
			if len(auditReport.Metadata.Warnings) == 0 {
				t.Error("expected a RAG-disabled warning because integration runs without a retriever")
			}

			if testCase.wantNoHighCrit {
				if auditReport.Summary.Critical != 0 || auditReport.Summary.High != 0 {
					t.Errorf("safe contract findings = %+v", auditReport.Summary)
				}
				return
			}

			if auditReport.Summary.Critical != testCase.wantCritical {
				t.Errorf("critical findings = %d, want %d", auditReport.Summary.Critical, testCase.wantCritical)
			}

			found := false
			for _, finding := range auditReport.Findings {
				if finding.VulnerabilityType == testCase.wantType {
					found = true
				}
			}
			if !found {
				t.Errorf("no %q finding in %+v", testCase.wantType, auditReport.Findings)
			}
		})
	}
}

func TestAPIRejectsInvalidInput(t *testing.T) {
	server, _ := setupIntegration(t)

	response, err := http.Post(server.URL+"/api/v1/audit", "application/json",
		bytes.NewReader([]byte(`{"contract_name":"X"}`)))
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", response.StatusCode)
	}

	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error != "INVALID_INPUT" || body.Message != "source_code is required" {
		t.Errorf("error = %+v", body)
	}

	response, err = http.Get(server.URL + "/api/v1/audit/" + uuid.NewString())
	if err != nil {
		t.Fatalf("GET unknown audit error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("unknown audit status = %d", response.StatusCode)
	}
}
