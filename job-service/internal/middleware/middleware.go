package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func UserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader != "" {
			userID, err := strconv.ParseUint(userIDHeader, 10, 32)
			if err == nil {
				c.Set("user_id", uint(userID))
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found"})
		c.Abort()
	}
}
