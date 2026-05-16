package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceStatus string

const (
	InvoicePending  InvoiceStatus = "pending"
	InvoicePartial  InvoiceStatus = "partial"  // at least one payment received, not fully settled
	InvoiceOverdue  InvoiceStatus = "overdue"  // past due_at with outstanding balance; set by background job
	InvoicePaid     InvoiceStatus = "paid"
	InvoiceCanceled InvoiceStatus = "canceled"
)

// ValidTransitions governs manual status changes via the admin API.
// InvoicePartial is set automatically by the payment system.
// InvoiceOverdue is set automatically by the overdue background job.
var ValidTransitions = map[InvoiceStatus][]InvoiceStatus{
	InvoicePending:  {InvoicePaid, InvoiceCanceled},
	InvoicePartial:  {InvoicePaid, InvoiceCanceled},
	InvoiceOverdue:  {InvoicePaid, InvoiceCanceled},
	InvoicePaid:     {},
	InvoiceCanceled: {},
}

type Invoice struct {
	ID            uuid.UUID     `gorm:"type:varchar(36);primaryKey"                  json:"id"`
	InvoiceNumber string        `gorm:"type:varchar(50);uniqueIndex;not null"        json:"invoice_number"`
	CustomerID    uint          `gorm:"not null;index"                               json:"customer_id"`
	Customer      Customer      `gorm:"foreignKey:CustomerID"                        json:"customer,omitempty"`
	CreatedBy     uuid.UUID     `gorm:"type:varchar(36);not null"                    json:"created_by"`
	Creator       User          `gorm:"foreignKey:CreatedBy"                         json:"creator,omitempty"`
	Items         []InvoiceItem `gorm:"foreignKey:InvoiceID"                         json:"items"`

	SubTotal      float64 `gorm:"type:decimal(12,2);not null;default:0" json:"sub_total"`
	Discount      float64 `gorm:"type:decimal(12,2);not null;default:0" json:"discount"`
	TaxableAmount float64 `gorm:"type:decimal(12,2);not null;default:0" json:"taxable_amount"`
	TaxAmount     float64 `gorm:"type:decimal(12,2);not null;default:0" json:"tax_amount"`
	TotalPrice    float64 `gorm:"type:decimal(12,2);not null"           json:"total_price"`
	PaidAmount    float64 `gorm:"type:decimal(12,2);not null;default:0" json:"paid_amount"`

	IntraState bool    `gorm:"type:tinyint(1);not null;default:1"   json:"intra_state"`
	CGSTAmount float64 `gorm:"type:decimal(12,2);not null;default:0" json:"cgst_amount"`
	SGSTAmount float64 `gorm:"type:decimal(12,2);not null;default:0" json:"sgst_amount"`
	IGSTAmount float64 `gorm:"type:decimal(12,2);not null;default:0" json:"igst_amount"`

	Status InvoiceStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Notes  string        `gorm:"type:text"                                        json:"notes"`

	// Composite index covers the two most common list-query filter combos:
	//   - status + issued_at  (e.g. all pending invoices sorted by date)
	//   - customer_id already has its own index above
	IssuedAt  time.Time  `gorm:"index"                json:"issued_at"`
	DueAt     time.Time  `json:"due_at"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (inv *Invoice) BeforeCreate(tx *gorm.DB) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	return nil
}

// OutstandingBalance returns the amount still owed on this invoice.
func (inv *Invoice) OutstandingBalance() float64 {
	bal := inv.TotalPrice - inv.PaidAmount
	if bal < 0 {
		return 0
	}
	return bal
}

type InvoiceItem struct {
	ID        uuid.UUID `gorm:"type:varchar(36);primaryKey"               json:"id"`
	InvoiceID uuid.UUID `gorm:"type:varchar(36);not null;index"           json:"invoice_id"`
	ProductID uuid.UUID `gorm:"type:varchar(36);not null"                 json:"product_id"`
	Product   Product   `gorm:"foreignKey:ProductID"                      json:"product,omitempty"`

	Quantity  int     `gorm:"not null"                               json:"quantity"`
	UnitPrice float64 `gorm:"type:decimal(12,2);not null"            json:"unit_price"`
	CostPrice float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"cost_price"` // snapshotted at sale time
	SubTotal  float64 `gorm:"type:decimal(12,2);not null"            json:"sub_total"`
	TaxRate   float64 `gorm:"type:decimal(5,2);not null;default:0"   json:"tax_rate"`
	TaxAmount float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"tax_amount"`
	Total     float64 `gorm:"type:decimal(12,2);not null"            json:"total"`

	CreatedAt time.Time `json:"created_at"`
}

func (item *InvoiceItem) BeforeCreate(tx *gorm.DB) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	return nil
}

// InvoiceFilter lives in domain to avoid circular imports.
type InvoiceFilter struct {
	CustomerID *uint
	Status     string
	FromDate   *time.Time
	ToDate     *time.Time
	SortBy     string // issued_at | total_price | due_at | created_at
	SortDir    string // asc | desc
}
