package parser

import (
	"fmt"
	"strings"
)

// DefaultMaxChunkChars is the maximum size of a single analysis chunk.
const DefaultMaxChunkChars = 4000

// Chunk converts a parsed contract into one code chunk per function. Functions
// larger than DefaultMaxChunkChars are split into sequential line-based blocks.
func Chunk(contract *ParsedContract) []CodeChunk {
	return chunkWithLimit(contract, DefaultMaxChunkChars)
}

func chunkWithLimit(contract *ParsedContract, maxChars int) []CodeChunk {
	if contract == nil {
		return []CodeChunk{}
	}
	if maxChars <= 0 {
		maxChars = DefaultMaxChunkChars
	}

	context := buildContextSummary(contract)
	chunks := make([]CodeChunk, 0, len(contract.Functions))

	for _, function := range contract.Functions {
		if len(function.Body) <= maxChars {
			chunks = append(chunks, CodeChunk{
				ContractName:   contract.Name,
				FunctionName:   function.Name,
				Code:           function.Body,
				ContextSummary: context,
				LineStart:      function.LineStart,
				LineEnd:        function.LineEnd,
			})
			continue
		}

		parts := splitFunction(function, maxChars)
		for index, part := range parts {
			chunks = append(chunks, CodeChunk{
				ContractName: contract.Name,
				FunctionName: function.Name,
				Code:         part.code,
				ContextSummary: fmt.Sprintf(
					"%s\n\nFunction excerpt: part %d of %d. The excerpt starts inside %s().",
					context, index+1, len(parts), function.Name,
				),
				LineStart: part.lineStart,
				LineEnd:   part.lineEnd,
			})
		}
	}

	return chunks
}

func buildContextSummary(contract *ParsedContract) string {
	var builder strings.Builder

	builder.WriteString("Contract state variables:\n")
	if len(contract.StateVariables) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, variable := range contract.StateVariables {
			fmt.Fprintf(&builder, "- %s %s (%s)\n", variable.Type, variable.Name, variable.Visibility)
		}
	}

	builder.WriteString("\nInherits from:\n")
	if len(contract.Inheritance) == 0 {
		builder.WriteString("- none")
	} else {
		for index, parent := range contract.Inheritance {
			if index > 0 {
				builder.WriteString("\n")
			}
			fmt.Fprintf(&builder, "- %s", parent)
		}
	}

	return builder.String()
}

type chunkPart struct {
	code      string
	lineStart int
	lineEnd   int
}

// splitFunction divides oversized function source at line boundaries while
// keeping accurate absolute line numbers for each part.
func splitFunction(function FunctionDef, maxChars int) []chunkPart {
	lines := strings.Split(function.Body, "\n")
	parts := []chunkPart{}

	var builder strings.Builder
	partStart := function.LineStart
	currentLine := function.LineStart

	flush := func(endLine int) {
		code := strings.TrimRight(builder.String(), "\n")
		if code == "" {
			return
		}
		parts = append(parts, chunkPart{
			code:      code,
			lineStart: partStart,
			lineEnd:   endLine,
		})
		builder.Reset()
	}

	for _, line := range lines {
		lineLength := len(line) + 1

		if lineLength > maxChars {
			flush(currentLine - 1)
			for start := 0; start < len(line); start += maxChars {
				end := start + maxChars
				if end > len(line) {
					end = len(line)
				}
				parts = append(parts, chunkPart{
					code:      line[start:end],
					lineStart: currentLine,
					lineEnd:   currentLine,
				})
			}
			currentLine++
			partStart = currentLine
			continue
		}

		if builder.Len() > 0 && builder.Len()+lineLength > maxChars {
			flush(currentLine - 1)
			partStart = currentLine
		}

		builder.WriteString(line)
		builder.WriteString("\n")
		currentLine++
	}

	flush(currentLine - 1)

	if len(parts) == 0 {
		parts = append(parts, chunkPart{
			code:      function.Body,
			lineStart: function.LineStart,
			lineEnd:   function.LineEnd,
		})
	}

	return parts
}
