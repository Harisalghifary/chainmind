package llm

import (
	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/rag"
)

// AnalysisRequest carries one chunk plus the retrieved exploit context.
type AnalysisRequest struct {
	Chunk    parser.CodeChunk      `json:"chunk"`
	Patterns []rag.RetrievalResult `json:"patterns"`
	Depth    string                `json:"depth"`
}

// LLMFinding is the strict JSON object returned by a provider.
type LLMFinding struct {
	VulnerabilityType string  `json:"vulnerability_type"`
	Severity          string  `json:"severity"`
	Confidence        float32 `json:"confidence"`
	AffectedFunction  string  `json:"affected_function"`
	LineStart         int     `json:"line_start"`
	LineEnd           int     `json:"line_end"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	VulnerableCode    string  `json:"vulnerable_code"`
	RecommendedFix    string  `json:"recommended_fix"`
	FixedCode         string  `json:"fixed_code"`
	MatchedPattern    string  `json:"matched_exploit_pattern"`
}

// AnalysisResponse is a validated provider response.
type AnalysisResponse struct {
	Findings   []LLMFinding `json:"findings"`
	Model      string       `json:"model"`
	TokensUsed int          `json:"tokens_used"`
}
