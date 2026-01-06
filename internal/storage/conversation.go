package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type ConversationRepo struct {
	db *PostgresDB
}

func NewConversationRepo(db *PostgresDB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

func (r *ConversationRepo) Create(ctx context.Context, conv *domain.Conversation) error {
	turnsJSON, err := json.Marshal(conv.Turns)
	if err != nil {
		return fmt.Errorf("marshal turns: %w", err)
	}

	var feedbackJSON []byte
	if conv.Feedback != nil {
		feedbackJSON, err = json.Marshal(conv.Feedback)
		if err != nil {
			return fmt.Errorf("marshal feedback: %w", err)
		}
	}

	metadataJSON := conv.Metadata
	if metadataJSON == nil {
		metadataJSON = json.RawMessage("{}")
	}

	_, err = r.db.Pool.Exec(ctx, `
		INSERT INTO conversations (id, agent_version, turns, feedback, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			agent_version = EXCLUDED.agent_version,
			turns = EXCLUDED.turns,
			feedback = COALESCE(EXCLUDED.feedback, conversations.feedback),
			metadata = EXCLUDED.metadata
	`, conv.ID, conv.AgentVersion, turnsJSON, feedbackJSON, metadataJSON, time.Now())

	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

func (r *ConversationRepo) CreateBatch(ctx context.Context, convs []*domain.Conversation) error {
	batch := &pgx.Batch{}
	now := time.Now()

	for _, conv := range convs {
		turnsJSON, err := json.Marshal(conv.Turns)
		if err != nil {
			return fmt.Errorf("marshal turns for %s: %w", conv.ID, err)
		}

		var feedbackJSON []byte
		if conv.Feedback != nil {
			feedbackJSON, _ = json.Marshal(conv.Feedback)
		}

		metadataJSON := conv.Metadata
		if metadataJSON == nil {
			metadataJSON = json.RawMessage("{}")
		}

		batch.Queue(`
			INSERT INTO conversations (id, agent_version, turns, feedback, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				agent_version = EXCLUDED.agent_version,
				turns = EXCLUDED.turns,
				feedback = COALESCE(EXCLUDED.feedback, conversations.feedback),
				metadata = EXCLUDED.metadata
		`, conv.ID, conv.AgentVersion, turnsJSON, feedbackJSON, metadataJSON, now)
	}

	results := r.db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	for range convs {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch exec: %w", err)
		}
	}

	return nil
}

func (r *ConversationRepo) GetByID(ctx context.Context, id string) (*domain.Conversation, error) {
	var conv domain.Conversation
	var turnsJSON, feedbackJSON, metadataJSON []byte

	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, agent_version, turns, feedback, metadata, created_at, processed_at
		FROM conversations
		WHERE id = $1
	`, id).Scan(&conv.ID, &conv.AgentVersion, &turnsJSON, &feedbackJSON, &metadataJSON, &conv.CreatedAt, &conv.ProcessedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query: %w", err)
	}

	if err := json.Unmarshal(turnsJSON, &conv.Turns); err != nil {
		return nil, fmt.Errorf("unmarshal turns: %w", err)
	}

	if feedbackJSON != nil {
		conv.Feedback = &domain.Feedback{}
		if err := json.Unmarshal(feedbackJSON, conv.Feedback); err != nil {
			return nil, fmt.Errorf("unmarshal feedback: %w", err)
		}
	}

	conv.Metadata = metadataJSON
	return &conv, nil
}

func (r *ConversationRepo) UpdateFeedback(ctx context.Context, id string, feedback *domain.Feedback) error {
	feedbackJSON, err := json.Marshal(feedback)
	if err != nil {
		return fmt.Errorf("marshal feedback: %w", err)
	}

	result, err := r.db.Pool.Exec(ctx, `
		UPDATE conversations SET feedback = $2 WHERE id = $1
	`, id, feedbackJSON)

	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}

	return nil
}

func (r *ConversationRepo) MarkProcessed(ctx context.Context, id string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE conversations SET processed_at = NOW() WHERE id = $1
	`, id)
	return err
}

func (r *ConversationRepo) GetUnprocessed(ctx context.Context, limit int) ([]*domain.Conversation, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, agent_version, turns, feedback, metadata, created_at
		FROM conversations
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var convs []*domain.Conversation
	for rows.Next() {
		var conv domain.Conversation
		var turnsJSON, feedbackJSON, metadataJSON []byte

		if err := rows.Scan(&conv.ID, &conv.AgentVersion, &turnsJSON, &feedbackJSON, &metadataJSON, &conv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		if err := json.Unmarshal(turnsJSON, &conv.Turns); err != nil {
			return nil, fmt.Errorf("unmarshal turns: %w", err)
		}

		if feedbackJSON != nil {
			conv.Feedback = &domain.Feedback{}
			json.Unmarshal(feedbackJSON, conv.Feedback)
		}

		conv.Metadata = metadataJSON
		convs = append(convs, &conv)
	}

	return convs, nil
}

