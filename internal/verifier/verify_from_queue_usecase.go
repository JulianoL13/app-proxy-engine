package verifier

import (
	"context"
	"net/url"
	"time"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

type Message struct {
	ID      string
	Payload []byte
}

type Consumer interface {
	Subscribe(ctx context.Context, topic, group, consumer string) (<-chan Message, error)
	Ack(ctx context.Context, topic, group, msgID string) error
}

type Verifiable interface {
	Address() string
	URL() *url.URL
}

type ProxyChecker interface {
	Verify(ctx context.Context, p Verifiable) VerifyOutput
}

type VerifyOutput struct {
	Success   bool
	Latency   time.Duration
	Anonymity string
	Error     error
}

type VerifiedProxy interface {
	Verifiable
	MarkSuccess(latency time.Duration, anonymity string)
}

type ProxyDeserializer interface {
	Deserialize(payload []byte) (VerifiedProxy, error)
}

type Writer interface {
	Save(ctx context.Context, p VerifiedProxy) error
}

const (
	// Default values if not provided via config, though we encourage explicit config
	DefaultTopicVerify  = "proxies:verify"
	DefaultGroupWorkers = "verifiers"
)

type VerifyFromQueueUseCase struct {
	consumer     Consumer
	checker      ProxyChecker
	deserializer ProxyDeserializer
	writer       Writer
	logger       Logger
	id           string
	topic        string
	group        string
}

func NewVerifyFromQueueUseCase(
	consumer Consumer,
	checker ProxyChecker,
	deserializer ProxyDeserializer,
	writer Writer,
	logger Logger,
	consumerID string,
	topic string,
	group string,
) *VerifyFromQueueUseCase {
	if topic == "" {
		topic = DefaultTopicVerify
	}
	if group == "" {
		group = DefaultGroupWorkers
	}
	return &VerifyFromQueueUseCase{
		consumer:     consumer,
		checker:      checker,
		deserializer: deserializer,
		writer:       writer,
		logger:       logger,
		id:           consumerID,
		topic:        topic,
		group:        group,
	}
}

func (uc *VerifyFromQueueUseCase) Execute(ctx context.Context) error {
	uc.logger.Info("starting verification", "consumer", uc.id, "topic", uc.topic, "group", uc.group)

	messages, err := uc.consumer.Subscribe(ctx, uc.topic, uc.group, uc.id)
	if err != nil {
		return err
	}

	processed := 0
	alive := 0

	for msg := range messages {
		p, err := uc.deserializer.Deserialize(msg.Payload)
		if err != nil {
			uc.logger.Warn("failed to deserialize proxy", "error", err, "msgID", msg.ID)
			uc.consumer.Ack(ctx, uc.topic, uc.group, msg.ID)
			continue
		}

		result := uc.checker.Verify(ctx, p)
		processed++

		if result.Success {
			p.MarkSuccess(result.Latency, result.Anonymity)
			if err := uc.writer.Save(ctx, p); err != nil {
				uc.logger.Warn("failed to save proxy", "address", p.Address(), "error", err)
			} else {
				alive++
				uc.logger.Debug("proxy verified", "address", p.Address(), "latency", result.Latency)
			}
		}

		uc.consumer.Ack(ctx, uc.topic, uc.group, msg.ID)

		if processed%100 == 0 {
			uc.logger.Info("progress", "processed", processed, "alive", alive)
		}
	}

	uc.logger.Info("verification stopped", "processed", processed, "alive", alive)
	return nil
}
