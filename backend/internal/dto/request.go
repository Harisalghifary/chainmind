package dto

import (
	"strings"
	"unicode/utf8"

	apperrors "chainmind/backend/pkg/errors"
)

const (
	MaxContractNameChars = 255
	MaxSourceCodeChars   = 100_000
)

// CreateAuditRequest is the POST /api/v1/audit payload.
type CreateAuditRequest struct {
	ContractName    string `json:"contract_name"`
	SourceCode      string `json:"source_code"`
	SolidityVersion string `json:"solidity_version"`
	AnalysisDepth   string `json:"analysis_depth"`
}

// Normalize trims input and applies defaults.
func (r *CreateAuditRequest) Normalize() {
	r.ContractName = strings.TrimSpace(r.ContractName)
	r.SolidityVersion = strings.TrimSpace(r.SolidityVersion)
	r.AnalysisDepth = strings.ToLower(strings.TrimSpace(r.AnalysisDepth))
	if r.AnalysisDepth == "" {
		r.AnalysisDepth = "standard"
	}
}

// Validate returns a structured INVALID_INPUT error when the payload does not
// satisfy the API contract.
func (r *CreateAuditRequest) Validate() error {
	if r.ContractName == "" {
		return apperrors.InvalidInput("contract_name is required")
	}
	if utf8.RuneCountInString(r.ContractName) > MaxContractNameChars {
		return apperrors.InvalidInput("contract_name must be at most 255 characters")
	}
	if strings.TrimSpace(r.SourceCode) == "" {
		return apperrors.InvalidInput("source_code is required")
	}
	if utf8.RuneCountInString(r.SourceCode) > MaxSourceCodeChars {
		return apperrors.InvalidInput("source_code must be at most 100000 characters")
	}

	switch r.AnalysisDepth {
	case "quick", "standard", "deep":
	default:
		return apperrors.InvalidInput("analysis_depth must be one of quick, standard, deep")
	}

	return nil
}
