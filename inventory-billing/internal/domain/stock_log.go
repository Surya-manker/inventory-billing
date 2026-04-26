package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StockChangeType string

const (
	StockSale       StockChangeType = "sale"
	StockPurchase   StockChangeType = "purchase"
	StockAdjustment StockChangeType = "adjustment"
	StockReturn     StockChangeType = "return"
)

type StockLog struct {
	ID             uuid.UUID       `gorm:"type:varchar(36);primaryKey"                    json:"id"`
	ProductID      uuid.UUID       `gorm:"type:varchar(36);not null;index:idx_sl_product"  json:"product_id"`
	Product        Product         `gorm:"foreignKey:ProductID"                           json:"product,omitempty"`
	UserID         uuid.UUID       `gorm:"type:varchar(36);not null"                      json:"user_id"`
	User           User            `gorm:"foreignKey:UserID"                              json:"user,omitempty"`
	InvoiceID      *uuid.UUID      `gorm:"type:varchar(36)"                               json:"invoice_id,omitempty"`
	ChangeType     StockChangeType `gorm:"type:varchar(30);not null"                      json:"change_type"`
	QuantityBefore int             `gorm:"not null"                                       json:"quantity_before"`
	QuantityChange int             `gorm:"not null"                                       json:"quantity_change"`
	QuantityAfter  int             `gorm:"not null"                                       json:"quantity_after"`
	Note           string          `gorm:"type:text"                                      json:"note"`
	// index on created_at supports date-range queries in the audit log
	CreatedAt      time.Time       `gorm:"index"                                          json:"created_at"`
}

func (s *StockLog) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// StockLogFilter is shared by repository and service.
type StockLogFilter struct {
	ProductID  *uuid.UUID
	ChangeType string
	FromDate   *time.Time
	ToDate     *time.Time
}
