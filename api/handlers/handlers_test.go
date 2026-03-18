package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hongik-backend/config"
	"hongik-backend/service"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter() *gin.Engine {
	store := service.NewStore()
	cfg := &config.Config{
		InterpreterPath: "nonexistent-binary",
		ExecuteTimeout:  5,
	}
	interpreter := service.NewInterpreterService(cfg)
	h := New(store, interpreter)

	router := gin.New()
	router.GET("/health", h.HealthCheck)
	router.GET("/api/snippets", h.ListSnippets)
	router.GET("/api/snippets/:id", h.GetSnippet)
	router.POST("/api/snippets", h.CreateSnippet)
	router.POST("/api/share", h.CreateShare)
	router.GET("/api/share/:token", h.GetShare)
	router.GET("/api/language/builtins", h.GetBuiltins)
	router.GET("/api/language/syntax", h.GetSyntax)
	router.POST("/api/execute", h.Execute)
	return router
}

func TestHealthCheck(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestListSnippets(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string][]map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp["snippets"]) != 5 {
		t.Errorf("expected 5 seed snippets, got %d", len(resp["snippets"]))
	}
}

func TestCreateAndGetSnippet(t *testing.T) {
	router := setupRouter()

	// Create
	body, _ := json.Marshal(map[string]string{
		"title": "테스트",
		"code":  "출력(\"테스트\")",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Get
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/snippets/"+id, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("get: expected 200, got %d", w2.Code)
	}
}

func TestGetSnippetNotFound(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateSnippetBadRequest(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestShareCreateAndGet(t *testing.T) {
	router := setupRouter()

	// Create share
	body, _ := json.Marshal(map[string]string{
		"code":  "출력(\"공유\")",
		"title": "공유 테스트",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/share", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("share create: expected 201, got %d", w.Code)
	}

	var shareResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	token := shareResp["token"]
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Get share
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/share/"+token, nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("share get: expected 200, got %d", w2.Code)
	}

	var shared map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &shared)
	if shared["code"] != "출력(\"공유\")" {
		t.Errorf("expected shared code, got %v", shared["code"])
	}
}

func TestGetShareNotFound(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/share/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetBuiltins(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/language/builtins", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	builtins, ok := resp["builtins"].([]interface{})
	if !ok || len(builtins) == 0 {
		t.Error("expected non-empty builtins array")
	}
}

func TestGetSyntax(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/language/syntax", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	for _, key := range []string{"types", "keywords", "operators", "delimiters"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("expected %q in syntax response", key)
		}
	}
}

func TestExecuteBadRequest(t *testing.T) {
	router := setupRouter()

	// Empty body
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/execute", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty code, got %d", w.Code)
	}
}

func TestExecuteCodeTooLarge(t *testing.T) {
	router := setupRouter()

	largeCode := make([]byte, 100001)
	for i := range largeCode {
		largeCode[i] = 'a'
	}
	body, _ := json.Marshal(map[string]string{
		"code": string(largeCode),
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for large code, got %d", w.Code)
	}
}

func TestExecuteInvalidTimeout(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"code":    "출력(1)",
		"timeout": 60,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid timeout, got %d", w.Code)
	}
}
