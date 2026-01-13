package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type SuggestionRepo struct {
	db *PostgresDB
}

func NewSuggestionRepo(db *PostgresDB) *SuggestionRepo {
	return &SuggestionRepo{db: db}
}

func (r *SuggestionRepo) Create(ctx context.Context, s *domain.Suggestion) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO improvement_suggestions (id, pattern_id, suggestion_type, target, suggestion, rationale, confidence, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, s.ID, s.PatternID, s.Type, s.Target, s.Suggestion, s.Rationale, s.Confidence, s.Status, s.CreatedAt, s.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

func (r *SuggestionRepo) GetByID(ctx context.Context, id string) (*domain.Suggestion, error) {
	var s domain.Suggestion
	var impactJSON []byte

	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, pattern_id, suggestion_type, target, suggestion, rationale, confidence, status, impact_measured, created_at, updated_at
		FROM improvement_suggestions
		WHERE id = $1
	`, id).Scan(&s.ID, &s.PatternID, &s.Type, &s.Target, &s.Suggestion, &s.Rationale, &s.Confidence, &s.Status, &impactJSON, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query: %w", err)
	}

	if impactJSON != nil {
		s.ImpactMeasured = &domain.ImpactMetrics{}
		json.Unmarshal(impactJSON, s.ImpactMeasured)
	}

	return &s, nil
}

func (r *SuggestionRepo) List(ctx context.Context, status *domain.SuggestionStatus, limit, offset int) ([]*domain.Suggestion, error) {
	query := `
		SELECT id, pattern_id, suggestion_type, target, suggestion, rationale, confidence, status, impact_measured, created_at, updated_at
		FROM improvement_suggestions
	`
	var args []interface{}

	if status != nil {
		query += " WHERE status = $1"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var suggestions []*domain.Suggestion
	for rows.Next() {
		var s domain.Suggestion
		var impactJSON []byte

		if err := rows.Scan(&s.ID, &s.PatternID, &s.Type, &s.Target, &s.Suggestion, &s.Rationale, &s.Confidence, &s.Status, &impactJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		if impactJSON != nil {
			s.ImpactMeasured = &domain.ImpactMetrics{}
			json.Unmarshal(impactJSON, s.ImpactMeasured)
		}

		suggestions = append(suggestions, &s)
	}

	return suggestions, nil
}

func (r *SuggestionRepo) UpdateStatus(ctx context.Context, id string, status domain.SuggestionStatus) error {
	result, err := r.db.Pool.Exec(ctx, `
		UPDATE improvement_suggestions SET status = $2, updated_at = NOW() WHERE id = $1
	`, id, status)

	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("suggestion not found")
	}

	return nil
}

func (r *SuggestionRepo) GetByPatternID(ctx context.Context, patternID string) (*domain.Suggestion, error) {
	var s domain.Suggestion
	var impactJSON []byte

	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, pattern_id, suggestion_type, target, suggestion, rationale, confidence, status, impact_measured, created_at, updated_at
		FROM improvement_suggestions
		WHERE pattern_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, patternID).Scan(&s.ID, &s.PatternID, &s.Type, &s.Target, &s.Suggestion, &s.Rationale, &s.Confidence, &s.Status, &impactJSON, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query: %w", err)
	}

	if impactJSON != nil {
		s.ImpactMeasured = &domain.ImpactMetrics{}
		json.Unmarshal(impactJSON, s.ImpactMeasured)
	}

	return &s, nil
}

func (r *SuggestionRepo) UpdateImpact(ctx context.Context, id string, impact *domain.ImpactMetrics) error {
	impactJSON, err := json.Marshal(impact)
	if err != nil {
		return fmt.Errorf("marshal impact: %w", err)
	}

	result, err := r.db.Pool.Exec(ctx, `
		UPDATE improvement_suggestions SET impact_measured = $2, updated_at = NOW() WHERE id = $1
	`, id, impactJSON)

	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("suggestion not found")
	}

	return nil
}

