package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hongik-backend/api"
	"hongik-backend/config"
	mw "hongik-backend/middleware"
	"hongik-backend/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func initLogger(level string) {
	var lvl slog.Level
	switch strings.ToUpper(level) {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARN":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})
	slog.SetDefault(slog.New(handler))
}

func main() {
	cfg := config.Load()
	initLogger(cfg.LogLevel)

	if _, err := os.Stat(cfg.InterpreterPath); os.IsNotExist(err) {
		slog.Warn("interpreter binary not found — /api/execute will fail",
			slog.String("path", cfg.InterpreterPath),
		)
	}

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	var store service.Store
	if cfg.DatabaseURL != "" {
		pgStore, err := service.NewPostgresStore(cfg.DatabaseURL)
		if err != nil {
			slog.Error("failed to connect to PostgreSQL", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer pgStore.Close()
		store = pgStore
		slog.Info("using PostgreSQL store")
	} else {
		store = service.NewStore()
		slog.Info("using in-memory store")
	}
	interpreter := service.NewInterpreterService(cfg)

	router := gin.New()

	// Request ID middleware (must come before logger)
	router.Use(mw.RequestID())

	// Request logging middleware (replaces default gin logger)
	router.Use(mw.RequestLogger())
	router.Use(gin.Recovery())

	// CORS — origins from environment variable
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// General API rate limit: 1 req/sec with burst of 60 (≈60 req/min)
	apiLimiter := mw.NewRateLimiter(rate.Limit(1), 60)
	router.Use(apiLimiter.Middleware())

	// Execute-specific rate limit: 0.5 req/sec with burst of 30 (≈30 req/min)
	executeLimiter := mw.NewRateLimiter(rate.Limit(0.5), 30)

	// Concurrent execution semaphore
	executeSemaphore := mw.ExecuteSemaphore(cfg.MaxConcurrent)

	api.RegisterRoutes(router, store, interpreter, cfg, executeLimiter.Middleware(), executeSemaphore)

	port := cfg.Port
	slog.Info("starting hong-ik backend",
		slog.String("port", port),
		slog.String("env", cfg.Env),
		slog.String("log_level", cfg.LogLevel),
		slog.Any("cors_origins", cfg.CORSOrigins),
		slog.Int("max_concurrent", cfg.MaxConcurrent),
	)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("server exited")
}
