package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/pkg/utils"
)

// OptionalAuth parses JWT if present. If missing or invalid, continues without user_id.
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		userID, err := utils.ParseJWT(parts[1], jwtSecret)
		if err != nil {
			c.Next() // invalid token = anonymous
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// GetOptionalUserID returns user ID if authenticated, or uuid.Nil if anonymous.
func GetOptionalUserID(c *gin.Context) uuid.UUID {
	val, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	return val.(uuid.UUID)
}
