package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

type StockLogRepository interface {
	List(ctx context.Context, filter domain.StockLogFilter, limit, offset int) ([]domain.StockLog, int64, error)
	ListByProduct(ctx context.Context, productID uuid.UUID, limit, offset int) ([]domain.StockLog, int64, error)
}

type stockLogRepository struct {
	db *gorm.DB
}

func NewStockLogRepository(db *gorm.DB) StockLogRepository {
	return &stockLogRepository{db: db}
}

func (r *stockLogRepository) List(ctx context.Context, filter domain.StockLogFilter, limit, offset int) ([]domain.StockLog, int64, error) {
	q := r.buildQuery(ctx, filter)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []domain.StockLog
	err := r.buildQuery(ctx, filter).
		Preload("Product").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&logs).Error
	return logs, total, err
}

func (r *stockLogRepository) ListByProduct(ctx context.Context, productID uuid.UUID, limit, offset int) ([]domain.StockLog, int64, error) {
	filter := domain.StockLogFilter{ProductID: &productID}
	return r.List(ctx, filter, limit, offset)
}

func (r *stockLogRepository) buildQuery(ctx context.Context, f domain.StockLogFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&domain.StockLog{})
	if f.ProductID != nil {
		q = q.Where("product_id = ?", f.ProductID.String())
	}
	if f.ChangeType != "" {
		q = q.Where("change_type = ?", f.ChangeType)
	}
	if f.FromDate != nil {
		q = q.Where("created_at >= ?", *f.FromDate)
	}
	if f.ToDate != nil {
		q = q.Where("created_at <= ?", *f.ToDate)
	}
	return q
}

// resolveChangeTypeOrder builds an ORDER BY with a direction guard.
func resolveChangeTypeOrder(dir string) string {
	if strings.EqualFold(dir, "asc") {
		return fmt.Sprintf("created_at %s", "ASC")
	}
	return "created_at DESC"
}
