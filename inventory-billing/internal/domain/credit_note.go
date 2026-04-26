package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreditNoteStatus models the lifecycle of a credit note.
// issued  → the CN is created and valid, customer balance is reduced
// applied → the CN has been used to offset a future invoice (handled externally for now)
// voided  → the CN is cancelled, stock that was restored is re-deducted
type CreditNoteStatus string

const (
	CNIssued  CreditNoteStatus = "issued"
	CNApplied CreditNoteStatus = "applied"
	CNVoided  CreditNoteStatus = "voided"
)

// CreditNote is a formal document acknowledging a return or credit against a
// previously paid invoice.  It always references the original invoice so
// auditors can trace the full transaction chain.
type CreditNote struct {
	ID       uuid.UUID `gorm:"type:varchar(36);primaryKey"              json:"id"`
	CNNumber string    `gorm:"type:varchar(50);uniqueIndex;not null"    json:"cn_number"`

	InvoiceID uuid.UUID `gorm:"type:varchar(36);not null;index"          json:"invoice_id"`
	Invoice   Invoice   `gorm:"foreignKey:InvoiceID"                     json:"invoice,omitempty"`

	CustomerID uint     `gorm:"not null;index"                           json:"customer_id"`
	Customer   Customer `gorm:"foreignKey:CustomerID"                    json:"customer,omitempty"`

	IssuedBy uuid.UUID `gorm:"type:varchar(36);not null"                json:"issued_by"`
	Issuer   User      `gorm:"foreignKey:IssuedBy"                      json:"issuer,omitempty"`

	Items []CreditNoteItem `gorm:"foreignKey:CreditNoteID"                json:"items"`

	SubTotal    float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"sub_total"`
	TaxAmount   float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"tax_amount"`
	TotalAmount float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"total_amount"`

	Status CreditNoteStatus `gorm:"type:varchar(20);not null;default:'issued';index" json:"status"`
	Reason string           `gorm:"type:varchar(500);not null"                       json:"reason"`
	Notes  string           `gorm:"type:text"                                        json:"notes"`

	IssuedAt time.Time  `gorm:"index"  json:"issued_at"`
	VoidedAt *time.Time `json:"voided_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (cn *CreditNote) BeforeCreate(tx *gorm.DB) error {
	if cn.ID == uuid.Nil {
		cn.ID = uuid.New()
	}
	return nil
}

type CreditNoteItem struct {
	ID           uuid.UUID `gorm:"type:varchar(36);primaryKey"               json:"id"`
	CreditNoteID uuid.UUID `gorm:"type:varchar(36);not null;index"           json:"credit_note_id"`
	ProductID    uuid.UUID `gorm:"type:varchar(36);not null"                 json:"product_id"`
	Product      Product   `gorm:"foreignKey:ProductID"                      json:"product,omitempty"`

	Quantity  int     `gorm:"not null"                              json:"quantity"`
	UnitPrice float64 `gorm:"type:decimal(12,2);not null"           json:"unit_price"` // price at time of original invoice
	SubTotal  float64 `gorm:"type:decimal(12,2);not null"           json:"sub_total"`
	TaxRate   float64 `gorm:"type:decimal(5,2);not null;default:0"  json:"tax_rate"`
	TaxAmount float64 `gorm:"type:decimal(12,2);not null;default:0" json:"tax_amount"`
	Total     float64 `gorm:"type:decimal(12,2);not null"           json:"total"`

	CreatedAt time.Time `json:"created_at"`
}

func (i *CreditNoteItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// CNCounter provides gap-free credit note numbers via auto-increment.
type CNCounter struct {
	ID uint `gorm:"primaryKey;autoIncrement"`
}
