package rag

import (
	"context"
	"fmt"

	"chainmind/backend/internal/repository"
)

// Searcher abstracts vector similarity search for tests.
type Searcher interface {
	Search(ctx context.Context, embedding []float32, topK int, threshold float64) ([]RetrievalResult, error)
}

// VectorStore reads exploit patterns from pgvector.
type VectorStore struct {
	repo *repository.ExploitRepo
}

func NewVectorStore(repo *repository.ExploitRepo) *VectorStore {
	return &VectorStore{repo: repo}
}

func (s *VectorStore) Search(ctx context.Context, embedding []float32, topK int, threshold float64) ([]RetrievalResult, error) {
	matches, err := s.repo.Search(ctx, embedding, topK, threshold)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	results := make([]RetrievalResult, 0, len(matches))
	for _, match := range matches {
		results = append(results, RetrievalResult{
			Pattern:         fromRepository(match.Pattern),
			SimilarityScore: float32(match.Similarity),
		})
	}

	return results, nil
}

func fromRepository(pattern repository.ExploitPattern) ExploitPattern {
	result := ExploitPattern{
		ID:             pattern.ID,
		Name:           pattern.Name,
		Category:       pattern.Category,
		Severity:       pattern.Severity,
		Description:    pattern.Description,
		VulnerableCode: pattern.VulnerableCode,
		References:     pattern.References,
	}
	if pattern.SWCID != nil {
		result.SWCID = *pattern.SWCID
	}
	if pattern.FixedCode != nil {
		result.FixedCode = *pattern.FixedCode
	}
	if pattern.RealWorldCase != nil {
		result.RealWorldCase = *pattern.RealWorldCase
	}
	return result
}
