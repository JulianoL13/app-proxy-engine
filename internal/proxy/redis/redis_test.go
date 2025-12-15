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

	repo := proxyredis.NewRepository(client, "test-prefix").WithTTL(5 * time.Minute)

	t.Run("saves proxy to redis", func(t *testing.T) {
		p := proxy.NewProxy("192.168.1.1", 8080, proxy.HTTP, "test-source")
		p.MarkSuccess(100*time.Millisecond, proxy.Elite)

		err := repo.Save(ctx, p)
		assert.NoError(t, err)

		exists, err := client.Exists(ctx, "test-prefix:data:192.168.1.1:8080").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		_, err = client.ZScore(ctx, "test-prefix:alive", "192.168.1.1:8080").Result()
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "test-prefix:protocol:http", "192.168.1.1:8080").Result()
		assert.NoError(t, err)
	})

	t.Run("saves socks5 proxy to correct protocol set", func(t *testing.T) {
		p := proxy.NewProxy("10.0.0.1", 1080, proxy.SOCKS5, "socks-source")
		p.MarkSuccess(50*time.Millisecond, proxy.Anonymous)

		err := repo.Save(ctx, p)
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "test-prefix:protocol:socks5", "10.0.0.1:1080").Result()
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "test-prefix:protocol:http", "10.0.0.1:1080").Result()
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

		_, err = client.ZScore(ctx, "test-prefix:alive", "192.168.2.2:3128").Result()
		assert.NoError(t, err)
	})
}

func TestRepository_Save_Anonymity(t *testing.T) {
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

	repo := proxyredis.NewRepository(client, "test-prefix")

	t.Run("saves anonymity index", func(t *testing.T) {
		p := proxy.NewProxy("10.0.0.2", 8080, proxy.HTTP, "test")
		p.MarkSuccess(100*time.Millisecond, proxy.Transparent)

		err := repo.Save(ctx, p)
		assert.NoError(t, err)

		_, err = client.ZScore(ctx, "test-prefix:anonymity:transparent", "10.0.0.2:8080").Result()
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

	repo := proxyredis.NewRepository(client, "test-prefix")

	p := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "test")
	err := repo.Save(ctx, p)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save proxy")
}
func TestRepository_GetAlive(t *testing.T) {
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

	repo := proxyredis.NewRepository(client, "test-prefix")

	p1 := proxy.NewProxy("1.1.1.1", 80, proxy.HTTP, "s1")
	p1.MarkSuccess(100*time.Millisecond, proxy.Elite)
	repo.Save(ctx, p1)

	p2 := proxy.NewProxy("2.2.2.2", 1080, proxy.SOCKS5, "s1")
	p2.MarkSuccess(50*time.Millisecond, proxy.Anonymous)
	repo.Save(ctx, p2)

	p3 := proxy.NewProxy("3.3.3.3", 8080, proxy.HTTP, "s1")
	p3.MarkSuccess(200*time.Millisecond, proxy.Transparent)
	repo.Save(ctx, p3)

	t.Run("returns all alive proxies without filter", func(t *testing.T) {
		proxies, _, total, err := repo.GetAlive(ctx, 0, 10, proxy.FilterOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, proxies, 3)
	})

	t.Run("filters by protocol", func(t *testing.T) {
		proxies, _, total, err := repo.GetAlive(ctx, 0, 10, proxy.FilterOptions{Protocol: "socks5"})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, proxies, 1)
		assert.Equal(t, "2.2.2.2:1080", proxies[0].Address())
	})

	t.Run("filters by anonymity", func(t *testing.T) {
		proxies, _, total, err := repo.GetAlive(ctx, 0, 10, proxy.FilterOptions{Anonymity: "elite"})
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, proxies, 1)
		assert.Equal(t, "1.1.1.1:80", proxies[0].Address())
	})

	t.Run("filters by protocol and anonymity", func(t *testing.T) {
		proxies, _, total, err := repo.GetAlive(ctx, 0, 10, proxy.FilterOptions{
			Protocol:  "http",
			Anonymity: "transparent",
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Equal(t, 1, len(proxies))
		assert.Equal(t, "3.3.3.3:8080", proxies[0].Address())
	})
}
