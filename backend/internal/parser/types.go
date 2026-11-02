package parser

// ParsedContract is the structural summary extracted from Solidity source.
type ParsedContract struct {
	Name            string        `json:"name"`
	SolidityVersion string        `json:"solidity_version"`
	Functions       []FunctionDef `json:"functions"`
	StateVariables  []StateVar    `json:"state_variables"`
	Imports         []string      `json:"imports"`
	Inheritance     []string      `json:"inheritance"`
}

// FunctionDef describes a single function (or constructor/fallback/receive).
// Body holds the complete source of the declaration, including its signature
// and braces, so it can be shown to the LLM verbatim.
type FunctionDef struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
	Mutability string `json:"mutability"`
	Params     string `json:"params"`
	Body       string `json:"body"`
	LineStart  int    `json:"line_start"`
	LineEnd    int    `json:"line_end"`
}

// StateVar describes a contract-level state variable.
type StateVar struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Visibility string `json:"visibility"`
}

// CodeChunk is one function-sized unit sent to the RAG and LLM stages.
type CodeChunk struct {
	ContractName   string    `json:"contract_name"`
	FunctionName   string    `json:"function_name"`
	Code           string    `json:"code"`
	ContextSummary string    `json:"context_summary"`
	LineStart      int       `json:"line_start"`
	LineEnd        int       `json:"line_end"`
	Embedding      []float32 `json:"-"`
}
