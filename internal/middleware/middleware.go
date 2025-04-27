package middleware

import (
	"fmt"
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

// Middleware returns a standard Gin middleware handler for rate limiting.
func (m *MidWare) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP == "" {
			m.logger.Warn("RateLimit: Unable to determine client IP", map[string]any{"path": c.Request.URL.Path})

			c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden"})
			c.Abort()
			return
		}

		allowed, err := m.limiter.AllowRequest(c.Request.Context(), clientIP)
		if err != nil {
			m.logger.Error(fmt.Sprintf("RateLimit: Limiter error for IP %s: %v", clientIP, err), map[string]any{"ip": clientIP, "error": err.Error()})

			c.JSON(http.StatusInternalServerError, gin.H{fmt.Sprintf("RateLimit: Limiter error for IP %s: %v", clientIP, err): map[string]any{"ip": clientIP, "error": err.Error()}})
			c.Abort()
			return
		}

		if !allowed {
			m.logger.Info(fmt.Sprintf("RateLimit: Request rejected for IP %s", clientIP), map[string]any{"ip": clientIP, "path": c.Request.URL.Path})

			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
