package handlers

import (
	"net/http"

	"hongik-backend/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListSnippets(c *gin.Context) {
	snippets := h.store.ListSnippets()
	c.JSON(http.StatusOK, gin.H{"snippets": snippets})
}

func (h *Handler) GetSnippet(c *gin.Context) {
	id := c.Param("id")

	snippet, ok := h.store.GetSnippet(id)
	if !ok {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "스니펫을 찾을 수 없습니다"})
		return
	}

	c.JSON(http.StatusOK, snippet)
}

func (h *Handler) CreateSnippet(c *gin.Context) {
	var req model.CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "제목과 코드를 입력해주세요"})
		return
	}

	if len(req.Code) > 100000 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "코드가 100,000바이트 제한을 초과합니다"})
		return
	}

	userID, _ := c.Get("userID")
	userIDStr, _ := userID.(string)

	snippet := h.store.CreateSnippet(req, userIDStr)
	c.JSON(http.StatusCreated, snippet)
}

func (h *Handler) UpdateSnippet(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "제목과 코드를 입력해주세요"})
		return
	}

	if len(req.Code) > 100000 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "코드가 100,000바이트 제한을 초과합니다"})
		return
	}

	userID, _ := c.Get("userID")
	userIDStr, _ := userID.(string)

	snippet, found, owned := h.store.UpdateSnippet(id, req, userIDStr)
	if !found {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "스니펫을 찾을 수 없습니다"})
		return
	}
	if !owned {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "이 스니펫을 수정할 권한이 없습니다"})
		return
	}

	c.JSON(http.StatusOK, snippet)
}

func (h *Handler) DeleteSnippet(c *gin.Context) {
	id := c.Param("id")

	userID, _ := c.Get("userID")
	userIDStr, _ := userID.(string)

	found, owned := h.store.DeleteSnippet(id, userIDStr)
	if !found {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "스니펫을 찾을 수 없습니다"})
		return
	}
	if !owned {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "이 스니펫을 삭제할 권한이 없습니다"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "스니펫이 삭제되었습니다"})
}
