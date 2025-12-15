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
	defaultTTL = 30 * time.Minute
)

type Repository struct {
	client    *redis.Client
	ttl       time.Duration
	keyPrefix string
}

func NewRepository(client *redis.Client, keyPrefix string) *Repository {
	if keyPrefix == "" {
		keyPrefix = "proxies"
	}
	return &Repository{
		client:    client,
		ttl:       defaultTTL,
		keyPrefix: keyPrefix,
	}
}

func (r *Repository) WithTTL(ttl time.Duration) *Repository {
	r.ttl = ttl
	return r
}

func (r *Repository) proxyKey(address string) string {
	return fmt.Sprintf("%s:data:%s", r.keyPrefix, address)
}

func (r *Repository) aliveSetKey() string {
	return fmt.Sprintf("%s:alive", r.keyPrefix)
}

func (r *Repository) protocolSetKey(protocol string) string {
	return fmt.Sprintf("%s:protocol:%s", r.keyPrefix, protocol)
}

func (r *Repository) anonymitySetKey(anonymity string) string {
	return fmt.Sprintf("%s:anonymity:%s", r.keyPrefix, anonymity)
}

func (r *Repository) Save(ctx context.Context, p *proxy.Proxy) error {
	key := r.proxyKey(p.Address())

	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal proxy: %w", err)
	}

	score := float64(time.Now().UnixNano()) / 1e9

	pipe := r.client.Pipeline()

	pipe.Set(ctx, key, data, r.ttl)
	pipe.ZAdd(ctx, r.aliveSetKey(), redis.Z{Score: score, Member: p.Address()})
	pipe.ZAdd(ctx, r.protocolSetKey(string(p.Protocol)), redis.Z{Score: score, Member: p.Address()})
	pipe.ZAdd(ctx, r.anonymitySetKey(string(p.Anonymity)), redis.Z{Score: score, Member: p.Address()})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save proxy: %w", err)
	}

	return nil
}

func (r *Repository) GetAlive(ctx context.Context, cursor float64, limit int, filter proxy.FilterOptions) ([]*proxy.Proxy, float64, int, error) {
	targetKey := r.aliveSetKey()

	if filter.Protocol != "" && filter.Anonymity == "" {
		targetKey = r.protocolSetKey(filter.Protocol)
	} else if filter.Anonymity != "" && filter.Protocol == "" {
		targetKey = r.anonymitySetKey(filter.Anonymity)
	}

	if filter.Protocol != "" && filter.Anonymity != "" {
		targetKey = r.protocolSetKey(filter.Protocol)
	}

	total, err := r.client.ZCard(ctx, targetKey).Result()
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

		addresses, err = r.client.ZRangeByScore(ctx, targetKey, &redis.ZRangeBy{
			Min:   min,
			Max:   "+inf",
			Count: int64(limit),
		}).Result()
	} else {
		addresses, err = r.client.ZRange(ctx, targetKey, 0, -1).Result()
	}

	if err != nil {
		return nil, 0, 0, fmt.Errorf("zrange: %w", err)
	}

	if len(addresses) == 0 {
		return nil, 0, int(total), nil
	}

	keys := make([]string, len(addresses))
	for i, addr := range addresses {
		keys[i] = r.proxyKey(addr)
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

		if filter.Anonymity != "" && string(p.Anonymity) != filter.Anonymity {
			continue
		}
		if filter.Protocol != "" && string(p.Protocol) != filter.Protocol {
			continue
		}

		proxies = append(proxies, &p)
	}

	var nextCursor float64
	if limit > 0 && len(addresses) == limit {
		lastAddr := addresses[len(addresses)-1]
		score, err := r.client.ZScore(ctx, targetKey, lastAddr).Result()
		if err == nil {
			nextCursor = score
		}
	}

	return proxies, nextCursor, int(total), nil
}
