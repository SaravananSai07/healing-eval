package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type EvaluationRepo struct {
	db *PostgresDB
}

func NewEvaluationRepo(db *PostgresDB) *EvaluationRepo {
	return &EvaluationRepo{db: db}
}

func (r *EvaluationRepo) Create(ctx context.Context, eval *domain.Evaluation) error {
	if eval.ID == "" {
		eval.ID = uuid.New().String()
	}

	scoresJSON, err := json.Marshal(eval.Scores)
	if err != nil {
		return fmt.Errorf("marshal scores: %w", err)
	}

	issuesJSON, err := json.Marshal(eval.Issues)
	if err != nil {
		return fmt.Errorf("marshal issues: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, `
		INSERT INTO evaluations (
			id, conversation_id, evaluator_type, 
			status, model_name, prompt_tokens, completion_tokens, total_tokens, 
			estimated_cost_usd, error_message,
			scores, issues, confidence, raw_output, latency_ms, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, eval.ID, eval.ConversationID, eval.EvaluatorType,
		eval.Status, eval.ModelName, eval.PromptTokens, eval.CompletionTokens, eval.TotalTokens,
		eval.EstimatedCostUSD, eval.ErrorMessage,
		scoresJSON, issuesJSON, eval.Confidence, eval.RawOutput, eval.LatencyMs, time.Now())

	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

func (r *EvaluationRepo) CreateBatch(ctx context.Context, evals []*domain.Evaluation) error {
	batch := &pgx.Batch{}
	now := time.Now()

	for _, eval := range evals {
		if eval.ID == "" {
			eval.ID = uuid.New().String()
		}

		scoresJSON, _ := json.Marshal(eval.Scores)
		issuesJSON, _ := json.Marshal(eval.Issues)

		batch.Queue(`
			INSERT INTO evaluations (
				id, conversation_id, evaluator_type, 
				status, model_name, prompt_tokens, completion_tokens, total_tokens, 
				estimated_cost_usd, error_message,
				scores, issues, confidence, raw_output, latency_ms, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`, eval.ID, eval.ConversationID, eval.EvaluatorType,
			eval.Status, eval.ModelName, eval.PromptTokens, eval.CompletionTokens, eval.TotalTokens,
			eval.EstimatedCostUSD, eval.ErrorMessage,
			scoresJSON, issuesJSON, eval.Confidence, eval.RawOutput, eval.LatencyMs, now)
	}

	results := r.db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	for range evals {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch exec: %w", err)
		}
	}

	return nil
}

func (r *EvaluationRepo) GetByConversationID(ctx context.Context, conversationID string) ([]*domain.Evaluation, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, conversation_id, evaluator_type, 
			status, model_name, prompt_tokens, completion_tokens, total_tokens, 
			estimated_cost_usd, error_message,
			scores, issues, confidence, raw_output, latency_ms, created_at
		FROM evaluations
		WHERE conversation_id = $1
		ORDER BY created_at DESC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	return r.scanEvaluations(rows)
}

func (r *EvaluationRepo) Query(ctx context.Context, req *domain.EvaluationsQueryRequest) (*domain.EvaluationsQueryResponse, error) {
	req.SetDefaults()

	var conditions []string
	var args []interface{}
	argIdx := 1

	if len(req.ConversationIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("conversation_id = ANY($%d)", argIdx))
		args = append(args, req.ConversationIDs)
		argIdx++
	}

	if len(req.EvaluatorTypes) > 0 {
		types := make([]string, len(req.EvaluatorTypes))
		for i, t := range req.EvaluatorTypes {
			types[i] = string(t)
		}
		conditions = append(conditions, fmt.Sprintf("evaluator_type = ANY($%d)", argIdx))
		args = append(args, types)
		argIdx++
	}

	if req.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, req.DateFrom)
		argIdx++
	}

	if req.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, req.DateTo)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM evaluations %s", whereClause)
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count: %w", err)
	}

	orderColumn := "created_at"
	if req.SortBy == "overall_score" {
		orderColumn = "scores->>'overall'"
	}
	orderDir := "DESC"
	if req.SortOrder == "asc" {
		orderDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, conversation_id, evaluator_type, 
			status, model_name, prompt_tokens, completion_tokens, total_tokens, 
			estimated_cost_usd, error_message,
			scores, issues, confidence, raw_output, latency_ms, created_at
		FROM evaluations
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderColumn, orderDir, argIdx, argIdx+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	evals, err := r.scanEvaluations(rows)
	if err != nil {
		return nil, err
	}

	return &domain.EvaluationsQueryResponse{
		Evaluations: r.toDomainSlice(evals),
		Total:       total,
		Limit:       req.Limit,
		Offset:      req.Offset,
		HasMore:     req.Offset+len(evals) < total,
	}, nil
}

func (r *EvaluationRepo) scanEvaluations(rows pgx.Rows) ([]*domain.Evaluation, error) {
	var evals []*domain.Evaluation

	for rows.Next() {
		var eval domain.Evaluation
		var scoresJSON, issuesJSON []byte

		if err := rows.Scan(
			&eval.ID, &eval.ConversationID, &eval.EvaluatorType,
			&eval.Status, &eval.ModelName, &eval.PromptTokens, &eval.CompletionTokens, &eval.TotalTokens,
			&eval.EstimatedCostUSD, &eval.ErrorMessage,
			&scoresJSON, &issuesJSON, &eval.Confidence, &eval.RawOutput, &eval.LatencyMs, &eval.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		if err := json.Unmarshal(scoresJSON, &eval.Scores); err != nil {
			return nil, fmt.Errorf("unmarshal scores: %w", err)
		}

		if issuesJSON != nil {
			json.Unmarshal(issuesJSON, &eval.Issues)
		}

		evals = append(evals, &eval)
	}

	return evals, nil
}

func (r *EvaluationRepo) toDomainSlice(evals []*domain.Evaluation) []domain.Evaluation {
	result := make([]domain.Evaluation, len(evals))
	for i, e := range evals {
		result[i] = *e
	}
	return result
}

