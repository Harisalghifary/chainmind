// Package validator performs inexpensive Solidity source checks before the
// parser and audit worker are invoked.
package validator

import (
	"errors"
	"strings"
)

const MaxSourceChars = 100_000

var (
	// ErrEmptySource is returned for blank input.
	ErrEmptySource = errors.New("source code is empty")
	// ErrTooLarge is returned when the source exceeds the API limit.
	ErrTooLarge = errors.New("source code exceeds the 100000 character limit")
	// ErrNotSolidity is returned when no Solidity declaration is present.
	ErrNotSolidity = errors.New("source does not look like a Solidity contract")
)

// ValidateSource checks that the contract is non-empty, within the size limit
// and contains at least one Solidity declaration. It is intentionally shallow;
// parser.Parse performs the structural validation.
func ValidateSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return ErrEmptySource
	}
	if len([]rune(source)) > MaxSourceChars {
		return ErrTooLarge
	}

	lower := strings.ToLower(source)
	for _, marker := range []string{
		"pragma solidity",
		"contract ",
		"library ",
		"interface ",
	} {
		if strings.Contains(lower, marker) {
			return nil
		}
	}

	return ErrNotSolidity
}
