package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Vendor ────────────────────────────────────────────────────────────────────

type Vendor struct {
	ID            uuid.UUID `gorm:"type:varchar(36);primaryKey"              json:"id"`
	Name          string    `gorm:"type:varchar(100);not null"               json:"name"`
	Email         string    `gorm:"type:varchar(255);uniqueIndex;not null"   json:"email"`
	Phone         string    `gorm:"type:varchar(20)"                         json:"phone"`
	Address       string    `gorm:"type:text"                                json:"address"`
	Company       string    `gorm:"type:varchar(100)"                        json:"company"`
	GSTIN         string    `gorm:"type:varchar(20)"                         json:"gstin"`
	ContactPerson string    `gorm:"type:varchar(100)"                        json:"contact_person"`
	// PaymentTermsDays: how many days after PO receipt the invoice is due
	PaymentTermsDays int       `gorm:"not null;default:30"                      json:"payment_terms_days"`
	IsActive         bool      `gorm:"not null;default:true"                    json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (v *Vendor) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// ── Purchase Order ────────────────────────────────────────────────────────────

type POStatus string

const (
	PODraft     POStatus = "draft"
	POConfirmed POStatus = "confirmed"
	// POReceived means all line items have been received into stock.
	POReceived POStatus = "received"
	POCanceled POStatus = "canceled"
)

// ValidPOTransitions mirrors the invoice state machine pattern.
var ValidPOTransitions = map[POStatus][]POStatus{
	PODraft:     {POConfirmed, POCanceled},
	POConfirmed: {POReceived, POCanceled},
	POReceived:  {},
	POCanceled:  {},
}

type PurchaseOrder struct {
	ID       uuid.UUID `gorm:"type:varchar(36);primaryKey"              json:"id"`
	PONumber string    `gorm:"type:varchar(50);uniqueIndex;not null"    json:"po_number"`

	VendorID uuid.UUID `gorm:"type:varchar(36);not null;index"          json:"vendor_id"`
	Vendor   Vendor    `gorm:"foreignKey:VendorID"                      json:"vendor,omitempty"`

	CreatedBy uuid.UUID `gorm:"type:varchar(36);not null"               json:"created_by"`
	Creator   User      `gorm:"foreignKey:CreatedBy"                     json:"creator,omitempty"`

	Items []POItem `gorm:"foreignKey:POID" json:"items"`

	SubTotal    float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"sub_total"`
	TaxAmount   float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"tax_amount"`
	TotalAmount float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"total_amount"`

	Status     POStatus   `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	Notes      string     `gorm:"type:text"                                       json:"notes"`
	ExpectedAt time.Time  `json:"expected_at"`
	ReceivedAt *time.Time `json:"received_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (po *PurchaseOrder) BeforeCreate(tx *gorm.DB) error {
	if po.ID == uuid.Nil {
		po.ID = uuid.New()
	}
	return nil
}

type POItem struct {
	ID        uuid.UUID `gorm:"type:varchar(36);primaryKey"               json:"id"`
	POID      uuid.UUID `gorm:"type:varchar(36);not null;index"           json:"po_id"`
	ProductID uuid.UUID `gorm:"type:varchar(36);not null"                 json:"product_id"`
	Product   Product   `gorm:"foreignKey:ProductID"                      json:"product,omitempty"`

	Quantity    int     `gorm:"not null"                               json:"quantity"`
	UnitCost    float64 `gorm:"type:decimal(12,2);not null"            json:"unit_cost"`
	TaxRate     float64 `gorm:"type:decimal(5,2);not null;default:0"   json:"tax_rate"`
	TaxAmount   float64 `gorm:"type:decimal(12,2);not null;default:0"  json:"tax_amount"`
	TotalCost   float64 `gorm:"type:decimal(12,2);not null"            json:"total_cost"`
	ReceivedQty int     `gorm:"not null;default:0"                     json:"received_qty"`

	CreatedAt time.Time `json:"created_at"`
}

func (p *POItem) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// POCounter gives gap-free purchase order numbers (same pattern as InvoiceCounter).
type POCounter struct {
	ID uint `gorm:"primaryKey;autoIncrement"`
}

// VendorFilter is shared between repository and service.
type VendorFilter struct {
	Search   string
	IsActive *bool
}
