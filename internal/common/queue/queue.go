package queue

import "context"

type Message struct {
	ID      string
	Payload []byte
}

type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Close() error
}

type Consumer interface {
	Subscribe(ctx context.Context, topic, group, consumer string) (<-chan Message, error)
	Ack(ctx context.Context, topic, group, msgID string) error
	Close() error
}
