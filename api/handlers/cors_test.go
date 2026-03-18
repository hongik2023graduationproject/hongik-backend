package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hongik-backend/config"
	mw "hongik-backend/middleware"
	"hongik-backend/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func setupFullRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		InterpreterPath: "nonexistent",
		ExecuteTimeout:  5,
		CORSOrigins:     origins,
		MaxConcurrent:   5,
	}

	store := service.NewStore()
	interpreter := service.NewInterpreterService(cfg)
	h := New(store, interpreter)

	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	apiLimiter := mw.NewRateLimiter(rate.Limit(100), 100)
	router.Use(apiLimiter.Middleware())

	router.GET("/health", h.HealthCheck)
	api := router.Group("/api")
	{
		api.GET("/snippets", h.ListSnippets)
		api.POST("/execute", h.Execute)
	}

	return router
}

func TestCORSAllowedOrigin(t *testing.T) {
	router := setupFullRouter([]string{"http://localhost:3000", "http://localhost:5173"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("expected CORS origin http://localhost:3000, got %q", origin)
	}

	creds := w.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials: true, got %q", creds)
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	router := setupFullRouter([]string{"http://localhost:3000"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets", nil)
	req.Header.Set("Origin", "http://evil.com")
	router.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected empty CORS origin for disallowed origin, got %q", origin)
	}
}

func TestCORSPreflight(t *testing.T) {
	router := setupFullRouter([]string{"http://localhost:3000"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/execute", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("preflight: expected 200 or 204, got %d", w.Code)
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("preflight: expected origin http://localhost:3000, got %q", origin)
	}

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("preflight: expected Access-Control-Allow-Methods header")
	}
}

func TestCORSCustomOriginFromEnv(t *testing.T) {
	router := setupFullRouter([]string{"https://hongik.example.com"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "https://hongik.example.com")
	router.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://hongik.example.com" {
		t.Errorf("expected custom origin, got %q", origin)
	}
}
