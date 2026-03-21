package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID assigns a unique ID to each request and stores it in the context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.New().String()
		c.Set("requestID", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// RequestLogger logs each HTTP request with structured fields using slog.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get("requestID")

		attrs := []slog.Attr{
			slog.Int("status", status),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		}
		if rid, ok := requestID.(string); ok && rid != "" {
			attrs = append(attrs, slog.String("request_id", rid))
		}
		if c.Request.URL.RawQuery != "" {
			attrs = append(attrs, slog.String("query", c.Request.URL.RawQuery))
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "HTTP request",
			attrs...,
		)
	}
}
