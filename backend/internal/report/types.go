package report

// AuditReport is the API representation of a finished audit.
type AuditReport struct {
	AuditID      string    `json:"audit_id"`
	ContractName string    `json:"contract_name"`
	Summary      Summary   `json:"summary"`
	Findings     []Finding `json:"findings"`
	Metadata     Metadata  `json:"metadata"`
}

// Summary aggregates finding counts by severity.
type Summary struct {
	TotalFindings   int     `json:"total_findings"`
	Critical        int     `json:"critical"`
	High            int     `json:"high"`
	Medium          int     `json:"medium"`
	Low             int     `json:"low"`
	Informational   int     `json:"informational"`
	ConfidenceScore float32 `json:"confidence_score"`
}

// Finding is one vulnerability in the final report.
type Finding struct {
	ID                string          `json:"id"`
	VulnerabilityType string          `json:"vulnerability_type"`
	Severity          string          `json:"severity"`
	Confidence        float32         `json:"confidence"`
	AffectedFunction  string          `json:"affected_function"`
	LineStart         int             `json:"line_start"`
	LineEnd           int             `json:"line_end"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	VulnerableCode    string          `json:"vulnerable_code"`
	RecommendedFix    string          `json:"recommended_fix"`
	FixedCode         string          `json:"fixed_code"`
	MatchedPattern    *MatchedPattern `json:"matched_exploit_pattern,omitempty"`
	References        []string        `json:"references"`
	MatchedPatternID  string          `json:"-"`
}

// MatchedPattern links a finding to a retrieved exploit pattern.
type MatchedPattern struct {
	Name            string  `json:"name"`
	RealWorldCase   string  `json:"real_world_case"`
	SimilarityScore float32 `json:"similarity_score"`
}

// Metadata describes how the report was produced.
type Metadata struct {
	LLMModel         string   `json:"llm_model"`
	AnalysisDepth    string   `json:"analysis_depth"`
	RAGPatternsUsed  int      `json:"rag_patterns_used"`
	ProcessingTimeMS int64    `json:"processing_time_ms"`
	Warnings         []string `json:"warnings,omitempty"`
}
