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

	pattern := fmt.Sprintf("%s:idx:*", c.keyPrefix)
	var keys []string
	var cursor uint64
	for {
		scanned, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		keys = append(keys, scanned...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	pipe := c.client.Pipeline()
	for _, key := range keys {
		pipe.ZRemRangeByScore(ctx, key, "-inf", now)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}

	return nil
}
