package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/saisaravanan/healing-eval/internal/config"
	"github.com/saisaravanan/healing-eval/internal/domain"
)

type RedisQueue struct {
	client        *redis.Client
	streamName    string
	consumerGroup string
	consumerName  string
}

func NewRedisQueue(cfg *config.RedisConfig, workerCfg *config.WorkerConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	q := &RedisQueue{
		client:        client,
		streamName:    workerCfg.StreamName,
		consumerGroup: workerCfg.ConsumerGroup,
		consumerName:  workerCfg.ConsumerName,
	}

	if err := q.ensureConsumerGroup(ctx); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *RedisQueue) ensureConsumerGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.streamName, q.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

func (q *RedisQueue) Publish(ctx context.Context, conv *domain.Conversation) error {
	data, err := json.Marshal(conv)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamName,
		Values: map[string]interface{}{
			"conversation_id": conv.ID,
			"data":            string(data),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("xadd: %w", err)
	}

	return nil
}

func (q *RedisQueue) PublishBatch(ctx context.Context, convs []*domain.Conversation) error {
	pipe := q.client.Pipeline()

	for _, conv := range convs {
		data, err := json.Marshal(conv)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", conv.ID, err)
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: q.streamName,
			Values: map[string]interface{}{
				"conversation_id": conv.ID,
				"data":            string(data),
			},
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline exec: %w", err)
	}

	return nil
}

type Message struct {
	ID           string
	Conversation *domain.Conversation
}

func (q *RedisQueue) Consume(ctx context.Context, count int64, blockDuration time.Duration) ([]Message, error) {
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.consumerGroup,
		Consumer: q.consumerName,
		Streams:  []string{q.streamName, ">"},
		Count:    count,
		Block:    blockDuration,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("xreadgroup: %w", err)
	}

	var messages []Message
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			data, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}

			var conv domain.Conversation
			if err := json.Unmarshal([]byte(data), &conv); err != nil {
				continue
			}

			messages = append(messages, Message{
				ID:           msg.ID,
				Conversation: &conv,
			})
		}
	}

	return messages, nil
}

func (q *RedisQueue) Ack(ctx context.Context, messageIDs ...string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	_, err := q.client.XAck(ctx, q.streamName, q.consumerGroup, messageIDs...).Result()
	if err != nil {
		return fmt.Errorf("xack: %w", err)
	}

	return nil
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}

func (q *RedisQueue) Len(ctx context.Context) (int64, error) {
	return q.client.XLen(ctx, q.streamName).Result()
}

func (q *RedisQueue) Client() *redis.Client {
	return q.client
}

