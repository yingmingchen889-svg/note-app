package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
)

type Handlers struct {
	Auth      *AuthHandler
	Note      *NoteHandler
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

		protected := v1.Group("", middleware.Auth(h.JWTSecret))
		{
			notes := protected.Group("/notes")
			{
				notes.GET("", h.Note.List)
				notes.POST("", h.Note.Create)
				notes.GET("/:id", h.Note.Get)
				notes.PUT("/:id", h.Note.Update)
				notes.DELETE("/:id", h.Note.Delete)
				notes.PUT("/:id/share", h.Note.Share)
			}
		}
	}

	return r
}
