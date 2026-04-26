package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleStaff Role = "staff"
)

type User struct {
	ID        uuid.UUID `gorm:"type:varchar(36);primaryKey"              json:"id"`
	Name      string    `gorm:"type:varchar(100);not null"               json:"name"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"   json:"email"`
	Password  string    `gorm:"type:varchar(255);not null"               json:"-"`
	Role      Role      `gorm:"type:varchar(20);default:'staff'"         json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
