package rag

import (
	"context"
	"fmt"

	"chainmind/backend/internal/parser"
)

// Retriever embeds a code chunk and fetches the most similar exploit patterns.
type Retriever struct {
	embedder  Embedder
	searcher  Searcher
	topK      int
	threshold float64
}

func NewRetriever(embedder Embedder, searcher Searcher, topK int, threshold float64) *Retriever {
	if topK <= 0 {
		topK = 5
	}
	return &Retriever{
		embedder:  embedder,
		searcher:  searcher,
		topK:      topK,
		threshold: threshold,
	}
}

// Retrieve searches for patterns similar to the chunk code and its contract
// context. It returns an error when the embedding or search fails; callers may
// continue without RAG.
func (r *Retriever) Retrieve(ctx context.Context, chunk parser.CodeChunk) ([]RetrievalResult, error) {
	text := chunk.Code + "\n\n" + chunk.ContextSummary

	embedding, err := r.embedder.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embed chunk %s.%s: %w", chunk.ContractName, chunk.FunctionName, err)
	}

	results, err := r.searcher.Search(ctx, embedding, r.topK, r.threshold)
	if err != nil {
		return nil, fmt.Errorf("retrieve patterns for %s.%s: %w", chunk.ContractName, chunk.FunctionName, err)
	}

	return results, nil
}
