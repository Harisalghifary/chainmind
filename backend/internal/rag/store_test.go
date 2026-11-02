package rag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"chainmind/backend/internal/parser"
	"chainmind/backend/internal/repository"
)

func integrationStore(t *testing.T) (*VectorStore, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping database integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := repository.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	if err := repository.Migrate(ctx, pool, zerolog.Nop()); err != nil {
		pool.Close()
		t.Fatalf("Migrate() error = %v", err)
	}
	t.Cleanup(pool.Close)

	return NewVectorStore(repository.NewExploitRepo(pool)), pool
}

func TestVectorStoreRoundTrip(t *testing.T) {
	store, pool := integrationStore(t)
	ctx := context.Background()
	embedder := NewMockEmbedder()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	marker := "zqx" + suffix
	reentrantName := "test-store-reentrancy-" + suffix
	otherName := "test-store-access-" + suffix
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM exploit_patterns WHERE name = $1 OR name = $2`, reentrantName, otherName)
	})

	reentrantText := "reentrancy external call before state update withdraw balances " + marker
	reentrantEmbedding, err := embedder.Embed(ctx, reentrantText)
	if err != nil {
		t.Fatalf("Embed(reentrant) error = %v", err)
	}
	otherEmbedding, err := embedder.Embed(ctx, "access control onlyOwner mint privileged role "+marker+"other")
	if err != nil {
		t.Fatalf("Embed(other) error = %v", err)
	}

	repo := repository.NewExploitRepo(pool)
	swc := "SWC-107"
	realCase := "The DAO Hack, 2016"
	fixed := "balances[msg.sender] = 0;"
	reentrant, err := repo.Insert(ctx, &repository.ExploitPattern{
		Name:           reentrantName,
		Category:       "reentrancy",
		SWCID:          &swc,
		Severity:       "critical",
		Description:    reentrantText,
		VulnerableCode: "msg.sender.call{value: amount}(\"\");",
		FixedCode:      &fixed,
		RealWorldCase:  &realCase,
		References:     []string{"https://swcregistry.io/docs/SWC-107"},
		Embedding:      reentrantEmbedding,
	})
	if err != nil {
		t.Fatalf("insert reentrant pattern: %v", err)
	}

	if _, err := repo.Insert(ctx, &repository.ExploitPattern{
		Name:           otherName,
		Category:       "access_control",
		Severity:       "critical",
		Description:    "access control onlyOwner mint privileged role",
		VulnerableCode: "function mint() external {}",
		Embedding:      otherEmbedding,
	}); err != nil {
		t.Fatalf("insert other pattern: %v", err)
	}

	queryEmbedding, err := embedder.Embed(ctx, reentrantText)
	if err != nil {
		t.Fatalf("Embed(query) error = %v", err)
	}

	results, err := store.Search(ctx, queryEmbedding, 5, 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search() returned no results")
	}
	if results[0].Pattern.ID != reentrant.ID {
		t.Errorf("top result = %q, want %q", results[0].Pattern.Name, reentrantName)
	}
	if results[0].SimilarityScore < 0.98 {
		t.Errorf("similarity = %v, want ~1 for an exact match", results[0].SimilarityScore)
	}

	top := results[0].Pattern
	if top.SWCID != "SWC-107" || top.RealWorldCase != realCase || top.FixedCode != fixed {
		t.Errorf("mapped pattern = %+v", top)
	}
	if len(top.References) != 1 {
		t.Errorf("references = %v", top.References)
	}

	filtered, err := store.Search(ctx, queryEmbedding, 5, 0.98)
	if err != nil {
		t.Fatalf("Search(threshold) error = %v", err)
	}
	for _, result := range filtered {
		if result.Pattern.ID != reentrant.ID {
			t.Errorf("threshold kept unrelated pattern %q", result.Pattern.Name)
		}
	}
}
func TestSeededReentrancyCorpusRanking(t *testing.T) {
	store, pool := integrationStore(t)
	ctx := context.Background()

	var embedded int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM exploit_patterns WHERE embedding IS NOT NULL`).Scan(&embedded); err != nil {
		t.Fatalf("count seeded patterns: %v", err)
	}
	if embedded < 20 {
		t.Skipf("only %d embedded patterns; run `make seed` first", embedded)
	}

	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vulnerable", "reentrancy.sol"))
	if err != nil {
		t.Fatalf("read test contract: %v", err)
	}
	contract, err := parser.Parse(string(content))
	if err != nil {
		t.Fatalf("parse test contract: %v", err)
	}

	var withdraw parser.CodeChunk
	for _, chunk := range parser.Chunk(contract) {
		if chunk.FunctionName == "withdraw" {
			withdraw = chunk
		}
	}
	if withdraw.FunctionName == "" {
		t.Fatal("withdraw chunk not found")
	}

	retriever := NewRetriever(NewMockEmbedder(), store, 5, 0.1)
	results, err := retriever.Retrieve(ctx, withdraw)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Retrieve() returned no patterns")
	}
	if results[0].Pattern.Category != "reentrancy" {
		t.Errorf("top category = %q, want reentrancy (top: %q)", results[0].Pattern.Category, results[0].Pattern.Name)
	}
	for _, result := range results {
		t.Logf("%.3f %s [%s]", result.SimilarityScore, result.Pattern.Name, result.Pattern.Category)
	}
}
