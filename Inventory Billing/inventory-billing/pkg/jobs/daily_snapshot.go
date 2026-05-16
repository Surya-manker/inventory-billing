package jobs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/inventory-billing/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DailySnapshotJob runs once per day (shortly after midnight) and pre-computes
// yesterday's revenue stats into Redis.  The dashboard reads this cached value
// instead of running seven parallel DB queries on every page load.
//
// Cache key: "snapshot:daily:YYYY-MM-DD"
// TTL: 8 days (retains a week of snapshots for trend comparison)
type DailySnapshotJob struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
}

func NewDailySnapshotJob(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *DailySnapshotJob {
	return &DailySnapshotJob{db: db, rdb: rdb, log: log}
}

func (j *DailySnapshotJob) Name() string            { return "daily-snapshot" }
func (j *DailySnapshotJob) Schedule() time.Duration { return 24 * time.Hour }

type DailyStats struct {
	Date             string  `json:"date"`
	Revenue          float64 `json:"revenue"`
	InvoiceCount     int     `json:"invoice_count"`
	NewCustomers     int     `json:"new_customers"`
	LowStockProducts int     `json:"low_stock_products"`
}

func (j *DailySnapshotJob) Execute(ctx context.Context) error {
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24*time.Hour - time.Nanosecond)

	var revenue float64
	var invoiceCount int64
	j.db.WithContext(ctx).
		Model(&domain.Invoice{}).
		Where("status = ? AND issued_at BETWEEN ? AND ?", domain.InvoicePaid, start, end).
		Select("COALESCE(SUM(total_price),0)").Scan(&revenue)
	j.db.WithContext(ctx).
		Model(&domain.Invoice{}).
		Where("status = ? AND issued_at BETWEEN ? AND ?", domain.InvoicePaid, start, end).
		Count(&invoiceCount)

	var newCustomers int64
	j.db.WithContext(ctx).Model(&domain.Customer{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Count(&newCustomers)

	var lowStock int64
	j.db.WithContext(ctx).Model(&domain.Product{}).
		Where("stock <= low_stock_threshold").
		Count(&lowStock)

	stats := DailyStats{
		Date:             yesterday.Format(time.DateOnly),
		Revenue:          revenue,
		InvoiceCount:     int(invoiceCount),
		NewCustomers:     int(newCustomers),
		LowStockProducts: int(lowStock),
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	key := "snapshot:daily:" + stats.Date
	if err := j.rdb.Set(ctx, key, data, 8*24*time.Hour).Err(); err != nil {
		return err
	}

	j.log.Info("daily-snapshot: saved",
		zap.String("date", stats.Date),
		zap.Float64("revenue", stats.Revenue),
		zap.Int("invoices", stats.InvoiceCount),
	)
	return nil
}
