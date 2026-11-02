package parser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readContract(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func TestParseVulnerableVault(t *testing.T) {
	contract, err := Parse(readContract(t, "vulnerable/reentrancy.sol"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if contract.Name != "VulnerableVault" {
		t.Errorf("Name = %q", contract.Name)
	}
	if contract.SolidityVersion != "^0.8.0" {
		t.Errorf("SolidityVersion = %q", contract.SolidityVersion)
	}
	if len(contract.Inheritance) != 0 {
		t.Errorf("Inheritance = %v", contract.Inheritance)
	}

	if len(contract.StateVariables) != 1 {
		t.Fatalf("StateVariables = %+v", contract.StateVariables)
	}
	if contract.StateVariables[0].Name != "balances" ||
		contract.StateVariables[0].Type != "mapping(address => uint256)" ||
		contract.StateVariables[0].Visibility != "public" {
		t.Errorf("state variable = %+v", contract.StateVariables[0])
	}

	if len(contract.Functions) != 2 {
		t.Fatalf("Functions = %d, want 2: %+v", len(contract.Functions), contract.Functions)
	}

	withdraw := findFunction(t, contract, "withdraw")
	if withdraw.Visibility != "external" {
		t.Errorf("withdraw visibility = %q", withdraw.Visibility)
	}
	if withdraw.LineStart != 11 || withdraw.LineEnd != 18 {
		t.Errorf("withdraw lines = %d-%d, want 11-18", withdraw.LineStart, withdraw.LineEnd)
	}
	if !strings.Contains(withdraw.Body, `msg.sender.call{value: amount}("")`) {
		t.Errorf("withdraw body missing external call:\n%s", withdraw.Body)
	}
	if !strings.Contains(withdraw.Body, "balances[msg.sender] = 0;") {
		t.Errorf("withdraw body missing state reset")
	}

	deposit := findFunction(t, contract, "deposit")
	if deposit.Mutability != "payable" {
		t.Errorf("deposit mutability = %q, want payable", deposit.Mutability)
	}
}

func TestParseAccessControl(t *testing.T) {
	contract, err := Parse(readContract(t, "vulnerable/access_control.sol"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if contract.Name != "UnprotectedAdmin" {
		t.Errorf("Name = %q", contract.Name)
	}
	if len(contract.Functions) != 3 {
		t.Fatalf("Functions = %d, want 3: %+v", len(contract.Functions), contract.Functions)
	}
	if findFunction(t, contract, "constructor").LineStart != 8 {
		t.Errorf("constructor line = %d, want 8", findFunction(t, contract, "constructor").LineStart)
	}
	if findFunction(t, contract, "destroy").Visibility != "external" {
		t.Error("destroy should be external")
	}
}

func TestParseSafeVault(t *testing.T) {
	contract, err := Parse(readContract(t, "safe/safe_vault.sol"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if contract.Name != "SafeVault" || len(contract.Functions) != 2 {
		t.Errorf("contract = %+v", contract)
	}
}

func TestParseImportsInheritanceAndStateVariables(t *testing.T) {
	source := `pragma solidity ^0.8.20;
import {IERC20} from "./IERC20.sol";
import "./Utils.sol";

abstract contract Vault is ERC20("Vault", "V"), Ownable {
    mapping(address => mapping(address => uint256)) private allowances;
    uint256 public constant FEE_BPS = 30;
    address internal immutable treasury;
    IERC20 public token;

    function fee() external view returns (uint256) {
        return FEE_BPS;
    }
}
`
	contract, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if contract.Name != "Vault" {
		t.Errorf("Name = %q", contract.Name)
	}
	if contract.SolidityVersion != "^0.8.20" {
		t.Errorf("SolidityVersion = %q", contract.SolidityVersion)
	}
	if len(contract.Imports) != 2 {
		t.Errorf("Imports = %v", contract.Imports)
	}
	if len(contract.Inheritance) != 2 || contract.Inheritance[0] != `ERC20("Vault", "V")` || contract.Inheritance[1] != "Ownable" {
		t.Errorf("Inheritance = %v", contract.Inheritance)
	}
	if len(contract.StateVariables) != 4 {
		t.Fatalf("StateVariables = %+v", contract.StateVariables)
	}

	allowances := contract.StateVariables[0]
	if allowances.Name != "allowances" || allowances.Visibility != "private" {
		t.Errorf("allowances = %+v", allowances)
	}
	if contract.StateVariables[1].Name != "FEE_BPS" {
		t.Errorf("FEE_BPS = %+v", contract.StateVariables[1])
	}
	if contract.StateVariables[2].Name != "treasury" {
		t.Errorf("treasury = %+v", contract.StateVariables[2])
	}

	if len(contract.Functions) != 1 || contract.Functions[0].Mutability != "view" {
		t.Errorf("Functions = %+v", contract.Functions)
	}
}

func TestParseSkipsInterfaceDeclarations(t *testing.T) {
	source := `pragma solidity ^0.8.0;

interface IToken {
    function transfer(address to, uint256 amount) external returns (bool);
}

contract Holder {
    function run() external {}
}
`
	contract, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if contract.Name != "IToken" {
		t.Fatalf("Name = %q, want IToken (first declaration)", contract.Name)
	}
	if len(contract.Functions) != 0 {
		t.Errorf("Functions = %+v, want none for the interface", contract.Functions)
	}
}

func TestParseRejectsEmptySource(t *testing.T) {
	if _, err := Parse("   \n"); !errors.Is(err, ErrEmptySource) {
		t.Errorf("error = %v, want ErrEmptySource", err)
	}
}

func TestParseRejectsNonContractSource(t *testing.T) {
	if _, err := Parse("pragma solidity ^0.8.0;\n// no contract here"); !errors.Is(err, ErrNoContract) {
		t.Errorf("error = %v, want ErrNoContract", err)
	}
}

func findFunction(t *testing.T, contract *ParsedContract, name string) FunctionDef {
	t.Helper()
	for _, function := range contract.Functions {
		if function.Name == name {
			return function
		}
	}
	t.Fatalf("function %q not found in %+v", name, contract.Functions)
	return FunctionDef{}
}
