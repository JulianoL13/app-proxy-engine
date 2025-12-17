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
	return fmt.Sprintf("%s:idx:alive", r.keyPrefix)
}

func (r *Repository) protocolSetKey(protocol string) string {
	return fmt.Sprintf("%s:idx:proto:%s", r.keyPrefix, protocol)
}

func (r *Repository) anonymitySetKey(anonymity string) string {
	return fmt.Sprintf("%s:idx:anon:%s", r.keyPrefix, anonymity)
}

func (r *Repository) compositeSetKey(protocol, anonymity string) string {
	return fmt.Sprintf("%s:idx:proto:%s:anon:%s", r.keyPrefix, protocol, anonymity)
}

func (r *Repository) latencySetKey() string {
	return fmt.Sprintf("%s:idx:latency", r.keyPrefix)
}

func (r *Repository) Save(ctx context.Context, p *proxy.Proxy) error {
	key := r.proxyKey(p.Address())

	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal proxy: %w", err)
	}

	expirationScore := float64(time.Now().Add(r.ttl).Unix())
	latencyScore := float64(p.Latency.Milliseconds())

	pipe := r.client.Pipeline()

	pipe.Set(ctx, key, data, r.ttl)

	pipe.ZAdd(ctx, r.aliveSetKey(), redis.Z{Score: expirationScore, Member: p.Address()})
	pipe.ZAdd(ctx, r.protocolSetKey(string(p.Protocol)), redis.Z{Score: expirationScore, Member: p.Address()})
	pipe.ZAdd(ctx, r.anonymitySetKey(string(p.Anonymity)), redis.Z{Score: expirationScore, Member: p.Address()})
	pipe.ZAdd(ctx, r.compositeSetKey(string(p.Protocol), string(p.Anonymity)), redis.Z{Score: expirationScore, Member: p.Address()})

	pipe.ZAdd(ctx, r.latencySetKey(), redis.Z{Score: latencyScore, Member: p.Address()})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save proxy: %w", err)
	}

	return nil
}

func (r *Repository) GetAlive(ctx context.Context, cursor float64, limit int, filter proxy.FilterOptions) ([]*proxy.Proxy, float64, int, error) {
	targetKey := r.selectIndex(filter)
	now := float64(time.Now().Unix())

	total, err := r.client.ZCount(ctx, targetKey, fmt.Sprintf("%f", now), "+inf").Result()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("zcount: %w", err)
	}

	if total == 0 {
		return nil, 0, 0, nil
	}

	min := now
	if cursor > min {
		min = cursor
	}
	minStr := fmt.Sprintf("%f", min)
	if cursor > 0 {
		minStr = fmt.Sprintf("(%f", cursor)
	}

	var results []redis.Z
	if limit > 0 {
		results, err = r.client.ZRangeByScoreWithScores(ctx, targetKey, &redis.ZRangeBy{
			Min:   minStr,
			Max:   "+inf",
			Count: int64(limit),
		}).Result()
	} else {
		results, err = r.client.ZRangeByScoreWithScores(ctx, targetKey, &redis.ZRangeBy{
			Min: fmt.Sprintf("%f", now),
			Max: "+inf",
		}).Result()
	}

	if err != nil {
		return nil, 0, 0, fmt.Errorf("zrangebyscore: %w", err)
	}

	if len(results) == 0 {
		return nil, 0, int(total), nil
	}

	addresses := make([]string, len(results))
	var lastScore float64
	for i, z := range results {
		addresses[i] = z.Member.(string)
		lastScore = z.Score
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

		if filter.MaxLatency > 0 && p.Latency > filter.MaxLatency {
			continue
		}

		proxies = append(proxies, &p)
	}

	var nextCursor float64
	if limit > 0 && len(results) == limit {
		nextCursor = lastScore
	}

	return proxies, nextCursor, int(total), nil
}

func (r *Repository) selectIndex(filter proxy.FilterOptions) string {
	hasProtocol := filter.Protocol != ""
	hasAnonymity := filter.Anonymity != ""

	switch {
	case hasProtocol && hasAnonymity:
		return r.compositeSetKey(filter.Protocol, filter.Anonymity)
	case hasProtocol:
		return r.protocolSetKey(filter.Protocol)
	case hasAnonymity:
		return r.anonymitySetKey(filter.Anonymity)
	default:
		return r.aliveSetKey()
	}
}
