package rag

import (
	"context"
	"errors"
	"testing"

	"chainmind/backend/internal/parser"
)

type fakeEmbedder struct {
	text      string
	embedding []float32
	err       error
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	f.text = text
	return f.embedding, f.err
}

func (f *fakeEmbedder) Name() string { return "fake" }

type fakeSearcher struct {
	embedding []float32
	topK      int
	threshold float64
	results   []RetrievalResult
	err       error
}

func (f *fakeSearcher) Search(_ context.Context, embedding []float32, topK int, threshold float64) ([]RetrievalResult, error) {
	f.embedding = embedding
	f.topK = topK
	f.threshold = threshold
	return f.results, f.err
}

func TestRetrieverCombinesCodeAndContext(t *testing.T) {
	embedder := &fakeEmbedder{embedding: []float32{1, 2, 3}}
	searcher := &fakeSearcher{results: []RetrievalResult{{
		Pattern:         ExploitPattern{Name: "Reentrancy"},
		SimilarityScore: 0.91,
	}}}
	retriever := NewRetriever(embedder, searcher, 5, 0.75)

	chunk := parser.CodeChunk{
		ContractName:   "Vault",
		FunctionName:   "withdraw",
		Code:           "function withdraw() external {}",
		ContextSummary: "Contract state variables:\n- mapping(address => uint256) balances (public)",
	}

	results, err := retriever.Retrieve(context.Background(), chunk)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(results) != 1 || results[0].Pattern.Name != "Reentrancy" {
		t.Errorf("results = %+v", results)
	}
	if embedder.text != chunk.Code+"\n\n"+chunk.ContextSummary {
		t.Errorf("embedded text = %q", embedder.text)
	}
	if searcher.topK != 5 || searcher.threshold != 0.75 {
		t.Errorf("search args = topK %d, threshold %v", searcher.topK, searcher.threshold)
	}
	if len(searcher.embedding) != 3 {
		t.Errorf("embedding = %v", searcher.embedding)
	}
}

func TestRetrieverPropagatesEmbeddingError(t *testing.T) {
	embedder := &fakeEmbedder{err: errors.New("api down")}
	retriever := NewRetriever(embedder, &fakeSearcher{}, 5, 0.75)

	if _, err := retriever.Retrieve(context.Background(), parser.CodeChunk{}); err == nil {
		t.Fatal("expected an embedding error")
	}
}

func TestRetrieverPropagatesSearchError(t *testing.T) {
	embedder := &fakeEmbedder{embedding: []float32{1}}
	searcher := &fakeSearcher{err: errors.New("db down")}
	retriever := NewRetriever(embedder, searcher, 5, 0.75)

	if _, err := retriever.Retrieve(context.Background(), parser.CodeChunk{}); err == nil {
		t.Fatal("expected a search error")
	}
}
