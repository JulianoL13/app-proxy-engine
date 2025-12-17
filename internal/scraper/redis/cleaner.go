package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cleaner struct {
	client    *redis.Client
	keyPrefix string
}

func NewCleaner(client *redis.Client, keyPrefix string) *Cleaner {
	if keyPrefix == "" {
		keyPrefix = "proxies"
	}
	return &Cleaner{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

func (c *Cleaner) Cleanup(ctx context.Context) error {
	now := fmt.Sprintf("%f", float64(time.Now().Unix()))

	baseKeys := []string{
		fmt.Sprintf("%s:idx:alive", c.keyPrefix),
		fmt.Sprintf("%s:idx:latency", c.keyPrefix),
	}

	pattern := fmt.Sprintf("%s:idx:*", c.keyPrefix)
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		baseKeys = append(baseKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	pipe := c.client.Pipeline()
	for _, key := range baseKeys {
		pipe.ZRemRangeByScore(ctx, key, "-inf", now)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}

	return nil
}
