package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hongik-backend/config"
	mw "hongik-backend/middleware"
	"hongik-backend/model"
	"hongik-backend/service"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testConfig() *config.Config {
	return &config.Config{
		InterpreterPath: "nonexistent-binary",
		ExecuteTimeout:  5,
		MaxOutputBytes:  1048576,
		JWTSecret:       "test-secret-key",
	}
}

func setupRouter() *gin.Engine {
	store := service.NewStore()
	cfg := testConfig()
	interpreter := service.NewInterpreterService(cfg)
	h := New(store, interpreter, nil)
	authHandler := NewAuthHandler(store, cfg)
	authRequired := mw.AuthRequired(cfg.JWTSecret)

	router := gin.New()
	router.GET("/health", h.HealthCheck)
	router.GET("/api/snippets", h.ListSnippets)
	router.GET("/api/snippets/search", h.SearchSnippets)
	router.GET("/api/snippets/:id", h.GetSnippet)
	router.POST("/api/snippets", authRequired, h.CreateSnippet)
	router.PUT("/api/snippets/:id", authRequired, h.UpdateSnippet)
	router.DELETE("/api/snippets/:id", authRequired, h.DeleteSnippet)
	router.POST("/api/snippets/:id/fork", authRequired, h.ForkSnippet)
	router.POST("/api/share", h.CreateShare)
	router.GET("/api/share/:token", h.GetShare)
	router.GET("/api/language/builtins", h.GetBuiltins)
	router.GET("/api/language/syntax", h.GetSyntax)
	router.POST("/api/execute", h.Execute)
	router.POST("/api/auth/register", authHandler.Register)
	router.POST("/api/auth/login", authHandler.Login)
	return router
}

