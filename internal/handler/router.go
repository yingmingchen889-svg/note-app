package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
)

type Handlers struct {
	Auth      *AuthHandler
	JWTSecret string
}

func SetupRouter(h *Handlers) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
		}
	}

	_ = v1.Group("/").Use(middleware.Auth(h.JWTSecret))

	return r
}
