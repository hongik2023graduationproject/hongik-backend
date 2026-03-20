package handlers

import (
	"net/http"

	"hongik-backend/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Execute(c *gin.Context) {
	var req model.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "코드를 입력해주세요"})
		return
	}

	if len(req.Code) > 100000 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "코드가 100,000바이트 제한을 초과합니다"})
		return
	}

	if req.Timeout != 0 && (req.Timeout < 1 || req.Timeout > 30) {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "타임아웃은 1초~30초 범위여야 합니다"})
		return
	}

	resp := h.interpreter.Execute(req)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
