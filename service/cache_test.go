package service

import (
	"testing"

	"hongik-backend/config"
)

// Execute-related cache tests (executeKey, GetExecuteResult, SetExecuteResult, NilCacheGetExecuteResult, NilCacheSetExecuteResult)는
// WASM-only 전환으로 코드 실행 캐시 자체가 제거되어 함께 삭제되었다.

func TestNilCacheGet(t *testing.T) {
	var c *Cache
	var dest string
	if c.Get("key", &dest) {
		t.Fatal("expected false on nil cache")
	}
}

func TestNilCacheSet(t *testing.T) {
	var c *Cache
	// Should not panic
	c.Set("key", "value")
}

func TestNilCacheDelete(t *testing.T) {
	var c *Cache
	// Should not panic
	c.Delete("key1", "key2")
}

func TestNilCacheDeleteByPrefix(t *testing.T) {
	var c *Cache
	// Should not panic
	c.DeleteByPrefix("prefix:")
}

func TestNilCacheClose(t *testing.T) {
	var c *Cache
	if err := c.Close(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNewCacheEmptyURL(t *testing.T) {
	cfg := &config.Config{RedisURL: ""}
	c, err := NewCache(cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if c != nil {
		t.Fatal("expected nil cache for empty URL")
	}
}

func TestNewCacheInvalidURL(t *testing.T) {
	cfg := &config.Config{RedisURL: "not-a-valid-url"}
	_, err := NewCache(cfg)
	if err == nil {
		t.Fatal("expected error for invalid Redis URL")
	}
}
