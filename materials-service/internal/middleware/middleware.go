package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserIDMiddleware extracts user_id from context
// In a real scenario, this would extract from JWT token
// For now, we assume user_id comes from header X-User-ID (for testing)
// or from JWT claims set by auth middleware
func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get from header (for testing)
		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader != "" {
			userID, err := strconv.ParseUint(userIDHeader, 10, 32)
			if err == nil {
				c.Set("user_id", uint(userID))
				c.Next()
				return
			}
		}

		// Try to get from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found"})
			c.Abort()
			return
		}

		// Ensure it's uint
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id type"})
			c.Abort()
			return
		}

		c.Set("user_id", userIDUint)
		c.Next()
	}
}

// GetUserID extracts user_id from context
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userIDUint, ok := userID.(uint)
	return userIDUint, ok
}
