package redis

import (
	"context"
	"fmt"

	"github.com/JulianoL13/app-proxy-engine/internal/common/queue"
	"github.com/redis/go-redis/v9"
)

type StreamsClient struct {
	client *redis.Client
}

func NewStreamsClient(client *redis.Client) *StreamsClient {
	return &StreamsClient{client: client}
}

func (s *StreamsClient) Publish(ctx context.Context, topic string, payload []byte) error {
	_, err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"payload": payload,
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("xadd %s: %w", topic, err)
	}
	return nil
}

func (s *StreamsClient) Subscribe(ctx context.Context, topic, group, consumer string) (<-chan queue.Message, error) {
	err := s.client.XGroupCreateMkStream(ctx, topic, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, fmt.Errorf("create group %s: %w", group, err)
	}

	messages := make(chan queue.Message)

	go func() {
		defer close(messages)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			result, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{topic, ">"},
				Count:    1,
				Block:    0,
			}).Result()

			if err != nil {
				if ctx.Err() != nil {
					return
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
					case messages <- queue.Message{
						ID:      msg.ID,
						Payload: []byte(payload),
					}:
					}
				}
			}
		}
	}()

	return messages, nil
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

var (
	_ queue.Publisher = (*StreamsClient)(nil)
	_ queue.Consumer  = (*StreamsClient)(nil)
)
