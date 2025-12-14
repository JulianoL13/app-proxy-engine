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

// Compile-time check
var _ proxy.Writer = (*Repository)(nil)
