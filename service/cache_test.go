package service

import (
	"testing"

	"hongik-backend/config"
	"hongik-backend/model"
)

func TestNilCacheGetExecuteResult(t *testing.T) {
	var c *Cache
	resp, ok := c.GetExecuteResult(model.ExecuteRequest{Code: "출력(1)"})
	if ok {
		t.Fatal("expected cache miss on nil cache")
	}
	if resp.Status != "" {
		t.Fatal("expected zero-value response")
	}
}

func TestNilCacheSetExecuteResult(t *testing.T) {
	var c *Cache
	// Should not panic
	c.SetExecuteResult(
		model.ExecuteRequest{Code: "출력(1)"},
		model.ExecuteResponse{Status: "success", Output: "1"},
	)
}

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

func TestExecuteKeyDeterministic(t *testing.T) {
	k1 := executeKey("출력(1)", "", 5)
	k2 := executeKey("출력(1)", "", 5)
	if k1 != k2 {
		t.Fatalf("expected same key, got %s vs %s", k1, k2)
	}
}

func TestExecuteKeyDifferentCode(t *testing.T) {
	k1 := executeKey("출력(1)", "", 5)
	k2 := executeKey("출력(2)", "", 5)
	if k1 == k2 {
		t.Fatal("expected different keys for different code")
	}
}

func TestExecuteKeyDifferentInput(t *testing.T) {
	k1 := executeKey("입력()", "a", 5)
	k2 := executeKey("입력()", "b", 5)
	if k1 == k2 {
		t.Fatal("expected different keys for different input")
	}
}

func TestExecuteKeyDifferentTimeout(t *testing.T) {
	k1 := executeKey("출력(1)", "", 5)
	k2 := executeKey("출력(1)", "", 10)
	if k1 == k2 {
		t.Fatal("expected different keys for different timeout")
	}
}

func TestExecuteKeyPrefix(t *testing.T) {
	key := executeKey("test", "", 5)
	if len(key) < 10 || key[:5] != "exec:" {
		t.Fatalf("expected key with 'exec:' prefix, got %s", key)
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
