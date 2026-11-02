package report

import (
	"strings"
)

// Deduplicate merges findings that describe the same issue in the same
// function with overlapping or near-adjacent line ranges. The finding with the
// higher confidence is kept and enriched with references and fixes from the
// duplicate.
func Deduplicate(findings []Finding) []Finding {
	result := make([]Finding, 0, len(findings))

	for _, finding := range findings {
		matched := -1
		for index := range result {
			if sameIssue(result[index], finding) {
				matched = index
				break
			}
		}
		if matched == -1 {
			result = append(result, normalizeFinding(finding))
			continue
		}
		result[matched] = mergeFindings(result[matched], normalizeFinding(finding))
	}

	return result
}

func sameIssue(a, b Finding) bool {
	if !strings.EqualFold(a.VulnerabilityType, b.VulnerabilityType) {
		return false
	}
	if !strings.EqualFold(a.AffectedFunction, b.AffectedFunction) {
		return false
	}
	if a.LineStart == 0 || a.LineEnd == 0 || b.LineStart == 0 || b.LineEnd == 0 {
		return true
	}

	const tolerance = 3
	return a.LineStart <= b.LineEnd+tolerance && b.LineStart <= a.LineEnd+tolerance
}

func mergeFindings(keep, duplicate Finding) Finding {
	if betterFinding(duplicate, keep) {
		keep, duplicate = duplicate, keep
	}

	keep.References = mergeReferences(keep.References, duplicate.References)
	keep.Confidence = maxFloat32(keep.Confidence, duplicate.Confidence)

	if keep.VulnerableCode == "" {
		keep.VulnerableCode = duplicate.VulnerableCode
	}
	if keep.RecommendedFix == "" {
		keep.RecommendedFix = duplicate.RecommendedFix
	}
	if keep.FixedCode == "" {
		keep.FixedCode = duplicate.FixedCode
	}
	if keep.Description == "" {
		keep.Description = duplicate.Description
	}

	if keep.MatchedPattern == nil {
		keep.MatchedPattern = duplicate.MatchedPattern
		keep.MatchedPatternID = duplicate.MatchedPatternID
	} else if duplicate.MatchedPattern != nil &&
		duplicate.MatchedPattern.SimilarityScore > keep.MatchedPattern.SimilarityScore {
		keep.MatchedPattern = duplicate.MatchedPattern
		keep.MatchedPatternID = duplicate.MatchedPatternID
	}

	if keep.LineStart == 0 || (duplicate.LineStart > 0 && duplicate.LineStart < keep.LineStart) {
		keep.LineStart = duplicate.LineStart
	}
	if duplicate.LineEnd > keep.LineEnd {
		keep.LineEnd = duplicate.LineEnd
	}

	return keep
}

func betterFinding(candidate, current Finding) bool {
	if candidate.Confidence != current.Confidence {
		return candidate.Confidence > current.Confidence
	}
	return severityRank(candidate.Severity) > severityRank(current.Severity)
}

func mergeReferences(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	merged := make([]string, 0, len(a)+len(b))

	for _, group := range [][]string{a, b} {
		for _, reference := range group {
			reference = strings.TrimSpace(reference)
			if reference == "" {
				continue
			}
			if _, exists := seen[reference]; exists {
				continue
			}
			seen[reference] = struct{}{}
			merged = append(merged, reference)
		}
	}

	return merged
}

func normalizeFinding(finding Finding) Finding {
	finding.VulnerabilityType = strings.ToLower(strings.TrimSpace(finding.VulnerabilityType))
	finding.Severity = strings.ToLower(strings.TrimSpace(finding.Severity))
	if finding.References == nil {
		finding.References = []string{}
	}
	return finding
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
