package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/saisaravanan/healing-eval/internal/api/handler"
	"github.com/saisaravanan/healing-eval/internal/config"
	"github.com/saisaravanan/healing-eval/internal/llm"
	"github.com/saisaravanan/healing-eval/internal/queue"
	"github.com/saisaravanan/healing-eval/internal/storage"
)

type Router struct {
	engine *gin.Engine
}

func NewRouter(db *storage.PostgresDB, q *queue.RedisQueue) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	convRepo := storage.NewConversationRepo(db)
	evalRepo := storage.NewEvaluationRepo(db)
	suggRepo := storage.NewSuggestionRepo(db)
	reviewQueueRepo := storage.NewReviewQueueRepo(db)

	// Create LLM client for suggestion generation
	cfg, err := config.Load()
	var llmClient *llm.Client
	if err != nil {
		log.Printf("Warning: Failed to load config for LLM client: %v", err)
	} else {
		llmClient, err = llm.NewClient(&cfg.LLM)
		if err != nil {
			log.Printf("Warning: Failed to create LLM client: %v", err)
		}
	}

	convHandler := handler.NewConversationHandler(convRepo, q)
	evalHandler := handler.NewEvaluationHandler(evalRepo)
	suggHandler := handler.NewSuggestionHandler(suggRepo, evalRepo, llmClient)
	reviewHandler := handler.NewReviewHandler(reviewQueueRepo, evalRepo, convRepo)
	metricsHandler := handler.NewMetricsHandler()
	webHandler := handler.NewWebHandler(convRepo, evalRepo, suggRepo)

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	engine.GET("/", webHandler.Dashboard)
	engine.GET("/conversations", webHandler.Conversations)
	engine.GET("/suggestions", webHandler.Suggestions)
	engine.GET("/reviews", webHandler.Reviews)
	engine.GET("/metrics", webHandler.Metrics)

	engine.GET("/partials/recent-evaluations", webHandler.RecentEvaluations)
	engine.GET("/partials/active-issues", webHandler.ActiveIssues)
	engine.GET("/partials/evaluator-performance", webHandler.EvaluatorPerformance)
	engine.GET("/partials/suggestions-table", webHandler.SuggestionsTable)
	engine.GET("/partials/evaluator-accuracy", webHandler.EvaluatorAccuracy)
	engine.GET("/partials/calibration", webHandler.Calibration)
	engine.GET("/partials/blind-spots", webHandler.BlindSpots)
	engine.GET("/partials/conversations-list", webHandler.ConversationsList)
	engine.GET("/partials/conversation/:id", webHandler.ConversationDetail)

	engine.GET("/api/stats/conversations", webHandler.StatConversations)
	engine.GET("/api/stats/evaluations", webHandler.StatEvaluations)
	engine.GET("/api/stats/suggestions", webHandler.StatSuggestions)
	engine.GET("/api/stats/avg-score", webHandler.StatAvgScore)

	v1 := engine.Group("/api/v1")
	{
		conversations := v1.Group("/conversations")
		{
			conversations.POST("", convHandler.Ingest)
			conversations.GET("/:id", convHandler.GetByID)
			conversations.POST("/:id/feedback", convHandler.UpdateFeedback)
		}

		evaluations := v1.Group("/evaluations")
		{
			evaluations.GET("/:conversation_id", evalHandler.GetByConversationID)
			evaluations.POST("/query", evalHandler.Query)
		}

		suggestions := v1.Group("/suggestions")
		{
			suggestions.GET("", suggHandler.List)
			suggestions.POST("/generate", suggHandler.Generate)
			suggestions.GET("/:id", suggHandler.GetByID)
			suggestions.POST("/:id/approve", suggHandler.Approve)
			suggestions.POST("/:id/reject", suggHandler.Reject)
			suggestions.GET("/:id/impact", suggHandler.GetImpact)
		}

		reviews := v1.Group("/reviews")
		{
			reviews.GET("/pending", reviewHandler.GetPending)
			reviews.GET("/:id", reviewHandler.GetByID)
			reviews.POST("/:id/complete", reviewHandler.CompleteReview)
			reviews.POST("/:id/assign", reviewHandler.AssignReview)
		}

		metrics := v1.Group("/metrics")
		{
			metrics.GET("/evaluators", metricsHandler.GetEvaluators)
			metrics.GET("/calibration", metricsHandler.GetCalibration)
			metrics.GET("/blind-spots", metricsHandler.GetBlindSpots)
		}
	}

	return &Router{engine: engine}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

