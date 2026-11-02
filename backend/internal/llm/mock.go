package llm

import (
	"context"
	"regexp"
	"strings"
)

// MockClient is a deterministic development-only provider. It performs a few
// shallow heuristics so the full pipeline and UI can be exercised without API
// keys; it is never selected in production and must not be treated as a real
// security analyzer.
type MockClient struct{}

func NewMockClient() *MockClient { return &MockClient{} }

func (m *MockClient) Name() string { return "mock" }

var (
	externalCallRe = regexp.MustCompile(`\.call\s*[\({]`)
	balanceResetRe = regexp.MustCompile(`balances\s*\[[^\]]+\]\s*=\s*0`)
	accessGuardRe  = regexp.MustCompile(`(?i)onlyOwner|hasRole|msg\.sender\s*==\s*owner|msg\.sender\s*!=\s*owner|_checkOwner`)
)

func (m *MockClient) Analyze(_ context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	code := request.Chunk.Code
	function := request.Chunk.FunctionName
	if function == "" {
		function = "unknown"
	}

	findings := make([]LLMFinding, 0, 3)

	call := externalCallRe.FindStringIndex(code)
	if call != nil {
		if reset := balanceResetRe.FindStringIndex(code); reset != nil && reset[0] > call[1] {
			findings = append(findings, LLMFinding{
				VulnerabilityType: "reentrancy",
				Severity:          "critical",
				Confidence:        0.95,
				AffectedFunction:  function,
				LineStart:         request.Chunk.LineStart + lineOffset(code, call[0]),
				LineEnd:           request.Chunk.LineEnd,
				Title:             "Reentrancy vulnerability in " + function + "()",
				Description:       "The function performs an external value call before resetting the caller balance, allowing the recipient to re-enter and withdraw again.",
				VulnerableCode:    code,
				RecommendedFix:    "Follow checks-effects-interactions: reset the balance before the external call and add a reentrancy guard.",
				FixedCode:         "balances[msg.sender] = 0;\n(bool success, ) = msg.sender.call{value: amount}(\"\");\nrequire(success, \"Transfer failed\");",
				MatchedPattern:    "Reentrancy via external call before state update",
			})
		}

		if !strings.Contains(code, "require(success") && !strings.Contains(code, "if (!success)") {
			findings = append(findings, LLMFinding{
				VulnerabilityType: "unchecked_call",
				Severity:          "medium",
				Confidence:        0.8,
				AffectedFunction:  function,
				LineStart:         request.Chunk.LineStart + lineOffset(code, call[0]),
				LineEnd:           request.Chunk.LineEnd,
				Title:             "Unchecked external call in " + function + "()",
				Description:       "The return value of the external call is ignored, so a failed transfer can be treated as a success.",
				VulnerableCode:    code,
				RecommendedFix:    "Capture the call result and revert when it is false, or use a safe transfer helper.",
				FixedCode:         "(bool success, ) = recipient.call{value: amount}(\"\");\nrequire(success, \"Transfer failed\");",
				MatchedPattern:    "Unchecked low-level call return value",
			})
		}
	}

	privileged := map[string]bool{
		"mint":      true,
		"destroy":   true,
		"burn":      true,
		"setOwner":  true,
		"upgradeTo": true,
	}
	if (privileged[function] || strings.Contains(code, "selfdestruct")) && !accessGuardRe.MatchString(code) {
		findings = append(findings, LLMFinding{
			VulnerabilityType: "access_control",
			Severity:          "critical",
			Confidence:        0.98,
			AffectedFunction:  function,
			LineStart:         request.Chunk.LineStart,
			LineEnd:           request.Chunk.LineEnd,
			Title:             "Missing access control on " + function + "()",
			Description:       "A privileged function can be called by any account because no owner or role check is enforced.",
			VulnerableCode:    code,
			RecommendedFix:    "Restrict the function with an onlyOwner modifier or an equivalent role check.",
			FixedCode:         "function " + function + "(uint256 amount) external onlyOwner {\n    // privileged logic\n}",
			MatchedPattern:    "Missing access control on privileged function",
		})
	}

	return &AnalysisResponse{
		Findings:   findings,
		Model:      "mock-auditor",
		TokensUsed: 0,
	}, nil
}

func lineOffset(code string, index int) int {
	if index < 0 {
		return 0
	}
	if index > len(code) {
		index = len(code)
	}
	return strings.Count(code[:index], "\n")
}
