package llm

import (
	"strings"
	"testing"

	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/rag"
)

func TestBuildPromptIncludesCodeContextAndPatterns(t *testing.T) {
	request := AnalysisRequest{
		Chunk: parser.CodeChunk{
			ContractName:   "VulnerableVault",
			FunctionName:   "withdraw",
			Code:           `function withdraw() external { msg.sender.call{value: 1}(""); }`,
			ContextSummary: "Contract state variables:\n- mapping(address => uint256) balances (public)",
			LineStart:      11,
			LineEnd:        18,
		},
		Patterns: []rag.RetrievalResult{{
			Pattern: rag.ExploitPattern{
				Name:           "Reentrancy via external call before state update",
				Category:       "reentrancy",
				SWCID:          "SWC-107",
				Severity:       "critical",
				Description:    "State is updated after the call.",
				VulnerableCode: "msg.sender.call{value: amount}(\"\");",
				RealWorldCase:  "The DAO Hack, 2016",
				References:     []string{"https://swcregistry.io/docs/SWC-107"},
			},
			SimilarityScore: 0.94,
		}},
		Depth: "standard",
	}

	system, user := BuildPrompt(request)

	if !strings.Contains(system, "senior smart contract security auditor") {
		t.Error("system prompt missing auditor role")
	}
	for _, expected := range []string{
		"VulnerableVault",
		"withdraw",
		"Function starts at file line: 11",
		"function withdraw() external",
		"Contract state variables:",
		"Reentrancy via external call before state update",
		"SWC-107",
		"The DAO Hack, 2016",
		"confidence",
		"fixed_code",
		"Analysis depth: standard",
	} {
		if !strings.Contains(user, expected) {
			t.Errorf("user prompt missing %q", expected)
		}
	}
}

func TestBuildPromptWithoutPatterns(t *testing.T) {
	_, user := BuildPrompt(AnalysisRequest{
		Chunk: parser.CodeChunk{ContractName: "Safe", FunctionName: "run"},
	})

	if !strings.Contains(user, "No matching exploit patterns were retrieved") {
		t.Error("missing no-pattern placeholder")
	}
	if !strings.Contains(user, "Analysis depth: standard") {
		t.Error("default depth should be standard")
	}
}
