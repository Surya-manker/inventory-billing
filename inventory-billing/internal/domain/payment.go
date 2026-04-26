package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentMethod string

const (
	PaymentCash   PaymentMethod = "cash"
	PaymentNEFT   PaymentMethod = "neft"
	PaymentUPI    PaymentMethod = "upi"
	PaymentCheque PaymentMethod = "cheque"
	PaymentCard   PaymentMethod = "card"
)

// Payment records a single receipt against an invoice.
// An invoice may have multiple partial payments. When the sum of all payments
// reaches the invoice total, the invoice is automatically marked as "paid".
type Payment struct {
	ID uuid.UUID `gorm:"type:varchar(36);primaryKey" json:"id"`

	InvoiceID uuid.UUID `gorm:"type:varchar(36);not null;index" json:"invoice_id"`
	Invoice   Invoice   `gorm:"foreignKey:InvoiceID"            json:"invoice,omitempty"`

	RecordedBy uuid.UUID `gorm:"type:varchar(36);not null" json:"recorded_by"`
	Recorder   User      `gorm:"foreignKey:RecordedBy"     json:"recorder,omitempty"`

	Amount      float64       `gorm:"type:decimal(12,2);not null"      json:"amount"`
	Method      PaymentMethod `gorm:"type:varchar(20);not null"        json:"method"`
	ReferenceNo string        `gorm:"type:varchar(100)"                json:"reference_no"` // UTR / cheque no / UPI txn ID
	Notes       string        `gorm:"type:text"                        json:"notes"`
	PaidAt      time.Time     `gorm:"index"                            json:"paid_at"`
	CreatedAt   time.Time     `json:"created_at"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PaymentSummary is returned alongside invoice details to show running totals.
type PaymentSummary struct {
	TotalPaid    float64 `json:"total_paid"`
	TotalDue     float64 `json:"total_due"`
	Overpaid     bool    `json:"overpaid"`
	FullySettled bool    `json:"fully_settled"`
}
