package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/saisaravanan/healing-eval/internal/config"
	"github.com/saisaravanan/healing-eval/internal/evaluator"
	"github.com/saisaravanan/healing-eval/internal/llm"
	"github.com/saisaravanan/healing-eval/internal/queue"
	"github.com/saisaravanan/healing-eval/internal/storage"
	"github.com/saisaravanan/healing-eval/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := storage.NewPostgresDB(ctx, &cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	q, err := queue.NewRedisQueue(&cfg.Redis, &cfg.Worker)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer q.Close()

	llmClient, err := llm.NewClient(&cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	orchestrator := evaluator.NewOrchestrator(
		evaluator.NewHeuristicEvaluator(1000),
		evaluator.NewLLMJudgeEvaluator(llmClient),
		evaluator.NewToolCallEvaluator(llmClient),
		evaluator.NewCoherenceEvaluator(llmClient),
	)

	convRepo := storage.NewConversationRepo(db)
	evalRepo := storage.NewEvaluationRepo(db)

	w := worker.New(
		q,
		convRepo,
		evalRepo,
		orchestrator,
		cfg.Worker.Concurrency,
		cfg.Worker.BatchSize,
	)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down worker...")
		cancel()
	}()

	log.Println("Worker starting...")
	if err := w.Start(ctx); err != nil {
		log.Fatalf("Worker error: %v", err)
	}

	log.Println("Worker stopped")
}

