package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrEmptySource is returned when no source code is supplied.
	ErrEmptySource = errors.New("source code is empty")
	// ErrNoContract is returned when the source has no parseable contract.
	ErrNoContract = errors.New("no Solidity contract found")
)

var (
	contractRe = regexp.MustCompile(`(?m)\b(?:abstract\s+)?(?:contract|library|interface)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:is\s+([^{]+))?\{`)
	pragmaRe   = regexp.MustCompile(`(?m)\bpragma\s+solidity\s+([^;]+);`)
	importRe   = regexp.MustCompile(`(?m)^\s*import\s+([^;]+);`)
	functionRe = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	ctorRe     = regexp.MustCompile(`\bconstructor\s*\(`)
	fallbackRe = regexp.MustCompile(`\bfallback\s*\(`)
	receiveRe  = regexp.MustCompile(`\breceive\s*\(`)

	stateVarRe = regexp.MustCompile(
		`^\s*((?:mapping\s*\(.*\))|(?:[A-Za-z_][A-Za-z0-9_]*(?:\s*\[\s*[0-9]*\s*\])?))\s+` +
			`(?:(public|private|internal|external)\s+)?` +
			`(?:(constant|immutable)\s+)?` +
			`([A-Za-z_][A-Za-z0-9_]*)\s*(?:=.*)?;$`,
	)

	visibilityRe = regexp.MustCompile(`\b(public|private|internal|external)\b`)
	mutabilityRe = regexp.MustCompile(`\b(pure|view|payable|nonpayable)\b`)
)

// Parse extracts the contract name, functions, state variables, imports,
// inheritance and Solidity version using a regex-based scanner. Full solc AST
// parsing is out of scope for the MVP.
func Parse(source string) (*ParsedContract, error) {
	if strings.TrimSpace(source) == "" {
		return nil, ErrEmptySource
	}

	source = strings.ReplaceAll(source, "\r\n", "\n")
	lines := newLineIndex(source)

	match := contractRe.FindStringSubmatchIndex(source)
	if match == nil {
		return nil, ErrNoContract
	}

	contract := &ParsedContract{
		Name:        source[match[2]:match[3]],
		Functions:   []FunctionDef{},
		Imports:     extractImports(source),
		Inheritance: []string{},
	}

	if match[4] >= 0 {
		contract.Inheritance = splitInheritance(source[match[4]:match[5]])
	}

	if pragma := pragmaRe.FindStringSubmatch(source); pragma != nil {
		contract.SolidityVersion = strings.TrimSpace(pragma[1])
	}

	openBrace := match[1] - 1
	closeBrace := findMatching(source, openBrace, '{', '}')
	if closeBrace < 0 {
		return nil, fmt.Errorf("%w: unbalanced braces after contract %s", ErrNoContract, contract.Name)
	}

	bodyStart := openBrace + 1
	contract.StateVariables = parseStateVariables(source[bodyStart:closeBrace], lines, bodyStart)
	contract.Functions = parseFunctions(source, lines, bodyStart, closeBrace)

	return contract, nil
}

func extractImports(source string) []string {
	imports := []string{}
	for _, match := range importRe.FindAllStringSubmatch(source, -1) {
		if value := strings.TrimSpace(match[1]); value != "" {
			imports = append(imports, value)
		}
	}
	return imports
}

// splitInheritance splits an inheritance list on top-level commas, keeping
// constructor arguments such as ERC20("Vault", "V") intact.
func splitInheritance(raw string) []string {
	parents := []string{}
	depth := 0
	start := 0

	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case '\'', '"':
			end := skipString(raw, i)
			if end < 0 {
				i = len(raw)
			} else {
				i = end
			}
		case ',':
			if depth == 0 {
				if parent := strings.TrimSpace(raw[start:i]); parent != "" {
					parents = append(parents, parent)
				}
				start = i + 1
			}
		}
	}

	if parent := strings.TrimSpace(raw[start:]); parent != "" {
		parents = append(parents, parent)
	}

	return parents
}

func parseStateVariables(body string, lines lineIndex, offset int) []StateVar {
	vars := []StateVar{}
	depth := 0

	forEachLine(body, func(line string, lineOffset int) {
		trimmed := strings.TrimSpace(line)
		if depth == 0 && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			if match := stateVarRe.FindStringSubmatch(line); match != nil {
				visibility := match[2]
				if visibility == "" {
					visibility = "internal"
				}
				vars = append(vars, StateVar{
					Type:       strings.TrimSpace(match[1]),
					Visibility: visibility,
					Name:       match[4],
				})
			}
		}
		depth += braceDelta(line)
		if depth < 0 {
			depth = 0
		}
		_ = lineOffset
	})

	return vars
}

