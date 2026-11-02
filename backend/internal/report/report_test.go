package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDeduplicateMergesSameIssue(t *testing.T) {
	findings := []Finding{
		{
			ID: "f1", VulnerabilityType: "reentrancy", Severity: "critical", Confidence: 0.7,
			AffectedFunction: "withdraw", LineStart: 14, LineEnd: 16, Title: "Reentrancy",
			References:     []string{"https://swcregistry.io/docs/SWC-107"},
			MatchedPattern: &MatchedPattern{Name: "A", SimilarityScore: 0.8},
		},
		{
			ID: "f2", VulnerabilityType: "REENTRANCY", Severity: "critical", Confidence: 0.95,
			AffectedFunction: "withdraw", LineStart: 15, LineEnd: 17, Title: "Reentrancy",
			FixedCode: "balances[msg.sender] = 0;", RecommendedFix: "Reset first.",
			References:     []string{"https://consensys.io/reentrancy", "https://swcregistry.io/docs/SWC-107"},
			MatchedPattern: &MatchedPattern{Name: "B", SimilarityScore: 0.91},
		},
		{
			ID: "f3", VulnerabilityType: "reentrancy", Severity: "critical", Confidence: 0.9,
			AffectedFunction: "deposit", LineStart: 30, LineEnd: 32, Title: "Reentrancy",
		},
	}

	merged := Deduplicate(findings)
	if len(merged) != 2 {
		t.Fatalf("merged = %d, want 2: %+v", len(merged), merged)
	}

	var withdraw Finding
	for _, finding := range merged {
		if finding.AffectedFunction == "withdraw" {
			withdraw = finding
		}
	}
	if withdraw.ID != "f2" {
		t.Errorf("kept id = %s, want f2 (higher confidence)", withdraw.ID)
	}
	if withdraw.Confidence != 0.95 {
		t.Errorf("confidence = %v", withdraw.Confidence)
	}
	if len(withdraw.References) != 2 {
		t.Errorf("references = %v", withdraw.References)
	}
	if withdraw.FixedCode != "balances[msg.sender] = 0;" || withdraw.RecommendedFix != "Reset first." {
		t.Errorf("merged fix fields = %+v", withdraw)
	}
	if withdraw.MatchedPattern == nil || withdraw.MatchedPattern.Name != "B" {
		t.Errorf("matched pattern = %+v", withdraw.MatchedPattern)
	}
	if withdraw.LineStart != 14 || withdraw.LineEnd != 17 {
		t.Errorf("line range = %d-%d, want 14-17", withdraw.LineStart, withdraw.LineEnd)
	}
}

func TestDeduplicateKeepsSeparateLineRanges(t *testing.T) {
	findings := []Finding{
		{VulnerabilityType: "unchecked_call", Severity: "medium", Confidence: 0.8, AffectedFunction: "pay", LineStart: 10, LineEnd: 12, Title: "A"},
		{VulnerabilityType: "unchecked_call", Severity: "medium", Confidence: 0.8, AffectedFunction: "pay", LineStart: 40, LineEnd: 42, Title: "B"},
	}

	if merged := Deduplicate(findings); len(merged) != 2 {
		t.Errorf("merged = %d, want 2", len(merged))
	}
}

func TestDeduplicateKeepsDifferentFunctions(t *testing.T) {
	findings := []Finding{
		{VulnerabilityType: "access_control", Severity: "critical", Confidence: 0.9, AffectedFunction: "mint", LineStart: 10, LineEnd: 12, Title: "Mint"},
		{VulnerabilityType: "access_control", Severity: "critical", Confidence: 0.9, AffectedFunction: "burn", LineStart: 10, LineEnd: 12, Title: "Burn"},
	}

	if merged := Deduplicate(findings); len(merged) != 2 {
		t.Errorf("merged = %d, want 2", len(merged))
	}
}

func TestSortFindingsBySeverityThenConfidence(t *testing.T) {
	findings := []Finding{
		{Severity: "high", Confidence: 0.9, Title: "high"},
		{Severity: "critical", Confidence: 0.7, Title: "critical-low"},
		{Severity: "critical", Confidence: 0.95, Title: "critical-high"},
		{Severity: "informational", Confidence: 0.99, Title: "info"},
	}

	sorted := SortFindings(findings)
	titles := []string{sorted[0].Title, sorted[1].Title, sorted[2].Title, sorted[3].Title}
	want := []string{"critical-high", "critical-low", "high", "info"}

	for index := range want {
		if titles[index] != want[index] {
			t.Fatalf("order = %v, want %v", titles, want)
		}
	}
}

func TestSummaryAndConfidenceScore(t *testing.T) {
	findings := []Finding{
		{Severity: "critical", Confidence: 0.9},
		{Severity: "high", Confidence: 0.6},
		{Severity: "medium", Confidence: 0.75},
		{Severity: "low", Confidence: 1},
		{Severity: "informational", Confidence: 0.5},
	}

	summary := Summarize(findings)
	if summary.TotalFindings != 5 || summary.Critical != 1 || summary.High != 1 ||
		summary.Medium != 1 || summary.Low != 1 || summary.Informational != 1 {
		t.Errorf("summary = %+v", summary)
	}

	// (0.9*5 + 0.6*4 + 0.75*3 + 1*2 + 0.5*1) / 15
	want := float32((0.9*5 + 0.6*4 + 0.75*3 + 1*2 + 0.5*1) / 15)
	if diff := summary.ConfidenceScore - want; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("confidence = %v, want %v", summary.ConfidenceScore, want)
	}

	if score := ConfidenceScore(nil); score != 0 {
		t.Errorf("empty confidence = %v, want 0", score)
	}
}

func TestGenerateEmptyReportHasArrayFindings(t *testing.T) {
	report := Generate(GenerateInput{
		AuditID:       "audit-1",
		ContractName:  "SafeVault",
		LLMModel:      "mock-auditor",
		AnalysisDepth: "standard",
		Warnings:      []string{"rag unavailable"},
	})

	if report.Findings == nil {
		t.Error("Findings must be an empty slice, not nil")
	}
	if report.Summary.TotalFindings != 0 || report.Summary.ConfidenceScore != 0 {
		t.Errorf("summary = %+v", report.Summary)
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"findings":[]`) {
		t.Errorf("JSON findings not an empty array: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"warnings":["rag unavailable"]`) {
		t.Errorf("warnings missing: %s", encoded)
	}
}
