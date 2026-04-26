package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"golang.org/x/time/rate"
)

func RateLimiter(rps float64, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}
