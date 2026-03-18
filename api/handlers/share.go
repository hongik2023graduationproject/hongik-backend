package handlers

import (
	"net/http"

	"hongik-backend/model"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateShare(c *gin.Context) {
	var req model.ShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "코드를 입력해주세요"})
		return
	}

	shared := h.store.CreateShare(req)
	c.JSON(http.StatusCreated, model.ShareResponse{Token: shared.Token})
}

func (h *Handler) GetShare(c *gin.Context) {
	token := c.Param("token")

	shared, ok := h.store.GetShare(token)
	if !ok {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "공유 코드를 찾을 수 없습니다"})
		return
	}

	c.JSON(http.StatusOK, shared)
}
