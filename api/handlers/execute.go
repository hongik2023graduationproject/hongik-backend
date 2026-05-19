package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 이전에 이 파일은 사용자 코드 실행 핸들러(/api/execute)를 담았으나,
// WASM-only 전환으로 코드 실행이 클라이언트 측으로 이전되어 백엔드에서 제거되었다.
// 남아 있는 핸들러는 health/readiness/language 메타정보 조회 — 데이터 도메인 핸들러는 snippets.go에 분리되어 있다.

// HealthCheck (liveness probe): 프로세스가 살아 있고 HTTP 핸들러가 응답할 수 있는지만 확인한다.
// 의존 자원(DB/캐시)이 죽어 있어도 200을 반환한다 — readiness와 의도적으로 구분.
// k8s 컨벤션에 맞춰 /healthz alias도 동일하게 매핑된다.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ReadyCheck (readiness probe): 트래픽을 받을 준비가 되었는지 확인한다.
// 백킹 스토어 ping에 실패하면 503을 반환해 로드밸런서가 이 인스턴스를 빼도록 한다.
// 캐시 장애는 degraded mode로 계속 서빙하므로 readiness 실패 사유가 아니다.
func (h *Handler) ReadyCheck(c *gin.Context) {
	if err := h.store.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "store_unreachable",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (h *Handler) GetBuiltins(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"builtins": []gin.H{
			{"name": "출력", "description": "값을 콘솔에 출력합니다", "usage": "출력(값)"},
			{"name": "길이", "description": "배열 또는 문자열의 길이를 반환합니다", "usage": "길이(배열)"},
			{"name": "추가", "description": "배열에 요소를 추가합니다", "usage": "추가(배열, 값)"},
		},
	})
}

func (h *Handler) GetSyntax(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"types":      []string{"정수", "실수", "문자", "불", "배열"},
		"keywords":   []string{"만약", "라면", "아니면", "함수", "리턴"},
		"operators":  []string{"+", "-", "*", "/", "==", "!=", "<", ">", "<=", ">="},
		"delimiters": []string{"(", ")", "[", "]", "{", "}", ":", ","},
	})
}
