package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port            string
	Env             string
	InterpreterPath string
	ExecuteTimeout  int // seconds
	CORSOrigins     []string
	MaxConcurrent   int // max concurrent execute requests
}

func Load() *Config {
	origins := getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")

	return &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		InterpreterPath: getEnv("INTERPRETER_PATH", "../hong-ik/cmake-build-debug/HongIk"),
		ExecuteTimeout:  5,
		CORSOrigins:     parseOrigins(origins),
		MaxConcurrent:   getEnvInt("MAX_CONCURRENT_EXEC", 5),
	}
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
