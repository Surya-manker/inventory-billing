package domain

import "time"

// InvoiceCounter is a single-purpose append-only table that provides
// gap-safe, auto-incrementing invoice sequence numbers in MySQL.
// Each INSERT gives an atomically unique ID — no sequences needed.
type InvoiceCounter struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
}
