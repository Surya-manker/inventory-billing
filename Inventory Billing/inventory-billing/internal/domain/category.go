package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category supports one level of nesting (parent → children).
// Keeping it to one level avoids recursive CTE queries and covers 99 % of
// real-world catalogues (e.g. Electronics → Cables, Peripherals).
type Category struct {
	ID          uuid.UUID  `gorm:"type:varchar(36);primaryKey"                    json:"id"`
	Name        string     `gorm:"type:varchar(100);uniqueIndex;not null"         json:"name"`
	Description string     `gorm:"type:text"                                      json:"description"`
	ParentID    *uuid.UUID `gorm:"type:varchar(36);index"                         json:"parent_id,omitempty"`
	Parent      *Category  `gorm:"foreignKey:ParentID"                            json:"parent,omitempty"`
	IsActive    bool       `gorm:"not null;default:true"                          json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
