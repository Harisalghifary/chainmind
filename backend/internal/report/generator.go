package report

// GenerateInput contains the persisted metadata and raw findings needed to
// build the final report.
type GenerateInput struct {
	AuditID          string
	ContractName     string
	LLMModel         string
	AnalysisDepth    string
	RAGPatternsUsed  int
	ProcessingTimeMS int64
	Warnings         []string
	Findings         []Finding
}

// Generate deduplicates, sorts and summarizes findings, then assembles the
// report returned by the API and stored for later reads.
func Generate(input GenerateInput) AuditReport {
	findings := SortFindings(Deduplicate(input.Findings))
	if findings == nil {
		findings = []Finding{}
	}
	for index := range findings {
		if findings[index].References == nil {
			findings[index].References = []string{}
		}
	}

	return AuditReport{
		AuditID:      input.AuditID,
		ContractName: input.ContractName,
		Summary:      Summarize(findings),
		Findings:     findings,
		Metadata: Metadata{
			LLMModel:         input.LLMModel,
			AnalysisDepth:    input.AnalysisDepth,
			RAGPatternsUsed:  input.RAGPatternsUsed,
			ProcessingTimeMS: input.ProcessingTimeMS,
			Warnings:         input.Warnings,
		},
	}
}
