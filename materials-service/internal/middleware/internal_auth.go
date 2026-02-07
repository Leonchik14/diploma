package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func InternalAuthMiddleware(internalAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Internal-API-Key")
		if apiKey == "" || apiKey != internalAPIKey {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid or missing internal API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func InternalOrUserAuthMiddleware(internalAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Internal-API-Key")
		if apiKey != "" && apiKey == internalAPIKey {
			c.Next()
			return
		}

		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader != "" {
			userID, err := strconv.ParseUint(userIDHeader, 10, 32)
			if err == nil {
				c.Set("user_id", uint(userID))
				c.Next()
				return
			}
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
