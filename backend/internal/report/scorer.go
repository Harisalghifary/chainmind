package report

import "sort"

var severityWeights = map[string]int{
	"critical":      5,
	"high":          4,
	"medium":        3,
	"low":           2,
	"informational": 1,
}

func severityRank(severity string) int {
	if rank, ok := severityWeights[severity]; ok {
		return rank
	}
	return 0
}

// SortFindings orders findings from most to least severe, then by confidence.
func SortFindings(findings []Finding) []Finding {
	sorted := make([]Finding, len(findings))
	copy(sorted, findings)

	sort.SliceStable(sorted, func(i, j int) bool {
		left, right := sorted[i], sorted[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if left.Confidence != right.Confidence {
			return left.Confidence > right.Confidence
		}
		return left.Title < right.Title
	})

	return sorted
}

// ConfidenceScore is the severity-weighted average finding confidence. A report
// with no findings has a score of 0.
func ConfidenceScore(findings []Finding) float32 {
	if len(findings) == 0 {
		return 0
	}

	var weighted, totalWeight float64
	for _, finding := range findings {
		weight := float64(severityRank(finding.Severity))
		if weight == 0 {
			weight = 1
		}
		weighted += float64(finding.Confidence) * weight
		totalWeight += weight
	}
	if totalWeight == 0 {
		return 0
	}
	return float32(weighted / totalWeight)
}

// Summarize counts findings per severity and computes the confidence score.
func Summarize(findings []Finding) Summary {
	summary := Summary{TotalFindings: len(findings)}

	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			summary.Critical++
		case "high":
			summary.High++
		case "medium":
			summary.Medium++
		case "low":
			summary.Low++
		case "informational":
			summary.Informational++
		}
	}

	summary.ConfidenceScore = ConfidenceScore(findings)
	return summary
}
