package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var params model.RegisterParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), params)
	if err != nil {
		if isDuplicateKeyError(err) {
			RespondConflict(c, "email already registered")
			return
		}
		RespondInternalError(c)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var params model.LoginParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
			return
		}
		RespondInternalError(c)
		return
	}

	RespondOK(c, gin.H{
		"user":  user,
		"token": token,
	})
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key"))
}
