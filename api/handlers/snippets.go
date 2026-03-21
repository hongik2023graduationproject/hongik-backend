package handlers

import (
	"net/http"
	"strconv"

	"hongik-backend/model"

	"github.com/gin-gonic/gin"
)

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	limit := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// getUserID extracts the authenticated user's ID from context.
// Returns an empty string if not authenticated.
func getUserID(c *gin.Context) string {
	userID, _ := c.Get("userID")
	id, _ := userID.(string)
	return id
}

// validateCodeSize checks that the code does not exceed the size limit.
// Writes a 400 response and returns false if the limit is exceeded.
func validateCodeSize(c *gin.Context, code string) bool {
	if len(code) > 100000 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "코드가 100,000바이트 제한을 초과합니다"})
		return false
	}
	return true
}

func (h *Handler) ListSnippets(c *gin.Context) {
	page, limit := parsePagination(c)
	snippets, total := h.store.ListSnippets(page, limit)
	c.JSON(http.StatusOK, model.SnippetListResponse{
		Snippets: snippets,
		Total:    total,
		Page:     page,
		Limit:    limit,
	})
}

func (h *Handler) SearchSnippets(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "검색어를 입력해주세요"})
		return
	}
	page, limit := parsePagination(c)
	snippets, total := h.store.SearchSnippets(query, page, limit)
	c.JSON(http.StatusOK, model.SnippetListResponse{
		Snippets: snippets,
		Total:    total,
		Page:     page,
		Limit:    limit,
	})
}

func (h *Handler) ForkSnippet(c *gin.Context) {
	id := c.Param("id")

	forked, ok := h.store.ForkSnippet(id, getUserID(c))
	if !ok {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "스니펫을 찾을 수 없습니다"})
		return
	}

	c.JSON(http.StatusCreated, forked)
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

	if !validateCodeSize(c, req.Code) {
		return
	}

	snippet := h.store.CreateSnippet(req, getUserID(c))
	c.JSON(http.StatusCreated, snippet)
}

func (h *Handler) UpdateSnippet(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "제목과 코드를 입력해주세요"})
		return
	}

	if !validateCodeSize(c, req.Code) {
		return
	}

	snippet, found, owned := h.store.UpdateSnippet(id, req, getUserID(c))
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

	found, owned := h.store.DeleteSnippet(id, getUserID(c))
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
