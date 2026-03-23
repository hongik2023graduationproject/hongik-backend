package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"hongik-backend/config"
	"hongik-backend/model"

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

// executeKey returns a deterministic cache key for code execution.
func executeKey(code, input string, timeout int) string {
	h := sha256.New()
	h.Write([]byte(code))
	h.Write([]byte{0})
	h.Write([]byte(input))
	h.Write([]byte{0})
	h.Write([]byte(fmt.Sprintf("%d", timeout)))
	return fmt.Sprintf("exec:%x", h.Sum(nil))
}

// GetExecuteResult looks up a cached execution result.
func (c *Cache) GetExecuteResult(req model.ExecuteRequest) (model.ExecuteResponse, bool) {
	if c == nil {
		return model.ExecuteResponse{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	key := executeKey(req.Code, req.Input, req.Timeout)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return model.ExecuteResponse{}, false
	}

	var resp model.ExecuteResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return model.ExecuteResponse{}, false
	}

	slog.Debug("cache hit", slog.String("key", key[:20]))
	return resp, true
}

// SetExecuteResult caches a successful execution result.
func (c *Cache) SetExecuteResult(req model.ExecuteRequest, resp model.ExecuteResponse) {
	if c == nil {
		return
	}
	// Only cache successful results
	if resp.Status != "success" {
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	key := executeKey(req.Code, req.Input, req.Timeout)
	ttl := time.Duration(c.cfg.CacheTTLExecute) * time.Second
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		slog.Warn("cache set failed", slog.String("error", err.Error()))
	}
}

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
