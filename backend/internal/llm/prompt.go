package llm

import (
	"fmt"
	"strings"

	"chainmind/backend/internal/rag"
)

// SystemPrompt instructs the model to behave as a conservative auditor that
// only emits strict JSON.
const SystemPrompt = `You are a senior smart contract security auditor.

Your task is to analyze Solidity code for security vulnerabilities.

Rules:
1. Only report vulnerabilities that are visible in the provided code.
2. Do not speculate about missing code.
3. Cross-reference the provided known exploit patterns.
4. Return only valid JSON.
5. If no vulnerability is found, return an empty findings array.
6. For each finding, include severity, confidence, affected function, line range, description, and recommended fix.
7. Be conservative. Avoid false positives.`

const outputSchema = `{
  "findings": [
    {
      "vulnerability_type": "string",
      "severity": "critical|high|medium|low|informational",
      "confidence": 0.0,
      "affected_function": "string",
      "line_start": 0,
      "line_end": 0,
      "title": "string",
      "description": "string",
      "vulnerable_code": "string",
      "recommended_fix": "string",
      "fixed_code": "string",
      "matched_exploit_pattern": "string|null"
    }
  ]
}`

// BuildPrompt renders the system and user prompts for one chunk.
func BuildPrompt(request AnalysisRequest) (string, string) {
	var builder strings.Builder

	builder.WriteString("## Contract Code\n\n")
	fmt.Fprintf(&builder, "Contract Name: %s\n", request.Chunk.ContractName)
	fmt.Fprintf(&builder, "Function Name: %s\n", request.Chunk.FunctionName)
	fmt.Fprintf(&builder, "Function starts at file line: %d\n", request.Chunk.LineStart)
	fmt.Fprintf(&builder, "Analysis depth: %s\n\n", defaultDepth(request.Depth))
	fmt.Fprintf(&builder, "```solidity\n%s\n```\n\n", request.Chunk.Code)

	builder.WriteString("## Contract Context\n\n")
	builder.WriteString(request.Chunk.ContextSummary)
	builder.WriteString("\n\n")

	builder.WriteString("## Known Exploit Patterns\n\n")
	builder.WriteString(formatPatterns(request.Patterns))
	builder.WriteString("\n\n")

	builder.WriteString("## Required Output\n\n")
	builder.WriteString("Return JSON using this schema. Use absolute line numbers from the original file.\n\n")
	builder.WriteString(outputSchema)

	return SystemPrompt, builder.String()
}

func defaultDepth(depth string) string {
	switch depth {
	case "quick", "standard", "deep":
		return depth
	default:
		return "standard"
	}
}

func formatPatterns(patterns []rag.RetrievalResult) string {
	if len(patterns) == 0 {
		return "No matching exploit patterns were retrieved. Analyze the code on its own merits."
	}

	var builder strings.Builder
	for index, result := range patterns {
		pattern := result.Pattern
		fmt.Fprintf(&builder, "### %d. %s (%s, %s)\n", index+1, pattern.Name, pattern.Category, pattern.Severity)
		if pattern.SWCID != "" {
			fmt.Fprintf(&builder, "SWC: %s\n", pattern.SWCID)
		}
		if pattern.Description != "" {
			fmt.Fprintf(&builder, "Description: %s\n", pattern.Description)
		}
		if pattern.RealWorldCase != "" {
			fmt.Fprintf(&builder, "Real-world case: %s\n", pattern.RealWorldCase)
		}
		if pattern.VulnerableCode != "" {
			fmt.Fprintf(&builder, "Vulnerable example:\n```solidity\n%s\n```\n", pattern.VulnerableCode)
		}
		if pattern.FixedCode != "" {
			fmt.Fprintf(&builder, "Fixed example:\n```solidity\n%s\n```\n", pattern.FixedCode)
		}
		if len(pattern.References) > 0 {
			fmt.Fprintf(&builder, "References: %s\n", strings.Join(pattern.References, ", "))
		}
		if index < len(patterns)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
