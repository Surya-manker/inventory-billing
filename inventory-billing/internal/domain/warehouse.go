package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Warehouse represents a physical storage location.
// Each product can have separate stock levels per warehouse via WarehouseStock.
type Warehouse struct {
	ID        uuid.UUID `gorm:"type:varchar(36);primaryKey"         json:"id"`
	Name      string    `gorm:"type:varchar(100);not null"          json:"name"`
	Code      string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"` // e.g. "WH-BLR-01"
	Address   string    `gorm:"type:text"                           json:"address"`
	IsDefault bool      `gorm:"not null;default:false"              json:"is_default"`
	IsActive  bool      `gorm:"not null;default:true"               json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (w *Warehouse) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// WarehouseStock holds the current stock level of one product in one warehouse.
// The composite unique index prevents duplicate (warehouse, product) pairs.
type WarehouseStock struct {
	ID          uuid.UUID `gorm:"type:varchar(36);primaryKey"                           json:"id"`
	WarehouseID uuid.UUID `gorm:"type:varchar(36);not null;uniqueIndex:ux_wh_prod"      json:"warehouse_id"`
	Warehouse   Warehouse `gorm:"foreignKey:WarehouseID"                                json:"warehouse,omitempty"`
	ProductID   uuid.UUID `gorm:"type:varchar(36);not null;uniqueIndex:ux_wh_prod"      json:"product_id"`
	Product     Product   `gorm:"foreignKey:ProductID"                                  json:"product,omitempty"`
	Stock       int       `gorm:"not null;default:0"                                    json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ws *WarehouseStock) BeforeCreate(tx *gorm.DB) error {
	if ws.ID == uuid.Nil {
		ws.ID = uuid.New()
	}
	return nil
}
