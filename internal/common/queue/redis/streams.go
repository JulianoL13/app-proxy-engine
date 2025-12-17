package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultMaxLen     = 1000000
	errorBackoff      = 1 * time.Second
	readBlockDuration = 5 * time.Second
)

type Message struct {
	ID      string
	Payload []byte
}

type StreamsClient struct {
	client *redis.Client
	maxLen int64
}

func NewStreamsClient(client *redis.Client) *StreamsClient {
	return &StreamsClient{
		client: client,
		maxLen: defaultMaxLen,
	}
}

func (s *StreamsClient) WithMaxLen(maxLen int64) *StreamsClient {
	s.maxLen = maxLen
	return s
}

func (s *StreamsClient) Publish(ctx context.Context, topic string, payload []byte) error {
	_, err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		MaxLen: s.maxLen,
		Approx: true,
		Values: map[string]interface{}{
			"payload": payload,
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("xadd %s: %w", topic, err)
	}
	return nil
}

func (s *StreamsClient) Subscribe(ctx context.Context, topic, group, consumer string) (<-chan Message, error) {
	err := s.client.XGroupCreateMkStream(ctx, topic, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, fmt.Errorf("create group %s: %w", group, err)
	}

	messages := make(chan Message)

	go func() {
		defer close(messages)

		s.recoverPending(ctx, topic, group, consumer, messages)

		s.consumeLive(ctx, topic, group, consumer, messages)
	}()

	return messages, nil
}

func (s *StreamsClient) recoverPending(ctx context.Context, topic, group, consumer string, messages chan<- Message) {
	for {
		if ctx.Err() != nil {
			return
		}

		result, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{topic, "0"},
			Count:    100,
		}).Result()

		if err != nil {
			time.Sleep(errorBackoff)
			continue
		}

		if len(result) == 0 || len(result[0].Messages) == 0 {
			return
		}

		for _, stream := range result {
			for _, msg := range stream.Messages {
				payload, ok := msg.Values["payload"].(string)
				if !ok {
					continue
				}

				select {
				case <-ctx.Done():
					return
				case messages <- Message{ID: msg.ID, Payload: []byte(payload)}:
				}
			}
		}
	}
}

func (s *StreamsClient) consumeLive(ctx context.Context, topic, group, consumer string, messages chan<- Message) {
	for {
		if ctx.Err() != nil {
			return
		}

		result, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{topic, ">"},
			Count:    10,
			Block:    readBlockDuration,
		}).Result()

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err != redis.Nil {
				time.Sleep(errorBackoff)
			}
			continue
		}

		for _, stream := range result {
			for _, msg := range stream.Messages {
				payload, ok := msg.Values["payload"].(string)
				if !ok {
					continue
				}

				select {
				case <-ctx.Done():
					return
				case messages <- Message{ID: msg.ID, Payload: []byte(payload)}:
				}
			}
		}
	}
}

func (s *StreamsClient) Ack(ctx context.Context, topic, group, msgID string) error {
	_, err := s.client.XAck(ctx, topic, group, msgID).Result()
	if err != nil {
		return fmt.Errorf("xack %s: %w", msgID, err)
	}
	return nil
}

func (s *StreamsClient) Close() error {
	return nil
}
