package llm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"chainmind/backend/internal/parser"
)

func loadChunks(t *testing.T, path string) []parser.CodeChunk {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	contract, err := parser.Parse(string(content))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parser.Chunk(contract)
}

func chunkByName(t *testing.T, chunks []parser.CodeChunk, name string) parser.CodeChunk {
	t.Helper()
	for _, chunk := range chunks {
		if chunk.FunctionName == name {
			return chunk
		}
	}
	t.Fatalf("chunk %q not found", name)
	return parser.CodeChunk{}
}

func TestMockClientDetectsReentrancy(t *testing.T) {
	chunk := chunkByName(t, loadChunks(t, "vulnerable/reentrancy.sol"), "withdraw")

	response, err := NewMockClient().Analyze(context.Background(), AnalysisRequest{Chunk: chunk})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(response.Findings) != 1 {
		t.Fatalf("findings = %+v, want exactly one", response.Findings)
	}

	finding := response.Findings[0]
	if finding.VulnerabilityType != "reentrancy" || finding.Severity != "critical" {
		t.Errorf("finding = %+v", finding)
	}
	if finding.AffectedFunction != "withdraw" {
		t.Errorf("AffectedFunction = %q", finding.AffectedFunction)
	}
	if finding.LineStart != 14 {
		t.Errorf("LineStart = %d, want 14 (the external call line)", finding.LineStart)
	}
}

func TestMockClientDetectsAccessControl(t *testing.T) {
	chunks := loadChunks(t, "vulnerable/access_control.sol")

	for _, name := range []string{"mint", "destroy"} {
		chunk := chunkByName(t, chunks, name)
		response, err := NewMockClient().Analyze(context.Background(), AnalysisRequest{Chunk: chunk})
		if err != nil {
			t.Fatalf("Analyze(%s) error = %v", name, err)
		}
		if len(response.Findings) == 0 {
			t.Fatalf("%s: expected an access control finding", name)
		}
		if response.Findings[0].VulnerabilityType != "access_control" {
			t.Errorf("%s: type = %q", name, response.Findings[0].VulnerabilityType)
		}
	}
}

func TestMockClientLeavesSafeContractAlone(t *testing.T) {
	chunk := chunkByName(t, loadChunks(t, "safe/safe_vault.sol"), "withdraw")

	response, err := NewMockClient().Analyze(context.Background(), AnalysisRequest{Chunk: chunk})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(response.Findings) != 0 {
		t.Errorf("findings = %+v, want none", response.Findings)
	}
}

func TestMockClientDetectsUncheckedCall(t *testing.T) {
	chunk := parser.CodeChunk{
		ContractName: "Payout",
		FunctionName: "pay",
		Code:         "function pay() external { msg.sender.call{value: 1}(\"\"); }",
		LineStart:    10,
		LineEnd:      12,
	}

	response, err := NewMockClient().Analyze(context.Background(), AnalysisRequest{Chunk: chunk})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(response.Findings) != 1 || response.Findings[0].VulnerabilityType != "unchecked_call" {
		t.Errorf("findings = %+v", response.Findings)
	}
}
