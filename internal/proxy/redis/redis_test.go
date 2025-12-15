package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	proxyredis "github.com/JulianoL13/app-proxy-engine/internal/proxy/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := goredis.NewClient(&goredis.Options{
		Addr: endpoint,
	})
	defer client.Close()

	repo := proxyredis.NewRepository(client).WithTTL(5 * time.Minute)

	t.Run("saves proxy to redis", func(t *testing.T) {
		p := proxy.NewProxy("192.168.1.1", 8080, proxy.HTTP, "test-source")
		p.MarkSuccess(100*time.Millisecond, proxy.Elite)

		err := repo.Save(ctx, p)
		assert.NoError(t, err)

		exists, err := client.Exists(ctx, "proxy:192.168.1.1:8080").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		_, err = client.ZScore(ctx, "proxies:alive", "192.168.1.1:8080").Result()
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "proxies:protocol:http", "192.168.1.1:8080").Result()
		assert.NoError(t, err)
	})

	t.Run("saves socks5 proxy to correct protocol set", func(t *testing.T) {
		p := proxy.NewProxy("10.0.0.1", 1080, proxy.SOCKS5, "socks-source")
		p.MarkSuccess(50*time.Millisecond, proxy.Anonymous)

		err := repo.Save(ctx, p)
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "proxies:protocol:socks5", "10.0.0.1:1080").Result()
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "proxies:protocol:http", "10.0.0.1:1080").Result()
		assert.Error(t, err)
	})

	t.Run("overwrites existing proxy", func(t *testing.T) {
		p := proxy.NewProxy("192.168.2.2", 3128, proxy.HTTP, "source-v1")
		err := repo.Save(ctx, p)
		require.NoError(t, err)

		p2 := proxy.NewProxy("192.168.2.2", 3128, proxy.HTTP, "source-v2")
		p2.MarkSuccess(200*time.Millisecond, proxy.Elite)
		err = repo.Save(ctx, p2)
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "proxies:alive", "192.168.2.2:3128").Result()
		assert.NoError(t, err)
	})
}

func TestRepository_Save_ConnectionError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	client := goredis.NewClient(&goredis.Options{
		Addr:        "localhost:59999",
		DialTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	repo := proxyredis.NewRepository(client)

	p := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "test")
	err := repo.Save(ctx, p)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save proxy")
}
