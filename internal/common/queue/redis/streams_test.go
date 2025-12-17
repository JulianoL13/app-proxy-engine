package redis_test

import (
	"context"
	"testing"
	"time"

	queueredis "github.com/JulianoL13/app-proxy-engine/internal/common/queue/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestStreamsClient_Publish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := goredis.NewClient(&goredis.Options{Addr: endpoint})
	defer client.Close()

	streams := queueredis.NewStreamsClient(client)

	t.Run("publishes message to stream", func(t *testing.T) {
		err := streams.Publish(ctx, "test-topic", []byte(`{"test":"data"}`))
		assert.NoError(t, err)

		info, err := client.XInfoStream(ctx, "test-topic").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), info.Length)
	})
}

func TestStreamsClient_Subscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := goredis.NewClient(&goredis.Options{Addr: endpoint})
	defer client.Close()

	streams := queueredis.NewStreamsClient(client)

	t.Run("receives published messages", func(t *testing.T) {
		topic := "sub-test"
		group := "test-group"
		consumer := "test-consumer"

		subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		messages, err := streams.Subscribe(subCtx, topic, group, consumer)
		require.NoError(t, err)

		err = streams.Publish(ctx, topic, []byte(`{"msg":"hello"}`))
		require.NoError(t, err)

		select {
		case msg := <-messages:
			assert.Equal(t, `{"msg":"hello"}`, string(msg.Payload))
			err = streams.Ack(ctx, topic, group, msg.ID)
			assert.NoError(t, err)
		case <-subCtx.Done():
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("recovers pending messages", func(t *testing.T) {
		topic := "pending-test"
		group := "pending-group"
		consumer := "pending-consumer"

		err := streams.Publish(ctx, topic, []byte(`{"pending":"msg"}`))
		require.NoError(t, err)

		firstCtx, firstCancel := context.WithTimeout(ctx, 2*time.Second)
		messages, err := streams.Subscribe(firstCtx, topic, group, consumer)
		require.NoError(t, err)

		select {
		case <-messages:
		case <-firstCtx.Done():
			t.Fatal("timeout on first subscribe")
		}
		firstCancel()

		time.Sleep(100 * time.Millisecond)

		secondCtx, secondCancel := context.WithTimeout(ctx, 2*time.Second)
		defer secondCancel()

		messages, err = streams.Subscribe(secondCtx, topic, group, consumer)
		require.NoError(t, err)

		select {
		case msg := <-messages:
			assert.Equal(t, `{"pending":"msg"}`, string(msg.Payload))
			err = streams.Ack(ctx, topic, group, msg.ID)
			assert.NoError(t, err)
		case <-secondCtx.Done():
			t.Fatal("timeout on recovery")
		}
	})
}
