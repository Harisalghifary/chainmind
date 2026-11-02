package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// testPool connects to TEST_DATABASE_URL and skips the test when it is unset,
// keeping `go test ./...` usable without Docker.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping database integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	if err := Migrate(ctx, pool, zerolog.Nop()); err != nil {
		pool.Close()
		t.Fatalf("Migrate() error = %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func TestExploitRepoInsertAndList(t *testing.T) {
	pool := testPool(t)
	repo := NewExploitRepo(pool)
	ctx := context.Background()

	name := "test-pattern-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM exploit_patterns WHERE name = $1`, name)
	})

	swc := "SWC-107"
	pattern, err := repo.Insert(ctx, &ExploitPattern{
		Name:           name,
		Category:       "reentrancy",
		SWCID:          &swc,
		Severity:       "critical",
		Description:    "test description",
		VulnerableCode: "function withdraw() external {}",
		References:     []string{"https://swcregistry.io/docs/SWC-107"},
	})
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	if pattern.ID == "" {
		t.Fatal("Insert() returned an empty id")
	}

	fetched, err := repo.GetByName(ctx, name)
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}
	if fetched.Name != name || fetched.Severity != "critical" {
		t.Errorf("fetched pattern = %+v", fetched)
	}
	if len(fetched.References) != 1 {
		t.Errorf("References = %v", fetched.References)
	}

	patterns, total, err := repo.List(ctx, ExploitFilter{Category: "reentrancy", Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 {
		t.Errorf("total = %d, want >= 1", total)
	}

	found := false
	for _, p := range patterns {
		if p.ID == pattern.ID {
			found = true
		}
	}
	if !found {
		t.Error("inserted pattern missing from List()")
	}
}

func TestExploitRepoUpsertIsIdempotent(t *testing.T) {
	pool := testPool(t)
	repo := NewExploitRepo(pool)
	ctx := context.Background()

	name := "test-upsert-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM exploit_patterns WHERE name = $1`, name)
	})

	input := &ExploitPattern{
		Name:           name,
		Category:       "oracle",
		Severity:       "high",
		Description:    "first",
		VulnerableCode: "code",
	}

	first, err := repo.UpsertByName(ctx, input)
	if err != nil {
		t.Fatalf("first UpsertByName() error = %v", err)
	}

	input.Description = "second"
	second, err := repo.UpsertByName(ctx, input)
	if err != nil {
		t.Fatalf("second UpsertByName() error = %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("upsert created a new row: %s != %s", second.ID, first.ID)
	}
	if second.Description != "second" {
		t.Errorf("Description = %q, want second", second.Description)
	}
}

