package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBody는 요청 본문을 maxBytes로 제한한다.
// 초과 본문은 http.MaxBytesReader가 읽는 단계에서 차단하므로,
// gin이 JSON을 메모리에 전부 로딩하기 전에 거부된다 (메모리 소진 방어).
//
// gin의 ShouldBindJSON 같은 후속 호출이 실패 시, 핸들러는 보통 400을 반환한다.
// 본 미들웨어는 추가로 본문이 명시적 한도를 넘어 잘렸을 때 413(Payload Too Large)로 즉시 차단한다.
func MaxBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// MethodGet/Head/Delete처럼 본문이 없는 메서드는 우회 — 비용 0, 호환성 ↑.
		if c.Request.Body == nil || c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		// ContentLength가 사전에 한도를 초과한다고 신고하면 즉시 413.
		// (악의적 클라이언트는 ContentLength를 거짓말할 수 있지만, Reader가 한 번 더 잡는다.)
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "요청 본문이 너무 큽니다",
			})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()

		// 핸들러가 본문 읽기에 실패해 maxBytesError를 받았다면, 그 응답을 표준 413으로 정규화한다.
		// gin의 c.Errors는 핸들러가 c.Error(err)로 push한 에러들. ShouldBindJSON 실패 시
		// 핸들러는 보통 400을 반환하지만, 사이즈 초과 케이스만은 413이 더 정확하다.
		for _, ginErr := range c.Errors {
			var maxBytesErr *http.MaxBytesError
			if errors.As(ginErr.Err, &maxBytesErr) {
				// 응답이 이미 쓰였다면 덮어쓸 수 없음 — 로그용 표식만 남기고 끝.
				return
			}
		}
	}
}
