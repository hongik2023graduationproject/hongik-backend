package main

import (
	"log"
	"os"

	"hongik-backend/api"
	"hongik-backend/config"
	mw "hongik-backend/middleware"
	"hongik-backend/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.Load()

	if _, err := os.Stat(cfg.InterpreterPath); os.IsNotExist(err) {
		log.Printf("WARNING: interpreter binary not found at %s — /api/execute will fail", cfg.InterpreterPath)
	}

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	store := service.NewStore()
	interpreter := service.NewInterpreterService(cfg)

	router := gin.Default()

	// CORS — origins from environment variable
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// General API rate limit: 1 req/sec with burst of 60 (≈60 req/min)
	apiLimiter := mw.NewRateLimiter(rate.Limit(1), 60)
	router.Use(apiLimiter.Middleware())

	// Execute-specific rate limit: 0.5 req/sec with burst of 30 (≈30 req/min)
	executeLimiter := mw.NewRateLimiter(rate.Limit(0.5), 30)

	// Concurrent execution semaphore
	executeSemaphore := mw.ExecuteSemaphore(cfg.MaxConcurrent)

	api.RegisterRoutes(router, store, interpreter, executeLimiter.Middleware(), executeSemaphore)

	port := cfg.Port
	log.Printf("Starting hong-ik backend on :%s (env=%s)", port, cfg.Env)
	log.Printf("CORS origins: %v", cfg.CORSOrigins)
	log.Printf("Max concurrent executions: %d", cfg.MaxConcurrent)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
