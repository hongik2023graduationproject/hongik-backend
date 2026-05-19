package api

import (
	"hongik-backend/api/handlers"
	"hongik-backend/config"
	mw "hongik-backend/middleware"
	"hongik-backend/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, store service.Store, interpreter *service.InterpreterService, cache *service.Cache, cfg *config.Config, executeMiddlewares ...gin.HandlerFunc) {
	h := handlers.New(store, interpreter, cache)
	authHandler := handlers.NewAuthHandler(store, cfg)
	authRequired := mw.AuthRequired(cfg.JWTSecret)

	// Liveness: 프로세스가 살아 있는지 (의존성 무관).
	// /health는 기존 호환성, /healthz는 k8s 표준.
	router.GET("/health", h.HealthCheck)
	router.GET("/healthz", h.HealthCheck)

	// Readiness: 트래픽 받을 준비가 됐는지 (DB 등 의존성 확인).
	router.GET("/readyz", h.ReadyCheck)

	api := router.Group("/api")
	{
		// Execute endpoint with dedicated rate limit + semaphore
		executeHandlers := make([]gin.HandlerFunc, 0, len(executeMiddlewares)+1)
		executeHandlers = append(executeHandlers, executeMiddlewares...)
		executeHandlers = append(executeHandlers, h.Execute)
		api.POST("/execute", executeHandlers...)

		// Auth endpoints (public)
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)

		// Snippet endpoints - GET is public, mutations require auth
		api.GET("/snippets", h.ListSnippets)
		api.GET("/snippets/search", h.SearchSnippets)
		api.GET("/snippets/:id", h.GetSnippet)
		api.POST("/snippets", authRequired, h.CreateSnippet)
		api.PUT("/snippets/:id", authRequired, h.UpdateSnippet)
		api.DELETE("/snippets/:id", authRequired, h.DeleteSnippet)
		api.POST("/snippets/:id/fork", authRequired, h.ForkSnippet)

		api.POST("/share", h.CreateShare)
		api.GET("/share/:token", h.GetShare)

		api.GET("/language/builtins", h.GetBuiltins)
		api.GET("/language/syntax", h.GetSyntax)
	}
}
