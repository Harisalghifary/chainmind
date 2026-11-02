package validator

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateSource(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantErr error
	}{
		{
			name:   "valid contract",
			source: "pragma solidity ^0.8.0;\ncontract Vault { uint256 x; }",
		},
		{
			name:   "contract without pragma",
			source: "contract Vault { function run() external {} }",
		},
		{
			name:    "empty",
			source:  "   \n\t ",
			wantErr: ErrEmptySource,
		},
		{
			name:    "not solidity",
			source:  "hello world",
			wantErr: ErrNotSolidity,
		},
		{
			name:    "too large",
			source:  "contract C { " + strings.Repeat("// padding\n", 10_001) + " }",
			wantErr: ErrTooLarge,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateSource(testCase.source)
			if testCase.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateSource() error = %v", err)
				}
				return
			}
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
