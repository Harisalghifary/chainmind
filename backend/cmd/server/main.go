package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"chainmind/backend/internal/api"
	"chainmind/backend/internal/config"
	"chainmind/backend/internal/llm"
	"chainmind/backend/internal/orchestrator"
	"chainmind/backend/internal/rag"
	"chainmind/backend/internal/repository"
)

func main() {
	log := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "chainmind-api").
		Logger()

	if err := run(log); err != nil {
		log.Fatal().Err(err).Msg("server_exited")
	}
}

func run(log zerolog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := connectWithRetry(startupCtx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := repository.Migrate(startupCtx, pool, log); err != nil {
		return err
	}

	auditRepo := repository.NewAuditRepo(pool)
	exploitRepo := repository.NewExploitRepo(pool)

	if stale, err := auditRepo.FailStale(startupCtx); err != nil {
		log.Warn().Err(err).Msg("stale_audit_cleanup_failed")
	} else if stale > 0 {
		log.Warn().Int64("count", stale).Msg("stale_audits_marked_failed")
	}

	var retriever orchestrator.PatternRetriever
	embedder, err := rag.NewEmbedder(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("rag_disabled")
	} else {
		retriever = rag.NewRetriever(
			embedder,
			rag.NewVectorStore(exploitRepo),
			cfg.RAGTopK,
			cfg.RAGSimilarityThreshold,
		)
		log.Info().Str("embedding_model", embedder.Name()).Msg("rag_enabled")
	}

	llmClient, err := llm.NewClient(cfg, log)
	if err != nil {
		log.Warn().Err(err).Msg("llm_disabled")
	}

	worker := orchestrator.New(orchestrator.Options{
		Store:     auditRepo,
		Retriever: retriever,
		LLM:       llmClient,
		Logger:    log,
	})
	defer worker.Close()

	router := api.NewRouter(cfg, pool, auditRepo, exploitRepo, worker, log)
	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info().
			Str("addr", server.Addr).
			Str("env", cfg.Env).
			Msg("server_started")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-stop:
		log.Info().Str("signal", sig.String()).Msg("shutdown_started")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	worker.Close()
	workerDone := make(chan struct{})
	go func() {
		worker.Wait()
		close(workerDone)
	}()
	select {
	case <-workerDone:
	case <-time.After(15 * time.Second):
		log.Warn().Msg("audit_workers_shutdown_timeout")
	}

	log.Info().Msg("server_stopped")
	return nil
}

func connectWithRetry(ctx context.Context, databaseURL string, log zerolog.Logger) (*pgxpool.Pool, error) {
	const attempts = 10

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		pool, err := repository.NewPool(ctx, databaseURL)
		if err == nil {
			return pool, nil
		}
		lastErr = err

		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("max_attempts", attempts).
			Msg("database_connect_retry")

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return nil, lastErr
}
