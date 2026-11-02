package parser

import (
	"fmt"
	"strings"
	"testing"
)

func TestChunkProducesOneChunkPerFunction(t *testing.T) {
	contract, err := Parse(readContract(t, "safe/safe_vault.sol"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	chunks := Chunk(contract)
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}

	withdraw := findChunk(t, chunks, "withdraw")
	if withdraw.ContractName != "SafeVault" {
		t.Errorf("ContractName = %q", withdraw.ContractName)
	}
	wantContext := "Contract state variables:\n- mapping(address => uint256) balances (public)\n\nInherits from:\n- none"
	if withdraw.ContextSummary != wantContext {
		t.Errorf("ContextSummary =\n%q\nwant\n%q", withdraw.ContextSummary, wantContext)
	}
	if withdraw.LineStart != 11 || withdraw.LineEnd != 19 {
		t.Errorf("withdraw lines = %d-%d, want 11-19", withdraw.LineStart, withdraw.LineEnd)
	}
	if !strings.Contains(withdraw.Code, "balances[msg.sender] = 0;") {
		t.Error("chunk code missing the state reset")
	}
}

func TestChunkIncludesInheritanceInContext(t *testing.T) {
	source := `pragma solidity ^0.8.0;

contract Child is Base, ReentrancyGuard {
    uint256 private counter;

    function increment() external {
        counter++;
    }
}
`
	contract, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	chunks := Chunk(contract)
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1", len(chunks))
	}
	for _, parent := range []string{"Base", "ReentrancyGuard", "uint256 counter (private)"} {
		if !strings.Contains(chunks[0].ContextSummary, parent) {
			t.Errorf("context missing %q:\n%s", parent, chunks[0].ContextSummary)
		}
	}
}

func TestChunkSplitsOversizedFunctions(t *testing.T) {
	var source strings.Builder
	source.WriteString("pragma solidity ^0.8.0;\ncontract Big {\n    function run() external {\n")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&source, "        total = total + %d;\n", i)
	}
	source.WriteString("    }\n}\n")

	contract, err := Parse(source.String())
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(contract.Functions) != 1 {
		t.Fatalf("Functions = %d", len(contract.Functions))
	}

	chunks := chunkWithLimit(contract, 500)
	if len(chunks) < 3 {
		t.Fatalf("chunks = %d, want at least 3", len(chunks))
	}

	for index, chunk := range chunks {
		if len(chunk.Code) > 500 {
			t.Errorf("chunk %d has %d chars, limit 500", index, len(chunk.Code))
		}
		if chunk.LineStart > chunk.LineEnd {
			t.Errorf("chunk %d has inverted lines %d-%d", index, chunk.LineStart, chunk.LineEnd)
		}
		if !strings.Contains(chunk.ContextSummary, "part") {
			t.Errorf("chunk %d context missing excerpt marker", index)
		}
		if index > 0 && chunk.LineStart <= chunks[index-1].LineEnd {
			t.Errorf("chunk %d starts at %d, previous ended at %d", index, chunk.LineStart, chunks[index-1].LineEnd)
		}
	}
}

func TestChunkEmptyContract(t *testing.T) {
	contract, err := Parse("pragma solidity ^0.8.0;\ncontract Empty { uint256 x; }")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if chunks := Chunk(contract); len(chunks) != 0 {
		t.Errorf("chunks = %+v, want none", chunks)
	}
}

func findChunk(t *testing.T, chunks []CodeChunk, name string) CodeChunk {
	t.Helper()
	for _, chunk := range chunks {
		if chunk.FunctionName == name {
			return chunk
		}
	}
	t.Fatalf("chunk %q not found", name)
	return CodeChunk{}
}
