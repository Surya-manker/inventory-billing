package jobs

import (
	"context"
	"time"

	"github.com/yourusername/inventory-billing/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LowStockAlertJob scans the product catalogue every hour and logs any product
// whose stock is at or below its low_stock_threshold.
//
// Production upgrade: replace the zap.Warn call with an email/SMS/webhook
// notification via a dedicated NotificationService.
type LowStockAlertJob struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewLowStockAlertJob(db *gorm.DB, log *zap.Logger) *LowStockAlertJob {
	return &LowStockAlertJob{db: db, log: log}
}

func (j *LowStockAlertJob) Name() string             { return "low-stock-alert" }
func (j *LowStockAlertJob) Schedule() time.Duration  { return time.Hour }

func (j *LowStockAlertJob) Execute(ctx context.Context) error {
	var products []domain.Product
	err := j.db.WithContext(ctx).
		Where("stock <= low_stock_threshold").
		Order("stock ASC").
		Find(&products).Error
	if err != nil {
		return err
	}

	if len(products) == 0 {
		j.log.Info("low-stock-alert: all products are adequately stocked")
		return nil
	}

	for _, p := range products {
		j.log.Warn("LOW STOCK",
			zap.String("product_id", p.ID.String()),
			zap.String("name", p.Name),
			zap.String("sku", p.SKU),
			zap.Int("stock", p.Stock),
			zap.Int("threshold", p.LowStockThreshold),
			zap.Int("deficit", p.LowStockThreshold-p.Stock),
		)
	}
	j.log.Info("low-stock-alert: scan complete",
		zap.Int("alert_count", len(products)))
	return nil
}
