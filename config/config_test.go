package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env vars that might interfere
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("CORS_ORIGINS")
	_ = os.Unsetenv("MAX_CONCURRENT_EXEC")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected env development, got %s", cfg.Env)
	}
	if cfg.ExecuteTimeout != 5 {
		t.Errorf("expected timeout 5, got %d", cfg.ExecuteTimeout)
	}
	if cfg.MaxConcurrent != 5 {
		t.Errorf("expected max concurrent 5, got %d", cfg.MaxConcurrent)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("expected 2 CORS origins, got %d", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected first origin http://localhost:3000, got %s", cfg.CORSOrigins[0])
	}
	if cfg.CORSOrigins[1] != "http://localhost:5173" {
		t.Errorf("expected second origin http://localhost:5173, got %s", cfg.CORSOrigins[1])
	}
}

func TestLoadFromEnv(t *testing.T) {
	_ = os.Setenv("PORT", "9090")
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("CORS_ORIGINS", "https://example.com,https://app.example.com")
	_ = os.Setenv("MAX_CONCURRENT_EXEC", "10")
	defer func() {
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("ENV")
		_ = os.Unsetenv("CORS_ORIGINS")
		_ = os.Unsetenv("MAX_CONCURRENT_EXEC")
	}()

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("expected env production, got %s", cfg.Env)
	}
	if cfg.MaxConcurrent != 10 {
		t.Errorf("expected max concurrent 10, got %d", cfg.MaxConcurrent)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("expected 2 CORS origins, got %d", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "https://example.com" {
		t.Errorf("expected first origin https://example.com, got %s", cfg.CORSOrigins[0])
	}
}

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"http://a.com", []string{"http://a.com"}},
		{"http://a.com,http://b.com", []string{"http://a.com", "http://b.com"}},
		{"http://a.com , http://b.com", []string{"http://a.com", "http://b.com"}},
		{" , , ", []string{}},
		{"", []string{}},
	}

	for _, tt := range tests {
		result := parseOrigins(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseOrigins(%q): expected %d items, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("parseOrigins(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], v)
			}
		}
	}
}

func TestGetEnvInt(t *testing.T) {
	_ = os.Setenv("TEST_INT", "42")
	defer func() { _ = os.Unsetenv("TEST_INT") }()

	if v := getEnvInt("TEST_INT", 0); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
	if v := getEnvInt("NONEXISTENT", 99); v != 99 {
		t.Errorf("expected 99, got %d", v)
	}

	_ = os.Setenv("TEST_INT_BAD", "not_a_number")
	defer func() { _ = os.Unsetenv("TEST_INT_BAD") }()
	if v := getEnvInt("TEST_INT_BAD", 7); v != 7 {
		t.Errorf("expected 7 for invalid int, got %d", v)
	}
}
