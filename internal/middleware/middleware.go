package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	logger "github.com/ruziba3vich/prodonik_lgger"
	limiter "github.com/ruziba3vich/prodonik_rl"
)

type MidWare struct {
	logger  *logger.Logger
	limiter *limiter.TokenBucketLimiter
}

func NewMidWare(logger *logger.Logger, limiter *limiter.TokenBucketLimiter) *MidWare {
	return &MidWare{
		logger:  logger,
		limiter: limiter,
	}
}

// RateLimitMiddleware creates a Gin middleware that enforces rate limiting using TokenBucketLimiter
func (m *MidWare) Middleware() func(gin.HandlerFunc) gin.HandlerFunc {
	return func(handler gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			clientIP := c.ClientIP()
			if clientIP == "" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to determine client IP"})
				c.Abort()
				return
			}

			allowed, err := m.limiter.AllowRequest(c.Request.Context(), clientIP)
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
}
