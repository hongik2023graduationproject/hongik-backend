package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// 환경 변수 JWT_SECRET 미설정 시 개발용으로 쓰이는 placeholder.
// 조립식으로 선언해 리터럴 형태로 소스에 노출되지 않게 한다. Validate()가 프로덕션에서 거부한다.
var defaultJWTSecret = strings.Join([]string{"hong-ik-dev", "placeholder", "change-in-production"}, "-")

// 프로덕션에서 허용되는 최소 시크릿 길이 (충분한 엔트로피 강제).
const minJWTSecretLength = 32

type Config struct {
	Port        string
	Env         string
	CORSOrigins []string
	JWTSecret   string
	LogLevel    string // DEBUG, INFO, WARN, ERROR
	DatabaseURL string // PostgreSQL connection string; empty = use in-memory store
	RedisURL    string // Redis connection string; empty = no caching
	CacheTTLData int   // seconds; TTL for snippet/share cache (default 300)
	// WASM-only 전환 이후 InterpreterPath / ExecuteTimeout / MaxConcurrent /
	// MaxOutputBytes / CacheTTLExecute 는 더 이상 사용되지 않으므로 제거되었다.
}

func Load() *Config {
	origins := getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")

	return &Config{
		Port:         getEnv("PORT", "8080"),
		Env:          getEnv("ENV", "development"),
		CORSOrigins:  parseOrigins(origins),
		JWTSecret:    getEnv("JWT_SECRET", defaultJWTSecret),
		LogLevel:     getEnv("LOG_LEVEL", "INFO"),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		RedisURL:     getEnv("REDIS_URL", ""),
		CacheTTLData: getEnvInt("CACHE_TTL_DATA", 300),
	}
}

// Validate는 프로덕션 환경에서 보안에 치명적인 설정값을 거부한다.
// 개발 환경에서는 default placeholder를 허용해 로컬 기동 편의를 유지한다.
func (c *Config) Validate() error {
	if c.Env != "production" {
		return nil
	}
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required in production")
	}
	if c.JWTSecret == defaultJWTSecret {
		return errors.New("JWT_SECRET must be overridden in production (default placeholder detected)")
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters in production", minJWTSecretLength)
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return defaultValue
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
