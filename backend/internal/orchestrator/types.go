package orchestrator

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"chainmind/backend/internal/llm"
	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/rag"
	"chainmind/backend/internal/repository"
)

// AuditStore is the persistence surface used by the worker.
type AuditStore interface {
	Create(ctx context.Context, params repository.CreateAuditParams) (*repository.AuditJob, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateProgress(ctx context.Context, id, status string, totalChunks, analyzedChunks int) error
	ReplaceFindings(ctx context.Context, auditID string, findings []repository.Finding) error
	Complete(ctx context.Context, id, llmModel string, ragPatternsUsed int, warnings []string) error
	Fail(ctx context.Context, id, message string) error
}

// PatternRetriever is the RAG surface used by the worker.
type PatternRetriever interface {
	Retrieve(ctx context.Context, chunk parser.CodeChunk) ([]rag.RetrievalResult, error)
}

// Auditor is the provider surface used by the worker.
type Auditor interface {
	Analyze(ctx context.Context, request llm.AnalysisRequest) (*llm.AnalysisResponse, error)
	Name() string
}

// Options configures an Orchestrator. Zero values fall back to safe defaults.
type Options struct {
	Store               AuditStore
	Retriever           PatternRetriever
	LLM                 Auditor
	Logger              zerolog.Logger
	MaxConcurrentAudits int
	ChunkTimeout        time.Duration
}
