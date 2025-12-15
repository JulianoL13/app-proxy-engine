package verifier_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
	"github.com/stretchr/testify/assert"
)

type mockConsumer struct {
	messages chan verifier.Message
	acked    []string
	err      error
}

func (m *mockConsumer) Subscribe(ctx context.Context, topic, group, consumer string) (<-chan verifier.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.messages, nil
}

func (m *mockConsumer) Ack(ctx context.Context, topic, group, msgID string) error {
	m.acked = append(m.acked, msgID)
	return nil
}

type mockChecker struct {
	output verifier.VerifyOutput
}

func (m *mockChecker) Verify(ctx context.Context, p verifier.Verifiable) verifier.VerifyOutput {
	return m.output
}

type mockDeserializer struct {
	proxy verifier.VerifiedProxy
	err   error
}

func (m *mockDeserializer) Deserialize(payload []byte) (verifier.VerifiedProxy, error) {
	return m.proxy, m.err
}

type mockWriter struct {
	saved []verifier.VerifiedProxy
	err   error
}

func (m *mockWriter) Save(ctx context.Context, p verifier.VerifiedProxy) error {
	if m.err != nil {
		return m.err
	}
	m.saved = append(m.saved, p)
	return nil
}

type stubProxy struct {
	address   string
	latency   time.Duration
	anonymity string
}

func (s *stubProxy) Address() string { return s.address }
func (s *stubProxy) URL() *url.URL   { return nil }
func (s *stubProxy) MarkSuccess(latency time.Duration, anonymity string) {
	s.latency = latency
	s.anonymity = anonymity
}

type verifierTestLogger struct{}

func (l verifierTestLogger) Info(msg string, args ...any)  {}
func (l verifierTestLogger) Warn(msg string, args ...any)  {}
func (l verifierTestLogger) Debug(msg string, args ...any) {}

type mockWorkerPool struct{}

func (m *mockWorkerPool) Submit(job func(ctx context.Context)) {
	job(context.Background())
}

func TestVerifyFromQueueUseCase_Execute(t *testing.T) {
	logger := verifierTestLogger{}
	pool := &mockWorkerPool{}

	t.Run("processes and saves successful proxy", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`{}`)}
		close(messages)

		proxy := &stubProxy{address: "1.1.1.1:8080"}

		consumer := &mockConsumer{messages: messages}
		checker := &mockChecker{output: verifier.VerifyOutput{
			Success:   true,
			Latency:   100 * time.Millisecond,
			Anonymity: "elite",
		}}
		deserializer := &mockDeserializer{proxy: proxy}
		writer := &mockWriter{}

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
		assert.Len(t, writer.saved, 1)
		assert.Contains(t, consumer.acked, "msg-1")
		assert.Equal(t, "elite", proxy.anonymity)
	})

	t.Run("acks but does not save failed proxy", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`{}`)}
		close(messages)

		proxy := &stubProxy{address: "1.1.1.1:8080"}

		consumer := &mockConsumer{messages: messages}
		checker := &mockChecker{output: verifier.VerifyOutput{Success: false}}
		deserializer := &mockDeserializer{proxy: proxy}
		writer := &mockWriter{}

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, writer.saved)
		assert.Contains(t, consumer.acked, "msg-1")
	})

	t.Run("handles deserialize error", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`invalid`)}
		close(messages)

		consumer := &mockConsumer{messages: messages}
		checker := &mockChecker{}
		deserializer := &mockDeserializer{err: errors.New("invalid json")}
		writer := &mockWriter{}

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, writer.saved)
		assert.Contains(t, consumer.acked, "msg-1")
	})

	t.Run("handles subscribe error", func(t *testing.T) {
		consumer := &mockConsumer{err: errors.New("redis connection failed")}
		checker := &mockChecker{}
		deserializer := &mockDeserializer{}
		writer := &mockWriter{}

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis connection failed")
	})

	t.Run("handles writer error gracefully", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`{}`)}
		close(messages)

		proxy := &stubProxy{address: "1.1.1.1:8080"}

		consumer := &mockConsumer{messages: messages}
		checker := &mockChecker{output: verifier.VerifyOutput{Success: true}}
		deserializer := &mockDeserializer{proxy: proxy}
		writer := &mockWriter{err: errors.New("save failed")}

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
		assert.Contains(t, consumer.acked, "msg-1")
	})
}
