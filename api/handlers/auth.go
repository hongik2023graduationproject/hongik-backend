package handlers

import (
	"net/http"
	"time"

	"hongik-backend/config"
	"hongik-backend/model"
	"hongik-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	store  service.Store
	secret string
}

func NewAuthHandler(store service.Store, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		store:  store,
		secret: cfg.JWTSecret,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "사용자 이름과 비밀번호를 입력해주세요"})
		return
	}

	if len(req.Username) < 2 || len(req.Username) > 50 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "사용자 이름은 2~50자여야 합니다"})
		return
	}

	if len(req.Password) < 8 || len(req.Password) > 100 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "비밀번호는 8~100자여야 합니다"})
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Password)
	if err == service.ErrUsernameTaken {
		c.JSON(http.StatusConflict, model.ErrorResponse{Error: "사용자 이름이 이미 존재합니다"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "사용자 생성에 실패했습니다"})
		return
	}

	token, err := h.generateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "토큰 생성에 실패했습니다"})
		return
	}

	c.JSON(http.StatusCreated, model.AuthResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "사용자 이름과 비밀번호를 입력해주세요"})
		return
	}

	user, err := h.store.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "사용자 이름 또는 비밀번호가 올바르지 않습니다"})
		return
	}

	token, err := h.generateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "토큰 생성에 실패했습니다"})
		return
	}

	c.JSON(http.StatusOK, model.AuthResponse{Token: token, User: user})
}

func (h *AuthHandler) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.secret))
}