func parseFunctions(source string, lines lineIndex, bodyStart, bodyEnd int) []FunctionDef {
	body := source[bodyStart:bodyEnd]
	functions := []FunctionDef{}

	for _, match := range functionRe.FindAllStringSubmatchIndex(body, -1) {
		name := body[match[2]:match[3]]
		if fn, ok := buildFunction(source, body, lines, bodyStart, match[0], name, match[1]-1); ok {
			functions = append(functions, fn)
		}
	}

	named := []struct {
		re   *regexp.Regexp
		name string
	}{
		{ctorRe, "constructor"},
		{fallbackRe, "fallback"},
		{receiveRe, "receive"},
	}

	for _, item := range named {
		for _, match := range item.re.FindAllStringIndex(body, -1) {
			if fn, ok := buildFunction(source, body, lines, bodyStart, match[0], item.name, match[1]-1); ok {
				functions = append(functions, fn)
			}
		}
	}

	sortFunctions(functions)
	return functions
}

// buildFunction assembles a FunctionDef from the opening parenthesis of its
// parameter list, returning ok=false for declarations without a body.
func buildFunction(source, body string, lines lineIndex, bodyStart, headerStart int, name string, openParen int) (FunctionDef, bool) {
	closeParen := findMatching(body, openParen, '(', ')')
	if closeParen < 0 || closeParen > len(body) {
		return FunctionDef{}, false
	}

	openBrace := -1
	semicolon := false
	for i := closeParen + 1; i < len(body); i++ {
		switch body[i] {
		case '{':
			openBrace = i
		case ';':
			semicolon = true
		}
		if openBrace >= 0 || semicolon {
			break
		}
	}
	if openBrace < 0 {
		return FunctionDef{}, false
	}

	closeBrace := findMatching(body, openBrace, '{', '}')
	if closeBrace < 0 {
		return FunctionDef{}, false
	}

	headerEnd := closeParen + 1
	modifiers := body[headerEnd:openBrace]

	visibility := ""
	if match := visibilityRe.FindStringSubmatch(modifiers); match != nil {
		visibility = match[1]
	}
	mutability := ""
	if match := mutabilityRe.FindStringSubmatch(modifiers); match != nil {
		mutability = match[1]
	}

	absoluteStart := bodyStart + headerStart
	absoluteEnd := bodyStart + closeBrace

	return FunctionDef{
		Name:       name,
		Visibility: visibility,
		Mutability: mutability,
		Params:     strings.TrimSpace(body[openParen+1 : closeParen]),
		Body:       body[headerStart : closeBrace+1],
		LineStart:  lines.lineAt(absoluteStart),
		LineEnd:    lines.lineAt(absoluteEnd),
	}, true
}

func sortFunctions(functions []FunctionDef) {
	// Insertion sort keeps constructor/receive ordering stable while making the
	// output follow source order.
	for i := 1; i < len(functions); i++ {
		current := functions[i]
		j := i - 1
		for j >= 0 && functions[j].LineStart > current.LineStart {
			functions[j+1] = functions[j]
			j--
		}
		functions[j+1] = current
	}
}

type lineIndex []int

func newLineIndex(source string) lineIndex {
	index := lineIndex{0}
	for i, char := range source {
		if char == '\n' {
			index = append(index, i+1)
		}
	}
	return index
}

// lineAt returns the 1-based line number containing the byte offset.
func (l lineIndex) lineAt(offset int) int {
	low, high := 0, len(l)-1
	for low < high {
		mid := (low + high + 1) / 2
		if l[mid] <= offset {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return low + 1
}

// findMatching returns the index of the delimiter matching the one at openIdx,
// ignoring delimiters inside strings and comments.
func findMatching(source string, openIdx int, open, close byte) int {
	if openIdx < 0 || openIdx >= len(source) || source[openIdx] != open {
		return -1
	}

	depth := 0
	inLineComment := false
	inBlockComment := false

	for i := openIdx; i < len(source); i++ {
		char := source[i]

		if inLineComment {
			if char == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if char == '*' && i+1 < len(source) && source[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		switch char {
		case '/':
			if i+1 < len(source) {
				if source[i+1] == '/' {
					inLineComment = true
					i++
					continue
				}
				if source[i+1] == '*' {
					inBlockComment = true
					i++
					continue
				}
			}
		case '\'', '"':
			end := skipString(source, i)
			if end < 0 {
				return -1
			}
			i = end
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

func skipString(source string, start int) int {
	quote := source[start]
	for i := start + 1; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == quote {
			return i
		}
	}
	return -1
}

// braceDelta counts net brace depth on a line, ignoring strings and comments.
func braceDelta(line string) int {
	delta := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '/':
			if i+1 < len(line) && line[i+1] == '/' {
				return delta
			}
		case '\'', '"':
			end := skipString(line, i)
			if end < 0 {
				return delta
			}
			i = end
		case '{':
			delta++
		case '}':
			delta--
		}
	}
	return delta
}

func forEachLine(text string, fn func(line string, offset int)) {
	offset := 0
	for _, line := range strings.Split(text, "\n") {
		fn(line, offset)
		offset += len(line) + 1
	}
}