// Helper to register a user and get a JWT token
func registerAndGetToken(t *testing.T, router *gin.Engine, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(model.RegisterRequest{
		Username: username,
		Password: password,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Token
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
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
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

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Snippets) != 5 {
		t.Errorf("expected 5 seed snippets, got %d", len(resp.Snippets))
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", resp.Limit)
	}
}

func TestCreateAndGetSnippet(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "testuser", "testpass")

	// Create
	body, _ := json.Marshal(map[string]string{
		"title": "테스트",
		"code":  "출력(\"테스트\")",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
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

func TestCreateSnippetRequiresAuth(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(map[string]string{
		"title": "테스트",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}

func TestCreateSnippetBadRequest(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "testuser2", "testpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateSnippet(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "updateuser", "testpass")

	// Create a snippet first
	createBody, _ := json.Marshal(map[string]string{
		"title": "원본",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Update it
	updateBody, _ := json.Marshal(map[string]string{
		"title": "수정됨",
		"code":  "출력(2)",
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PUT", "/api/snippets/"+id, bytes.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("update: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var updated map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &updated)
	if updated["title"] != "수정됨" {
		t.Errorf("expected title '수정됨', got %v", updated["title"])
	}
}

func TestUpdateSnippetNotFound(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "updateuser2", "testpass")

	body, _ := json.Marshal(map[string]string{
		"title": "없음",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/snippets/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateSnippetRequiresAuth(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(map[string]string{
		"title": "수정",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/snippets/someid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdateSnippetCodeTooLarge(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "updateuser3", "testpass")

	// Create a snippet first
	createBody, _ := json.Marshal(map[string]string{
		"title": "원본",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Update with too-large code
	largeCode := make([]byte, 100001)
	for i := range largeCode {
		largeCode[i] = 'a'
	}
	updateBody, _ := json.Marshal(map[string]string{
		"title": "큰코드",
		"code":  string(largeCode),
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PUT", "/api/snippets/"+id, bytes.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for large code, got %d", w2.Code)
	}
}

func TestDeleteSnippet(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "deleteuser", "testpass")

	// Create a snippet first
	createBody, _ := json.Marshal(map[string]string{
		"title": "삭제할것",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Delete it
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("DELETE", "/api/snippets/"+id, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("delete: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify it's gone
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/api/snippets/"+id, nil)
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w3.Code)
	}
}

func TestDeleteSnippetNotFound(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "deleteuser2", "testpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/snippets/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteSnippetRequiresAuth(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/snippets/someid", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
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
	_ = json.Unmarshal(w.Body.Bytes(), &shareResp)
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
	_ = json.Unmarshal(w2.Body.Bytes(), &shared)
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
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
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
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
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

// Auth tests

func TestRegister(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(model.RegisterRequest{
		Username: "newuser",
		Password: "password123",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Username != "newuser" {
		t.Errorf("expected username 'newuser', got %s", resp.User.Username)
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(model.RegisterRequest{
		Username: "dupuser",
		Password: "password123",
	})

	// First register
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", w.Code)
	}

	// Duplicate register
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("duplicate register: expected 409, got %d", w2.Code)
	}
}

func TestRegisterBadRequest(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegisterShortUsername(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(model.RegisterRequest{
		Username: "a",
		Password: "password123",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short username, got %d", w.Code)
	}
}

func TestRegisterShortPassword(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(model.RegisterRequest{
		Username: "validuser",
		Password: "ab",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", w.Code)
	}
}

func TestLogin(t *testing.T) {
	router := setupRouter()

	// Register first
	registerAndGetToken(t, router, "loginuser", "password123")

	// Login
	body, _ := json.Marshal(model.LoginRequest{
		Username: "loginuser",
		Password: "password123",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("login: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	router := setupRouter()

	registerAndGetToken(t, router, "loginuser2", "correctpass")

	body, _ := json.Marshal(model.LoginRequest{
		Username: "loginuser2",
		Password: "wrongpass",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(model.LoginRequest{
		Username: "nosuchuser",
		Password: "password",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthInvalidToken(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(map[string]string{
		"title": "테스트",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalidtoken")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthBadFormat(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(map[string]string{
		"title": "테스트",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "NotBearer token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdateSnippetForbidden(t *testing.T) {
	router := setupRouter()

	// User A creates a snippet
	tokenA := registerAndGetToken(t, router, "userA", "testpass")
	createBody, _ := json.Marshal(map[string]string{
		"title": "A의 스니펫",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// User B tries to update it
	tokenB := registerAndGetToken(t, router, "userB", "testpass")
	updateBody, _ := json.Marshal(map[string]string{
		"title": "B가수정",
		"code":  "출력(2)",
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PUT", "/api/snippets/"+id, bytes.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tokenB)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

func TestDeleteSnippetForbidden(t *testing.T) {
	router := setupRouter()

	// User A creates a snippet
	tokenA := registerAndGetToken(t, router, "userC", "testpass")
	createBody, _ := json.Marshal(map[string]string{
		"title": "C의 스니펫",
		"code":  "출력(1)",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// User D tries to delete it
	tokenD := registerAndGetToken(t, router, "userD", "testpass")
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("DELETE", "/api/snippets/"+id, nil)
	req2.Header.Set("Authorization", "Bearer "+tokenD)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

// Pagination tests

func TestListSnippetsPagination(t *testing.T) {
	router := setupRouter()

	// Request page 1 with limit 2
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets?page=1&limit=2", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Snippets) != 2 {
		t.Errorf("expected 2 snippets, got %d", len(resp.Snippets))
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.Limit != 2 {
		t.Errorf("expected limit 2, got %d", resp.Limit)
	}

	// Request page 3 with limit 2 (should get 1 item)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/snippets?page=3&limit=2", nil)
	router.ServeHTTP(w2, req2)

	var resp2 model.SnippetListResponse
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2.Snippets) != 1 {
		t.Errorf("expected 1 snippet on page 3, got %d", len(resp2.Snippets))
	}
}

func TestListSnippetsLimitCap(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets?limit=200", nil)
	router.ServeHTTP(w, req)

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Limit != 100 {
		t.Errorf("expected limit capped to 100, got %d", resp.Limit)
	}
}

func TestListSnippetsPageBeyondRange(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets?page=999", nil)
	router.ServeHTTP(w, req)

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Snippets) != 0 {
		t.Errorf("expected 0 snippets for out-of-range page, got %d", len(resp.Snippets))
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
}

// Search tests

func TestSearchSnippets(t *testing.T) {
	router := setupRouter()

	// Search for "배열" which matches a seed snippet
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets/search?q=배열", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total == 0 {
		t.Error("expected at least one result for '배열'")
	}
	for _, sn := range resp.Snippets {
		if !containsIgnoreCase(sn.Title, "배열") && !containsIgnoreCase(sn.Description, "배열") {
			t.Errorf("snippet %q does not match search query", sn.Title)
		}
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if strings.EqualFold(s[i:i+len(substr)], substr) {
					return true
				}
			}
			return false
		}())
}

func TestSearchSnippetsNoQuery(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets/search", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearchSnippetsNoResults(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets/search?q=존재하지않는검색어", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("expected 0 results, got %d", resp.Total)
	}
	if len(resp.Snippets) != 0 {
		t.Errorf("expected empty snippets, got %d", len(resp.Snippets))
	}
}

func TestSearchSnippetsPagination(t *testing.T) {
	router := setupRouter()

	// Search for "예제" which should match multiple seed snippets
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/snippets/search?q=예제&limit=2&page=1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.SnippetListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Snippets) > 2 {
		t.Errorf("expected at most 2 snippets, got %d", len(resp.Snippets))
	}
	if resp.Limit != 2 {
		t.Errorf("expected limit 2, got %d", resp.Limit)
	}
}

// Fork tests

func TestForkSnippet(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "forkuser", "testpass")

	// Create a snippet
	createBody, _ := json.Marshal(map[string]string{
		"title":       "원본 코드",
		"code":        "출력(1)",
		"description": "원본 설명",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	originalID := created["id"].(string)

	// Fork it with a different user
	token2 := registerAndGetToken(t, router, "forkuser2", "testpass")
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/snippets/"+originalID+"/fork", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("fork: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var forked map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &forked)

	if forked["title"] != "원본 코드 (복사본)" {
		t.Errorf("expected title '원본 코드 (복사본)', got %v", forked["title"])
	}
	if forked["code"] != "출력(1)" {
		t.Errorf("expected same code, got %v", forked["code"])
	}
	if forked["id"] == originalID {
		t.Error("forked snippet should have a different ID")
	}
}

func TestForkSnippetNotFound(t *testing.T) {
	router := setupRouter()
	token := registerAndGetToken(t, router, "forkuser3", "testpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets/nonexistent/fork", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestForkSnippetRequiresAuth(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/snippets/someid/fork", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// Execute with input test

func TestExecuteWithInput(t *testing.T) {
	router := setupRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"code":  "출력(입력())",
		"input": "안녕하세요",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// We can't test actual execution (no interpreter binary in test),
	// but we verify the request is accepted (not 400)
	if w.Code == http.StatusBadRequest {
		t.Errorf("expected request with input to be accepted, got 400")
	}
}
