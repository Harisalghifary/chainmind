package dto

import (
	"strings"
	"testing"
)

func TestCreateAuditRequestNormalizeAndValidate(t *testing.T) {
	request := CreateAuditRequest{
		ContractName:  "  Vault  ",
		SourceCode:    "pragma solidity ^0.8.0;\ncontract Vault {}",
		AnalysisDepth: "  DEEP ",
	}
	request.Normalize()

	if request.ContractName != "Vault" {
		t.Errorf("contract name = %q", request.ContractName)
	}
	if request.AnalysisDepth != "deep" {
		t.Errorf("depth = %q", request.AnalysisDepth)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestCreateAuditRequestDefaultsDepth(t *testing.T) {
	request := CreateAuditRequest{
		ContractName: "Vault",
		SourceCode:   "contract Vault {}",
	}
	request.Normalize()

	if request.AnalysisDepth != "standard" {
		t.Errorf("depth = %q, want standard", request.AnalysisDepth)
	}
}

func TestCreateAuditRequestValidationErrors(t *testing.T) {
	valid := CreateAuditRequest{
		ContractName:  "Vault",
		SourceCode:    "contract Vault {}",
		AnalysisDepth: "standard",
	}

	cases := []struct {
		name    string
		mutate  func(*CreateAuditRequest)
		message string
	}{
		{
			name:    "missing name",
			mutate:  func(r *CreateAuditRequest) { r.ContractName = "" },
			message: "contract_name is required",
		},
		{
			name:    "name too long",
			mutate:  func(r *CreateAuditRequest) { r.ContractName = strings.Repeat("a", 256) },
			message: "contract_name must be at most 255 characters",
		},
		{
			name:    "missing source",
			mutate:  func(r *CreateAuditRequest) { r.SourceCode = "   " },
			message: "source_code is required",
		},
		{
			name:    "source too large",
			mutate:  func(r *CreateAuditRequest) { r.SourceCode = strings.Repeat("a", 100_001) },
			message: "source_code must be at most 100000 characters",
		},
		{
			name:    "invalid depth",
			mutate:  func(r *CreateAuditRequest) { r.AnalysisDepth = "thorough" },
			message: "analysis_depth must be one of quick, standard, deep",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := valid
			testCase.mutate(&request)

			err := request.Validate()
			if err == nil {
				t.Fatal("expected a validation error")
			}
			if err.Error() != "INVALID_INPUT: "+testCase.message {
				t.Errorf("error = %q", err.Error())
			}
		})
	}
}
