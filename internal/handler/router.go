package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
)

type Handlers struct {
	Auth      *AuthHandler
	Note      *NoteHandler
	Plan      *PlanHandler
	CheckIn   *CheckInHandler
	Upload    *UploadHandler
	Social    *SocialHandler
	Explore   *ExploreHandler
	Growth    *GrowthHandler
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

			plans := protected.Group("/plans")
			{
				plans.GET("", h.Plan.List)
				plans.POST("", h.Plan.Create)
				plans.GET("/:id", h.Plan.Get)
				plans.PUT("/:id", h.Plan.Update)
				plans.DELETE("/:id", h.Plan.Delete)
				plans.PUT("/:id/share", h.Plan.Share)
				plans.POST("/:id/join", h.Plan.Join)
				plans.GET("/:id/members", h.Plan.Members)

				// Check-in routes (nested under plans)
				plans.POST("/:id/checkins", h.CheckIn.CheckIn)
				plans.GET("/:id/checkins", h.CheckIn.ListByPlan)
			plans.GET("/:id/leaderboard", h.Plan.Leaderboard)
			}

			// Calendar (top-level under protected)
			protected.GET("/checkins/calendar", h.CheckIn.Calendar)

			upload := protected.Group("/upload")
			{
				upload.POST("/presign", h.Upload.Presign)
				upload.POST("/confirm", h.Upload.Confirm)
			}

			if h.Growth != nil {
			growth := protected.Group("/growth")
			{
				growth.GET("/reports", h.Growth.List)
				growth.POST("/generate", h.Growth.Generate)
			}
		}

		if h.Social != nil {
				social := protected.Group("/social")
				{
					social.POST("/:target_type/:id/like", h.Social.Like)
					social.DELETE("/:target_type/:id/like", h.Social.Unlike)
					social.GET("/:target_type/:id/comments", h.Social.GetComments)
					social.POST("/:target_type/:id/comments", h.Social.CreateComment)
					social.DELETE("/comments/:id", h.Social.DeleteComment)
					social.GET("/comments/:id/replies", h.Social.GetReplies)
				}
			}
		}

		// Public explore (optional auth for is_liked)
		if h.Explore != nil {
			explore := v1.Group("/explore", middleware.OptionalAuth(h.JWTSecret))
			{
				explore.GET("/notes", h.Explore.ListNotes)
				explore.GET("/plans", h.Explore.ListPlans)
			}
		}
	}

	return r
}
