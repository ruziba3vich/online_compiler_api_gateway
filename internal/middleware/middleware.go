package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	limiter "github.com/ruziba3vich/prodonik_rl"
)

// RateLimitMiddleware creates a Gin middleware that enforces rate limiting using TokenBucketLimiter
func Middleware(limiter *limiter.TokenBucketLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to determine client IP"})
			c.Abort()
			return
		}

		allowed, err := limiter.AllowRequest(c.Request.Context(), clientIP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error: " + err.Error()})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
