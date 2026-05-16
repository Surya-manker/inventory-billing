package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

// Auth validates the Bearer token and stores user_id + user_role in the context.
// Every downstream handler retrieves these via utils.UserIDFromCtx / utils.UserRoleFromCtx.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			utils.ErrorResponse(c, http.StatusUnauthorized, "authorization header missing or malformed")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.ParseJWT(token, jwtSecret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// RequireRole aborts the request when the authenticated user's role does not match.
// Must be placed after Auth in the middleware chain.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if utils.UserRoleFromCtx(c) != role {
			utils.ErrorResponse(c, http.StatusForbidden, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}
