package api

import (
	"hongik-backend/api/handlers"
	"hongik-backend/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, store *service.Store, interpreter *service.InterpreterService, executeMiddlewares ...gin.HandlerFunc) {
	h := handlers.New(store, interpreter)

	router.GET("/health", h.HealthCheck)

	api := router.Group("/api")
	{
		// Execute endpoint with dedicated rate limit + semaphore
		executeHandlers := make([]gin.HandlerFunc, 0, len(executeMiddlewares)+1)
		executeHandlers = append(executeHandlers, executeMiddlewares...)
		executeHandlers = append(executeHandlers, h.Execute)
		api.POST("/execute", executeHandlers...)

		api.GET("/snippets", h.ListSnippets)
		api.GET("/snippets/:id", h.GetSnippet)
		api.POST("/snippets", h.CreateSnippet)

		api.POST("/share", h.CreateShare)
		api.GET("/share/:token", h.GetShare)

		api.GET("/language/builtins", h.GetBuiltins)
		api.GET("/language/syntax", h.GetSyntax)
	}
}
