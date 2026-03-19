package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/handler"
	"github.com/user/note-app/pkg/utils"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			handler.RespondUnauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			handler.RespondUnauthorized(c)
			c.Abort()
			return
		}

		userID, err := utils.ParseJWT(parts[1], jwtSecret)
		if err != nil {
			handler.RespondUnauthorized(c)
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
