package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupMaxBodyRouter(limit int64) *gin.Engine {
	r := gin.New()
	r.Use(MaxBody(limit))
	r.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"len": len(body)})
	})
	return r
}

func TestMaxBodyAcceptsSmallPayload(t *testing.T) {
	r := setupMaxBodyRouter(1024)
	body := strings.Repeat("a", 512)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/echo", bytes.NewReader([]byte(body)))
	req.ContentLength = int64(len(body))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for body under limit, got %d", w.Code)
	}
}

func TestMaxBodyRejectsByContentLength(t *testing.T) {
	r := setupMaxBodyRouter(1024)
	// 본문은 작지만 ContentLength를 부풀려 신고 → 413 즉시 응답
	body := strings.Repeat("a", 100)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/echo", bytes.NewReader([]byte(body)))
	req.ContentLength = 5000 // 한도 초과 신고
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for oversized ContentLength, got %d", w.Code)
	}
}

func TestMaxBodyRejectsByActualSize(t *testing.T) {
	r := setupMaxBodyRouter(1024)
	// ContentLength를 미신고하지만 실제 본문은 한도 초과 → MaxBytesReader가 차단
	body := strings.Repeat("a", 5000)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/echo", bytes.NewReader([]byte(body)))
	req.ContentLength = -1 // chunked transfer encoding 시나리오
	r.ServeHTTP(w, req)
	// 핸들러가 ReadAll에서 에러를 만나 400을 반환할 수 있음 — 핵심은 200이 아니라는 것.
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for oversized actual body, got %d", w.Code)
	}
}

func TestMaxBodyAllowsBodylessRequest(t *testing.T) {
	r := setupMaxBodyRouter(1024)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/echo", http.NoBody)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for bodyless request, got %d", w.Code)
	}
}
