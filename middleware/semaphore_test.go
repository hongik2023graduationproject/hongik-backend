package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSemaphoreAllowsConcurrent(t *testing.T) {
	router := gin.New()
	router.Use(ExecuteSemaphore(2))
	router.POST("/execute", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	var wg sync.WaitGroup
	results := make([]int, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/execute", nil)
			router.ServeHTTP(w, req)
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()

	for i, code := range results {
		if code != http.StatusOK {
			t.Errorf("concurrent request %d: expected 200, got %d", i, code)
		}
	}
}

func TestSemaphoreRejectsOverLimit(t *testing.T) {
	router := gin.New()
	router.Use(ExecuteSemaphore(1)) // Only 1 concurrent
	router.POST("/execute", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	var wg sync.WaitGroup
	results := make([]int, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/execute", nil)
			router.ServeHTTP(w, req)
			results[idx] = w.Code
		}(i)
	}

	wg.Wait()

	okCount := 0
	rejectedCount := 0
	for _, code := range results {
		switch code {
		case http.StatusOK:
			okCount++
		case http.StatusServiceUnavailable:
			rejectedCount++
		default:
			t.Errorf("unexpected status code: %d", code)
		}
	}

	if okCount == 0 {
		t.Error("expected at least one request to succeed")
	}
	if rejectedCount == 0 {
		t.Error("expected at least one request to be rejected (503)")
	}
}

func TestSemaphoreReturnsErrorMessage(t *testing.T) {
	router := gin.New()
	router.Use(ExecuteSemaphore(0)) // Zero capacity = all rejected
	router.POST("/execute", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/execute", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}
