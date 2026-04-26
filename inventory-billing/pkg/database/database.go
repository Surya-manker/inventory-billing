package database

import (
	"fmt"
	"time"

	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn
	if cfg.Database.Debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetimeMin) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate runs AutoMigrate for every model in dependency order so foreign key
// references are always satisfied before the referencing table is created.
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// independent tables
		&domain.User{},
		&domain.Category{},  // must come before Product
		&domain.Customer{},
		&domain.Product{},
		&domain.InvoiceCounter{},
		&domain.CNCounter{},
		&domain.Vendor{},
		&domain.POCounter{},
		&domain.Warehouse{},
		&domain.AuditLog{},

		// tables that reference the above
		&domain.Invoice{},
		&domain.InvoiceItem{},
		&domain.StockLog{},
		&domain.PurchaseOrder{},
		&domain.POItem{},
		&domain.Payment{},
		&domain.WarehouseStock{},
		&domain.CreditNote{},
		&domain.CreditNoteItem{},
	)
}
