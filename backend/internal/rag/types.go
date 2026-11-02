package rag

// ExploitPattern is a known vulnerability pattern retrieved from the vector
// store and injected into the analysis prompt.
type ExploitPattern struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	SWCID          string   `json:"swc_id"`
	Severity       string   `json:"severity"`
	Description    string   `json:"description"`
	VulnerableCode string   `json:"vulnerable_code"`
	FixedCode      string   `json:"fixed_code"`
	RealWorldCase  string   `json:"real_world_case"`
	References     []string `json:"references"`
}

// RetrievalResult is one pattern plus its cosine similarity to the chunk.
type RetrievalResult struct {
	Pattern         ExploitPattern `json:"pattern"`
	SimilarityScore float32        `json:"similarity_score"`
}
