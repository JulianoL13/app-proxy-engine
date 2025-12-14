package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/redis/go-redis/v9"
)

const (
	proxyKeyPrefix = "proxy:"
	aliveSetKey    = "proxies:alive"
	protocolPrefix = "proxies:protocol:"
	defaultTTL     = 30 * time.Minute
)

type Repository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRepository(client *redis.Client) *Repository {
	return &Repository{
		client: client,
		ttl:    defaultTTL,
	}
}

func (r *Repository) WithTTL(ttl time.Duration) *Repository {
	r.ttl = ttl
	return r
}

func (r *Repository) Save(ctx context.Context, p *proxy.Proxy) error {
	key := proxyKeyPrefix + p.Address()

	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal proxy: %w", err)
	}

	pipe := r.client.Pipeline()

	pipe.Set(ctx, key, data, r.ttl)
	pipe.SAdd(ctx, aliveSetKey, p.Address())
	pipe.SAdd(ctx, protocolPrefix+string(p.Protocol), p.Address())

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save proxy: %w", err)
	}

	return nil
}

func (r *Repository) GetAlive(ctx context.Context) ([]*proxy.Proxy, error) {
	addresses, err := r.client.SMembers(ctx, aliveSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get alive set: %w", err)
	}

	if len(addresses) == 0 {
		return nil, nil
	}

	keys := make([]string, len(addresses))
	for i, addr := range addresses {
		keys[i] = proxyKeyPrefix + addr
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("mget proxies: %w", err)
	}

	proxies := make([]*proxy.Proxy, 0, len(values))
	for _, v := range values {
		if v == nil {
			continue
		}

		str, ok := v.(string)
		if !ok {
			continue
		}

		var p proxy.Proxy
		if err := json.Unmarshal([]byte(str), &p); err != nil {
			continue
		}

		proxies = append(proxies, &p)
	}

	return proxies, nil
}

// Compile-time checks
var (
	_ proxy.Writer = (*Repository)(nil)
	_ proxy.Reader = (*Repository)(nil)
)
