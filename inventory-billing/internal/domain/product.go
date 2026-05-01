package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID                uuid.UUID  `gorm:"type:varchar(36);primaryKey"                json:"id"`
	Name              string     `gorm:"type:varchar(100);not null"                 json:"name"`
	Description       string     `gorm:"type:text"                                  json:"description"`
	SKU               string     `gorm:"type:varchar(50);uniqueIndex;not null"      json:"sku"`
	CategoryID        *uuid.UUID `gorm:"type:varchar(36);index"                     json:"category_id,omitempty"`
	Category          *Category  `gorm:"foreignKey:CategoryID"                      json:"category,omitempty"`
	Price             float64    `gorm:"type:decimal(12,2);not null"                json:"price"`
	CostPrice         float64    `gorm:"type:decimal(12,2);not null;default:0"      json:"cost_price"`
	Stock             int        `gorm:"not null;default:0"                         json:"stock"`
	TaxRate           float64    `gorm:"type:decimal(5,2);not null;default:0"       json:"tax_rate"`
	LowStockThreshold int        `gorm:"not null;default:10"                        json:"low_stock_threshold"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// IsLowStock returns true when current stock is at or below the threshold.
func (p *Product) IsLowStock() bool {
	return p.Stock <= p.LowStockThreshold
}

// Margin returns the gross margin as a percentage of selling price.
func (p *Product) Margin() float64 {
	if p.Price == 0 {
		return 0
	}
	return (p.Price - p.CostPrice) / p.Price * 100
}

// ProductFilter is shared by repository and service to avoid circular imports.
type ProductFilter struct {
	Search      string
	CategoryID  *uuid.UUID // nil means "all categories"
	MinPrice    float64
	MaxPrice    float64
	LowStockMax *int   // filter: stock <= this value
	SortBy      string // name | price | stock | created_at
	SortDir     string // asc | desc
}

// StockAdjustInput carries everything the repository needs for an atomic
// stock movement + audit log in one transaction.
type StockAdjustInput struct {
	ProductID      uuid.UUID
	UserID         uuid.UUID
	InvoiceID      *uuid.UUID
	ChangeType     StockChangeType
	QuantityChange int // signed: positive = in, negative = out
	Note           string
}
