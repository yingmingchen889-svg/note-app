package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/pkg/utils"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondUnauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondUnauthorized(c)
			c.Abort()
			return
		}

		userID, err := utils.ParseJWT(parts[1], jwtSecret)
		if err != nil {
			respondUnauthorized(c)
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func respondUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":    "UNAUTHORIZED",
		"message": "authentication required",
	})
}
