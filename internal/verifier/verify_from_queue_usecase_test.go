package verifier_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier/mocks"
)

type verifierTestLogger struct{}

func (l verifierTestLogger) Info(msg string, args ...any)  {}
func (l verifierTestLogger) Warn(msg string, args ...any)  {}
func (l verifierTestLogger) Debug(msg string, args ...any) {}

func TestVerifyFromQueueUseCase_Execute(t *testing.T) {
	logger := verifierTestLogger{}

	t.Run("processes and saves successful proxy", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`{}`)}
		close(messages)

		consumer := mocks.NewConsumer(t)
		consumer.EXPECT().
			Subscribe(mock.Anything, "test-topic", "test-group", "test-worker").
			Return((<-chan verifier.Message)(messages), nil)
		consumer.EXPECT().
			Ack(mock.Anything, "test-topic", "test-group", "msg-1").
			Return(nil)

		proxyMock := mocks.NewVerifiedProxy(t)
		proxyMock.EXPECT().Address().Return("1.1.1.1:8080").Maybe()
		proxyMock.EXPECT().URL().Return(&url.URL{}).Maybe()
		proxyMock.EXPECT().MarkSuccess(100*time.Millisecond, "elite").Return().Maybe()

		deserializer := mocks.NewProxyDeserializer(t)
		deserializer.EXPECT().
			Deserialize([]byte(`{}`)).
			Return(proxyMock, nil)

		checker := mocks.NewProxyChecker(t)
		checker.EXPECT().
			Verify(mock.Anything, proxyMock).
			Return(verifier.VerifyOutput{
				Success:   true,
				Latency:   100 * time.Millisecond,
				Anonymity: "elite",
			})

		writer := mocks.NewWriter(t)
		writer.EXPECT().
			Save(mock.Anything, proxyMock).
			Return(nil)

		pool := mocks.NewWorkerPool(t)
		pool.EXPECT().
			Submit(mock.Anything, mock.AnythingOfType("func(context.Context)")).
			RunAndReturn(func(ctx context.Context, job func(context.Context)) error {
				job(ctx)
				return nil
			})

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
	})

	t.Run("acks but does not save failed proxy", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`{}`)}
		close(messages)

		consumer := mocks.NewConsumer(t)
		consumer.EXPECT().
			Subscribe(mock.Anything, "test-topic", "test-group", "test-worker").
			Return((<-chan verifier.Message)(messages), nil)
		consumer.EXPECT().
			Ack(mock.Anything, "test-topic", "test-group", "msg-1").
			Return(nil)

		proxyMock := mocks.NewVerifiedProxy(t)
		proxyMock.EXPECT().Address().Return("1.1.1.1:8080").Maybe()
		proxyMock.EXPECT().URL().Return(&url.URL{}).Maybe()

		deserializer := mocks.NewProxyDeserializer(t)
		deserializer.EXPECT().
			Deserialize([]byte(`{}`)).
			Return(proxyMock, nil)

		checker := mocks.NewProxyChecker(t)
		checker.EXPECT().
			Verify(mock.Anything, proxyMock).
			Return(verifier.VerifyOutput{Success: false})

		writer := mocks.NewWriter(t)

		pool := mocks.NewWorkerPool(t)
		pool.EXPECT().
			Submit(mock.Anything, mock.AnythingOfType("func(context.Context)")).
			RunAndReturn(func(ctx context.Context, job func(context.Context)) error {
				job(ctx)
				return nil
			})

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
	})

	t.Run("handles deserialize error", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`invalid`)}
		close(messages)

		consumer := mocks.NewConsumer(t)
		consumer.EXPECT().
			Subscribe(mock.Anything, "test-topic", "test-group", "test-worker").
			Return((<-chan verifier.Message)(messages), nil)
		consumer.EXPECT().
			Ack(mock.Anything, "test-topic", "test-group", "msg-1").
			Return(nil)

		deserializer := mocks.NewProxyDeserializer(t)
		deserializer.EXPECT().
			Deserialize([]byte(`invalid`)).
			Return(nil, errors.New("invalid json"))

		checker := mocks.NewProxyChecker(t)
		writer := mocks.NewWriter(t)

		pool := mocks.NewWorkerPool(t)
		pool.EXPECT().
			Submit(mock.Anything, mock.AnythingOfType("func(context.Context)")).
			RunAndReturn(func(ctx context.Context, job func(context.Context)) error {
				job(ctx)
				return nil
			})

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
	})

	t.Run("handles subscribe error", func(t *testing.T) {
		consumer := mocks.NewConsumer(t)
		consumer.EXPECT().
			Subscribe(mock.Anything, "test-topic", "test-group", "test-worker").
			Return(nil, errors.New("redis connection failed"))

		checker := mocks.NewProxyChecker(t)
		deserializer := mocks.NewProxyDeserializer(t)
		writer := mocks.NewWriter(t)
		pool := mocks.NewWorkerPool(t)

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis connection failed")
	})

	t.Run("handles writer error gracefully", func(t *testing.T) {
		messages := make(chan verifier.Message, 1)
		messages <- verifier.Message{ID: "msg-1", Payload: []byte(`{}`)}
		close(messages)

		consumer := mocks.NewConsumer(t)
		consumer.EXPECT().
			Subscribe(mock.Anything, "test-topic", "test-group", "test-worker").
			Return((<-chan verifier.Message)(messages), nil)
		consumer.EXPECT().
			Ack(mock.Anything, "test-topic", "test-group", "msg-1").
			Return(nil)

		proxyMock := mocks.NewVerifiedProxy(t)
		proxyMock.EXPECT().Address().Return("1.1.1.1:8080").Maybe()
		proxyMock.EXPECT().URL().Return(&url.URL{}).Maybe()
		proxyMock.EXPECT().MarkSuccess(mock.Anything, mock.Anything).Return().Maybe()

		deserializer := mocks.NewProxyDeserializer(t)
		deserializer.EXPECT().
			Deserialize([]byte(`{}`)).
			Return(proxyMock, nil)

		checker := mocks.NewProxyChecker(t)
		checker.EXPECT().
			Verify(mock.Anything, proxyMock).
			Return(verifier.VerifyOutput{Success: true})

		writer := mocks.NewWriter(t)
		writer.EXPECT().
			Save(mock.Anything, proxyMock).
			Return(errors.New("save failed"))

		pool := mocks.NewWorkerPool(t)
		pool.EXPECT().
			Submit(mock.Anything, mock.AnythingOfType("func(context.Context)")).
			RunAndReturn(func(ctx context.Context, job func(context.Context)) error {
				job(ctx)
				return nil
			})

		uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, pool, "test-worker", "test-topic", "test-group")

		err := uc.Execute(context.Background())

		assert.NoError(t, err)
	})
}
