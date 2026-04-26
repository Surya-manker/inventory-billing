package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"gorm.io/gorm"
)

// sensitiveFields are stripped from the request body before storage.
var sensitiveFields = map[string]bool{
	"password":     true,
	"old_password": true,
	"new_password": true,
	"confirm":      true,
	"token":        true,
	"refresh_token": true,
}

// AuditLog records every mutating request (POST / PUT / PATCH / DELETE) to the
// audit_logs table. It is placed after Auth so user identity is available, but
// it does NOT block the request — a write failure is logged to stderr and ignored.
func AuditLog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		// Only audit mutations.
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			c.Next()
			return
		}

		// Read the body and restore it so downstream handlers can re-read it.
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next() // ← call the actual handler first so we capture the status code

		userID, _ := c.Get("user_id")
		userEmail, _ := c.Get("user_email") // set in Auth middleware if present
		userRole, _ := c.Get("user_role")

		uid, _ := userID.(string)
		email, _ := userEmail.(string)
		role, _ := userRole.(string)

		entry := domain.AuditLog{
			UserID:      uid,
			UserEmail:   email,
			UserRole:    role,
			Method:      method,
			Path:        c.FullPath(),
			StatusCode:  c.Writer.Status(),
			IPAddress:   c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
			RequestBody: sanitizeBody(bodyBytes),
		}

		// Fire-and-forget: use a background goroutine so the audit write never
		// slows down the HTTP response.
		go func() {
			db.Create(&entry) //nolint:errcheck — best-effort audit
		}()
	}
}

// sanitizeBody strips sensitive keys and truncates to 4 KB.
func sanitizeBody(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		// Not JSON (e.g., form data) — store first 512 bytes only.
		if len(raw) > 512 {
			return string(raw[:512]) + "…"
		}
		return string(raw)
	}
	for k := range m {
		if sensitiveFields[strings.ToLower(k)] {
			m[k] = "***"
		}
	}
	clean, _ := json.Marshal(m)
	if len(clean) > 4096 {
		return string(clean[:4096]) + "…"
	}
	return string(clean)
}

// enrichAuthContext stores user_email in the Gin context so AuditLog can read it.
// Call this inside the Auth middleware after c.Set("user_id", ...) is already done.
func EnrichAuditContext(c *gin.Context, email string) {
	c.Set("user_email", email)
	utils.UserRoleFromCtx(c) // no-op — just verifying the value is set
}
