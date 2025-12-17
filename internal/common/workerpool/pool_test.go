package workerpool_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/common/workerpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_New(t *testing.T) {
	t.Run("creates pool with valid size", func(t *testing.T) {
		pool, err := workerpool.New(10)
		require.NoError(t, err)
		assert.NotNil(t, pool)
		assert.Equal(t, 10, pool.Workers())
		pool.Stop()
	})
}

func TestPool_Submit(t *testing.T) {
	t.Run("executes job with context", func(t *testing.T) {
		pool, err := workerpool.New(2)
		require.NoError(t, err)
		defer pool.Stop()

		var executed atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)

		err = pool.Submit(context.Background(), func(ctx context.Context) {
			executed.Store(true)
			wg.Done()
		})
		require.NoError(t, err)

		wg.Wait()
		assert.True(t, executed.Load())
	})

	t.Run("skips job when context is cancelled", func(t *testing.T) {
		pool, err := workerpool.New(1)
		require.NoError(t, err)
		defer pool.Stop()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var executed atomic.Bool
		err = pool.Submit(ctx, func(ctx context.Context) {
			executed.Store(true)
		})
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		assert.False(t, executed.Load())
	})

	t.Run("propagates context to job", func(t *testing.T) {
		pool, err := workerpool.New(2)
		require.NoError(t, err)
		defer pool.Stop()

		type contextKey string
		const key contextKey = "key"
		ctx := context.WithValue(context.Background(), key, "value")
		var receivedValue string
		var wg sync.WaitGroup
		wg.Add(1)

		err = pool.Submit(ctx, func(ctx context.Context) {
			receivedValue = ctx.Value(key).(string)
			wg.Done()
		})
		require.NoError(t, err)

		wg.Wait()
		assert.Equal(t, "value", receivedValue)
	})
}

func TestPool_Stop(t *testing.T) {
	t.Run("stops accepting new jobs", func(t *testing.T) {
		pool, err := workerpool.New(2)
		require.NoError(t, err)

		pool.Stop()

		err = pool.Submit(context.Background(), func(ctx context.Context) {})
		assert.Error(t, err)
	})
}
