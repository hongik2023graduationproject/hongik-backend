package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"hongik-backend/config"

	"github.com/redis/go-redis/v9"
)

// Cache provides Redis-backed caching for expensive operations.
type Cache struct {
	client *redis.Client
	cfg    *config.Config
}

// NewCache creates a new Cache connected to Redis.
// Returns nil if the Redis URL is empty (caching disabled).
func NewCache(cfg *config.Config) (*Cache, error) {
	if cfg.RedisURL == "" {
		return nil, nil
	}

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Cache{client: client, cfg: cfg}, nil
}

// Close shuts down the Redis connection.
func (c *Cache) Close() error {
	if c == nil {
		return nil
	}
	return c.client.Close()
}

// 코드 실행 결과 캐싱(executeKey, GetExecuteResult, SetExecuteResult)은
// WASM-only 전환으로 백엔드에서 코드 실행 자체가 사라져 제거되었다.

// Get retrieves a cached value by key.
func (c *Cache) Get(key string, dest any) bool {
	if c == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false
	}
	return true
}

// Set stores a value in the cache with the data TTL.
func (c *Cache) Set(key string, value any) {
	if c == nil {
		return
	}

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ttl := time.Duration(c.cfg.CacheTTLData) * time.Second
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		slog.Warn("cache set failed", slog.String("error", err.Error()))
	}
}

// Delete removes one or more keys from the cache.
func (c *Cache) Delete(keys ...string) {
	if c == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	c.client.Del(ctx, keys...)
}

// DeleteByPrefix removes all keys matching a prefix.
func (c *Cache) DeleteByPrefix(prefix string) {
	if c == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	iter := c.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		c.client.Del(ctx, keys...)
	}
}
