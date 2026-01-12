package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type ReviewQueueRepo struct {
	db *PostgresDB
}

func NewReviewQueueRepo(db *PostgresDB) *ReviewQueueRepo {
	return &ReviewQueueRepo{db: db}
}

func (r *ReviewQueueRepo) AddToQueue(ctx context.Context, item *domain.ReviewQueueItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO human_review_queue (
			id, conversation_id, evaluation_id, reason, priority, 
			status, assigned_to, routing_confidence, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, item.ID, item.ConversationID, item.EvaluationID, item.Reason, item.Priority,
		item.Status, item.AssignedTo, item.RoutingConfidence, time.Now())

	if err != nil {
		return fmt.Errorf("insert review queue item: %w", err)
	}

	return nil
}

func (r *ReviewQueueRepo) GetPending(ctx context.Context, limit int) ([]*domain.ReviewQueueItem, error) {
	return r.GetPendingPaginated(ctx, limit, 0)
}

func (r *ReviewQueueRepo) GetPendingPaginated(ctx context.Context, limit, offset int) ([]*domain.ReviewQueueItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, conversation_id, evaluation_id, reason, priority, 
			status, assigned_to, routing_confidence, created_at, reviewed_at, reviewer_notes
		FROM human_review_queue
		WHERE status = 'pending'
		ORDER BY priority ASC, created_at ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query pending reviews: %w", err)
	}
	defer rows.Close()

	return r.scanReviewQueueItems(rows)
}

func (r *ReviewQueueRepo) CountPending(ctx context.Context) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM human_review_queue WHERE status = 'pending'
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending reviews: %w", err)
	}
	return count, nil
}

func (r *ReviewQueueRepo) GetByID(ctx context.Context, id string) (*domain.ReviewQueueItem, error) {
	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, conversation_id, evaluation_id, reason, priority, 
			status, assigned_to, routing_confidence, created_at, reviewed_at, reviewer_notes
		FROM human_review_queue
		WHERE id = $1
	`, id)

	var item domain.ReviewQueueItem
	err := row.Scan(
		&item.ID, &item.ConversationID, &item.EvaluationID, &item.Reason, &item.Priority,
		&item.Status, &item.AssignedTo, &item.RoutingConfidence, &item.CreatedAt,
		&item.ReviewedAt, &item.ReviewerNotes,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("review queue item not found")
		}
		return nil, fmt.Errorf("scan review queue item: %w", err)
	}

	return &item, nil
}

func (r *ReviewQueueRepo) CompleteReview(ctx context.Context, id string, reviewerNotes string) error {
	now := time.Now()
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE human_review_queue
		SET status = 'completed', reviewed_at = $1, reviewer_notes = $2
		WHERE id = $3
	`, now, reviewerNotes, id)

	if err != nil {
		return fmt.Errorf("complete review: %w", err)
	}

	return nil
}

func (r *ReviewQueueRepo) AssignReview(ctx context.Context, id string, assignedTo string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE human_review_queue
		SET status = 'in_progress', assigned_to = $1
		WHERE id = $2 AND status = 'pending'
	`, assignedTo, id)

	if err != nil {
		return fmt.Errorf("assign review: %w", err)
	}

	return nil
}

func (r *ReviewQueueRepo) scanReviewQueueItems(rows pgx.Rows) ([]*domain.ReviewQueueItem, error) {
	var items []*domain.ReviewQueueItem

	for rows.Next() {
		var item domain.ReviewQueueItem
		if err := rows.Scan(
			&item.ID, &item.ConversationID, &item.EvaluationID, &item.Reason, &item.Priority,
			&item.Status, &item.AssignedTo, &item.RoutingConfidence, &item.CreatedAt,
			&item.ReviewedAt, &item.ReviewerNotes,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		items = append(items, &item)
	}

	return items, nil
}
