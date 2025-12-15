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

	score := float64(time.Now().UnixNano()) / 1e9

	pipe := r.client.Pipeline()

	pipe.Set(ctx, key, data, r.ttl)
	pipe.ZAdd(ctx, aliveSetKey, redis.Z{Score: score, Member: p.Address()})
	pipe.ZAdd(ctx, protocolPrefix+string(p.Protocol), redis.Z{Score: score, Member: p.Address()})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save proxy: %w", err)
	}

	return nil
}

func (r *Repository) GetAlive(ctx context.Context, cursor float64, limit int) ([]*proxy.Proxy, float64, int, error) {
	total, err := r.client.ZCard(ctx, aliveSetKey).Result()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("zcard: %w", err)
	}

	if total == 0 {
		return nil, 0, 0, nil
	}

	var addresses []string
	if limit > 0 {
		min := "-inf"
		if cursor > 0 {
			min = fmt.Sprintf("(%f", cursor)
		}

		addresses, err = r.client.ZRangeByScore(ctx, aliveSetKey, &redis.ZRangeBy{
			Min:   min,
			Max:   "+inf",
			Count: int64(limit),
		}).Result()
	} else {
		addresses, err = r.client.ZRange(ctx, aliveSetKey, 0, -1).Result()
	}

	if err != nil {
		return nil, 0, 0, fmt.Errorf("zrange: %w", err)
	}

	if len(addresses) == 0 {
		return nil, 0, int(total), nil
	}

	keys := make([]string, len(addresses))
	for i, addr := range addresses {
		keys[i] = proxyKeyPrefix + addr
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("mget proxies: %w", err)
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

	var nextCursor float64
	if limit > 0 && len(addresses) == limit {
		lastAddr := addresses[len(addresses)-1]
		score, err := r.client.ZScore(ctx, aliveSetKey, lastAddr).Result()
		if err == nil {
			nextCursor = score
		}
	}

	return proxies, nextCursor, int(total), nil
}
