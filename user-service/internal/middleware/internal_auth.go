package middleware

import (
	"net/http"

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
