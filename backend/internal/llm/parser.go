package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// ErrInvalidResponse marks a provider answer that is not usable JSON.
var ErrInvalidResponse = errors.New("llm returned an invalid analysis response")

var validSeverities = map[string]struct{}{
	"critical":      {},
	"high":          {},
	"medium":        {},
	"low":           {},
	"informational": {},
}

// ParseResponse extracts, validates and normalizes the JSON object returned by
// a provider. Findings with an unusable shape are dropped; a response whose
// findings are all invalid is treated as an invalid response.
func ParseResponse(raw string) (*AnalysisResponse, error) {
	jsonText, err := extractJSONObject(raw)
	if err != nil {
		return nil, err
	}

	var response AnalysisResponse
	if err := json.Unmarshal([]byte(jsonText), &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	findings := make([]LLMFinding, 0, len(response.Findings))
	for _, finding := range response.Findings {
		if validated, ok := validateFinding(finding); ok {
			findings = append(findings, validated)
		}
	}

	if len(response.Findings) > 0 && len(findings) == 0 {
		return nil, fmt.Errorf("%w: every finding failed schema validation", ErrInvalidResponse)
	}

	response.Findings = findings
	return &response, nil
}

func extractJSONObject(raw string) (string, error) {
	text := strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
	if text == "" {
		return "", fmt.Errorf("%w: empty response", ErrInvalidResponse)
	}

	if strings.HasPrefix(text, "```") {
		if newline := strings.Index(text, "\n"); newline >= 0 {
			text = text[newline+1:]
		}
		if fence := strings.LastIndex(text, "```"); fence >= 0 {
			text = text[:fence]
		}
		text = strings.TrimSpace(text)
	}

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("%w: no JSON object found", ErrInvalidResponse)
	}

	return text[start : end+1], nil
}

func validateFinding(finding LLMFinding) (LLMFinding, bool) {
	finding.VulnerabilityType = strings.ToLower(strings.TrimSpace(finding.VulnerabilityType))
	finding.Severity = strings.ToLower(strings.TrimSpace(finding.Severity))
	finding.AffectedFunction = strings.TrimSpace(finding.AffectedFunction)
	finding.Title = strings.TrimSpace(finding.Title)
	finding.Description = strings.TrimSpace(finding.Description)
	finding.MatchedPattern = strings.TrimSpace(finding.MatchedPattern)

	if finding.VulnerabilityType == "" || finding.AffectedFunction == "" || finding.Title == "" {
		return LLMFinding{}, false
	}
	if _, ok := validSeverities[finding.Severity]; !ok {
		return LLMFinding{}, false
	}
	if finding.Description == "" {
		return LLMFinding{}, false
	}

	finding.Confidence = float32(math.Max(0, math.Min(1, float64(finding.Confidence))))
	if finding.LineStart < 0 {
		finding.LineStart = 0
	}
	if finding.LineEnd < finding.LineStart {
		finding.LineEnd = finding.LineStart
	}

	return finding, true
}
