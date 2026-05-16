package domain

import "time"

// Customer uses an auto-increment uint primary key — idiomatic for MySQL and
// simpler for human-readable billing references (e.g. "Customer #42").
type Customer struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"              json:"id"`
	Name      string    `gorm:"type:varchar(100);not null"            json:"name"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Phone     string    `gorm:"type:varchar(20)"                      json:"phone"`
	Address   string    `gorm:"type:text"                             json:"address"`
	Company   string    `gorm:"type:varchar(100)"                     json:"company"`
	GSTIN     string    `gorm:"type:varchar(20)"                      json:"gstin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