func TestAuditRepoLifecycle(t *testing.T) {
	pool := testPool(t)
	repo := NewAuditRepo(pool)
	ctx := context.Background()

	contractName := "TestAudit-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_jobs WHERE contract_name = $1`, contractName)
	})

	job, err := repo.Create(ctx, CreateAuditParams{
		ContractName:  contractName,
		SourceCode:    "pragma solidity ^0.8.0;",
		AnalysisDepth: "standard",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if job.Status != StatusPending {
		t.Errorf("Status = %q, want PENDING", job.Status)
	}

	if err := repo.UpdateProgress(ctx, job.ID, StatusAnalyzing, 5, 2); err != nil {
		t.Fatalf("UpdateProgress() error = %v", err)
	}
	updated, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updated.Status != StatusAnalyzing || updated.TotalChunks != 5 || updated.AnalyzedChunks != 2 {
		t.Errorf("updated job = %+v", updated)
	}

	if err := repo.Complete(ctx, job.ID, "test-model", 5, []string{"warning"}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	completed, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID() after completion error = %v", err)
	}
	if completed.Status != StatusCompleted || completed.CompletedAt == nil {
		t.Errorf("completed job = %+v", completed)
	}

	if _, err := repo.GetByID(ctx, uuid.NewString()); err != ErrNotFound {
		t.Errorf("GetByID(unknown) error = %v, want ErrNotFound", err)
	}
}

func makeEmbedding(primary, secondary int) []float32 {
	vector := make([]float32, 1536)
	vector[primary] = 0.9
	vector[secondary] = 0.1
	return vector
}

func TestExploitRepoSearchOrdersBySimilarity(t *testing.T) {
	pool := testPool(t)
	repo := NewExploitRepo(pool)
	ctx := context.Background()

	suffix := uuid.NewString()
	relevantName := "search-relevant-" + suffix
	otherName := "search-other-" + suffix
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM exploit_patterns WHERE name = $1 OR name = $2`, relevantName, otherName)
	})

	relevant, err := repo.Insert(ctx, &ExploitPattern{
		Name:           relevantName,
		Category:       "reentrancy",
		Severity:       "critical",
		Description:    "call before state update",
		VulnerableCode: "msg.sender.call{value: amount}",
		Embedding:      makeEmbedding(0, 1),
	})
	if err != nil {
		t.Fatalf("insert relevant: %v", err)
	}
	other, err := repo.Insert(ctx, &ExploitPattern{
		Name:           otherName,
		Category:       "access_control",
		Severity:       "critical",
		Description:    "missing owner check",
		VulnerableCode: "function mint() external {}",
		Embedding:      makeEmbedding(1, 0),
	})
	if err != nil {
		t.Fatalf("insert other: %v", err)
	}
	results, err := repo.Search(ctx, makeEmbedding(0, 1), 5, 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("results = %d, want at least 2", len(results))
	}
	if results[0].Pattern.ID != relevant.ID {
		t.Errorf("top result = %s, want %s", results[0].Pattern.Name, relevant.Name)
	}
	if results[0].Similarity <= results[1].Similarity {
		t.Errorf("similarities not descending: %v", results)
	}

	filtered, err := repo.Search(ctx, makeEmbedding(0, 1), 5, 0.5)
	if err != nil {
		t.Fatalf("Search() with threshold error = %v", err)
	}
	for _, result := range filtered {
		if result.Pattern.ID == other.ID {
			t.Error("threshold did not filter the unrelated pattern")
		}
	}
}

func TestReplaceAndListFindings(t *testing.T) {
	pool := testPool(t)
	auditRepo := NewAuditRepo(pool)
	exploitRepo := NewExploitRepo(pool)
	ctx := context.Background()

	patternName := "test-finding-pattern-" + uuid.NewString()
	realCase := "The DAO Hack, 2016"
	pattern, err := exploitRepo.Insert(ctx, &ExploitPattern{
		Name:           patternName,
		Category:       "reentrancy",
		Severity:       "critical",
		Description:    "test pattern",
		VulnerableCode: "msg.sender.call{value: amount}(\"\");",
		RealWorldCase:  &realCase,
	})
	if err != nil {
		t.Fatalf("insert pattern: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM exploit_patterns WHERE name = $1`, patternName)
	})

	contractName := "TestFindings-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_jobs WHERE contract_name = $1`, contractName)
	})
	job, err := auditRepo.Create(ctx, CreateAuditParams{
		ContractName:  contractName,
		SourceCode:    "pragma solidity ^0.8.0;",
		AnalysisDepth: "standard",
	})
	if err != nil {
		t.Fatalf("create audit: %v", err)
	}

	lineStart, lineEnd := 12, 16
	confidence := float32(0.95)
	similarity := 0.94
	title := "Reentrancy in withdraw()"
	description := "External call before state update."
	vulnerableCode := "msg.sender.call{value: amount}(\"\");"
	recommendedFix := "Update state first."
	fixedCode := "balances[msg.sender] = 0;"

	findingID := uuid.NewString()
	err = auditRepo.ReplaceFindings(ctx, job.ID, []Finding{{
		ID:                 findingID,
		VulnerabilityType:  "reentrancy",
		Severity:           "critical",
		Confidence:         confidence,
		AffectedFunction:   "withdraw",
		LineStart:          &lineStart,
		LineEnd:            &lineEnd,
		Title:              title,
		Description:        description,
		VulnerableCode:     &vulnerableCode,
		RecommendedFix:     &recommendedFix,
		FixedCode:          &fixedCode,
		MatchedPatternID:   &pattern.ID,
		MatchedPatternName: &patternName,
		SimilarityScore:    &similarity,
		References:         []string{"https://swcregistry.io/docs/SWC-107"},
	}})
	if err != nil {
		t.Fatalf("ReplaceFindings() error = %v", err)
	}

	findings, err := auditRepo.ListFindings(ctx, job.ID)
	if err != nil {
		t.Fatalf("ListFindings() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %+v", findings)
	}

	finding := findings[0]
	if finding.ID != findingID || finding.Title != title {
		t.Errorf("finding = %+v", finding)
	}
	if finding.LineStart == nil || *finding.LineStart != lineStart {
		t.Errorf("line start = %v", finding.LineStart)
	}
	if finding.MatchedPatternID == nil || *finding.MatchedPatternID != pattern.ID {
		t.Errorf("matched pattern id = %v", finding.MatchedPatternID)
	}
	if finding.MatchedPatternRealWorldCase == nil || *finding.MatchedPatternRealWorldCase != realCase {
		t.Errorf("real world case = %v", finding.MatchedPatternRealWorldCase)
	}
	if len(finding.References) != 1 {
		t.Errorf("references = %v", finding.References)
	}

	if err := auditRepo.ReplaceFindings(ctx, job.ID, nil); err != nil {
		t.Fatalf("ReplaceFindings(nil) error = %v", err)
	}
	findings, err = auditRepo.ListFindings(ctx, job.ID)
	if err != nil {
		t.Fatalf("ListFindings() after clear error = %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("findings after clear = %+v", findings)
	}
}
