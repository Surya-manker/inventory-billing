package domain

import "time"

// AuditLog is an append-only record of every mutating API call.
// It is written by middleware — handlers never touch it directly.
type AuditLog struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Who
	UserID    string `gorm:"type:varchar(36);index" json:"user_id"`   // empty for unauthenticated
	UserEmail string `gorm:"type:varchar(255)"      json:"user_email"`
	UserRole  string `gorm:"type:varchar(20)"       json:"user_role"`

	// What
	Method     string `gorm:"type:varchar(10);not null"  json:"method"`      // POST, PUT, PATCH, DELETE
	Path       string `gorm:"type:varchar(500);not null" json:"path"`
	StatusCode int    `gorm:"not null"                   json:"status_code"`

	// Context
	IPAddress string `gorm:"type:varchar(45)"   json:"ip_address"` // supports IPv6
	UserAgent string `gorm:"type:varchar(500)"  json:"user_agent"`

	// Payload — request body stored as-is (password fields stripped in middleware)
	RequestBody string `gorm:"type:text" json:"request_body"`

	CreatedAt time.Time `gorm:"index" json:"created_at"`
}
