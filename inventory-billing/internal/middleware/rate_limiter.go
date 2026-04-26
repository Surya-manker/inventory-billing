package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter returns a per-IP token-bucket middleware.
// Stale entries (no request for cleanupAfter) are removed to bound memory usage.
func RateLimiter(rps float64, burst int) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		clients = make(map[string]*ipLimiter)
	)

	// Background cleanup: remove IPs that haven't sent a request in 3 minutes.
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, cl := range clients {
				if time.Since(cl.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		cl, ok := clients[ip]
		if !ok {
			cl = &ipLimiter{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		return cl.limiter
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !getLimiter(ip).Allow() {
			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}
