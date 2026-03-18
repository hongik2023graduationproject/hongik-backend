package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExecuteSemaphore(maxConcurrent int) gin.HandlerFunc {
	sem := make(chan struct{}, maxConcurrent)

	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "서버가 현재 많은 요청을 처리 중입니다. 잠시 후 다시 시도해주세요.",
			})
		}
	}
}
