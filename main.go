package main

import (
	"context"
	"log/slog"
	"net"
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

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Root context cancelled on SIGINT/SIGTERM. Propagated to background
	// goroutines (DB cleanup, etc.) so they exit cleanly during shutdown.
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	var store service.Store
	var pgStore *service.PostgresStore
	var memStore *service.InMemoryStore
	if cfg.DatabaseURL != "" {
		var err error
		pgStore, err = service.NewPostgresStore(rootCtx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("failed to connect to PostgreSQL", slog.String("error", err.Error()))
			os.Exit(1)
		}
		store = pgStore
		slog.Info("using PostgreSQL store")
	} else {
		memStore = service.NewStore()
		store = memStore
		slog.Info("using in-memory store")
	}

	// 사용자 코드 실행은 WASM-only로 클라이언트로 이전됨. 백엔드는 데이터 도메인(스니펫/공유/auth)만 담당.

	cache, err := service.NewCache(cfg)
	if err != nil {
		slog.Warn("Redis not available — caching disabled", slog.String("error", err.Error()))
	}
	if cache != nil {
		store = service.NewCachedStore(store, cache)
		slog.Info("using Redis cache")
	}

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

	// 코드 실행 전용 rate limiter / semaphore는 WASM-only 전환으로 더 이상 필요 없음.

	api.RegisterRoutes(router, store, cache, cfg)

	port := cfg.Port
	slog.Info("starting hong-ik backend",
		slog.String("port", port),
		slog.String("env", cfg.Env),
		slog.String("log_level", cfg.LogLevel),
		slog.Any("cors_origins", cfg.CORSOrigins),
	)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		// BaseContext ties every request's ctx to rootCtx, so a shutdown
		// signal cancels in-flight handler ctx (and any background work) instead of leaving them dangling.
		BaseContext: func(_ net.Listener) context.Context { return rootCtx },
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-rootCtx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			slog.Error("http server failed", slog.String("error", err.Error()))
		}
	}

	// Stop accepting new signals so a second Ctrl-C terminates immediately
	// instead of being swallowed by the shutdown context below.
	stopSignals()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	if cache != nil {
		if err := cache.Close(); err != nil {
			slog.Warn("cache close failed", slog.String("error", err.Error()))
		}
	}
	if pgStore != nil {
		if err := pgStore.Close(); err != nil {
			slog.Warn("postgres close failed", slog.String("error", err.Error()))
		}
	}
	if memStore != nil {
		if err := memStore.Close(); err != nil {
			slog.Warn("in-memory store close failed", slog.String("error", err.Error()))
		}
	}

	slog.Info("server exited")
}
