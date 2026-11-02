package llm

import (
	"errors"
	"testing"
)

func TestParseResponseValid(t *testing.T) {
	raw := `{
		"findings": [
			{
				"vulnerability_type": "Reentrancy",
				"severity": "critical",
				"confidence": 1.7,
				"affected_function": "withdraw",
				"line_start": 12,
				"line_end": 16,
				"title": "Reentrancy in withdraw()",
				"description": "External call before state update.",
				"vulnerable_code": "msg.sender.call{value: amount}(\"\");",
				"recommended_fix": "Update state first.",
				"fixed_code": "balances[msg.sender] = 0;",
				"matched_exploit_pattern": "Reentrancy via external call before state update"
			}
		],
		"model": "claude-test",
		"tokens_used": 120
	}`

	response, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(response.Findings) != 1 {
		t.Fatalf("findings = %+v", response.Findings)
	}

	finding := response.Findings[0]
	if finding.VulnerabilityType != "reentrancy" {
		t.Errorf("type = %q", finding.VulnerabilityType)
	}
	if finding.Confidence != 1 {
		t.Errorf("confidence = %v, want clamped to 1", finding.Confidence)
	}
	if response.Model != "claude-test" || response.TokensUsed != 120 {
		t.Errorf("metadata = %s/%d", response.Model, response.TokensUsed)
	}
}

func TestParseResponseMarkdownFence(t *testing.T) {
	raw := "```json\n{\"findings\": [], \"model\": \"gpt-test\"}\n```"

	response, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if response.Findings == nil {
		t.Error("findings should be an empty slice, not nil")
	}
	if response.Model != "gpt-test" {
		t.Errorf("model = %q", response.Model)
	}
}

func TestParseResponseSurroundingProse(t *testing.T) {
	raw := `Here is the audit result:
{"findings": [{"vulnerability_type":"oracle_manipulation","severity":"high","confidence":0.8,"affected_function":"getPrice","line_start":5,"line_end":8,"title":"Oracle","description":"Single feed.","vulnerable_code":"price = feed.latestAnswer();","recommended_fix":"Use a TWAP.","fixed_code":"price = twap();","matched_exploit_pattern":null}]}
Thanks.`

	response, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(response.Findings) != 1 || response.Findings[0].MatchedPattern != "" {
		t.Errorf("findings = %+v", response.Findings)
	}
}

func TestParseResponseMalformedJSON(t *testing.T) {
	_, err := ParseResponse(`{"findings": [`)
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestParseResponseNoJSON(t *testing.T) {
	_, err := ParseResponse("I could not find any issues.")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestParseResponseDropsInvalidFindings(t *testing.T) {
	raw := `{"findings": [
		{"vulnerability_type":"reentrancy","severity":"severe","confidence":0.9,"affected_function":"withdraw","title":"Bad severity","description":"x"},
		{"vulnerability_type":"","severity":"high","confidence":0.9,"affected_function":"withdraw","title":"No type","description":"x"},
		{"vulnerability_type":"access_control","severity":"critical","confidence":0.9,"affected_function":"mint","title":"Valid","description":"Missing guard.","line_start":8,"line_end":4}
	]}`

	response, err := ParseResponse(raw)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(response.Findings) != 1 || response.Findings[0].Title != "Valid" {
		t.Fatalf("findings = %+v", response.Findings)
	}
	if response.Findings[0].LineEnd != 8 {
		t.Errorf("LineEnd = %d, want 8 (normalized to LineStart)", response.Findings[0].LineEnd)
	}
}

func TestParseResponseRejectsAllInvalidFindings(t *testing.T) {
	raw := `{"findings": [{"vulnerability_type":"","severity":"nope","confidence":2,"affected_function":"","title":"","description":""}]}`

	_, err := ParseResponse(raw)
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want ErrInvalidResponse", err)
	}
}
