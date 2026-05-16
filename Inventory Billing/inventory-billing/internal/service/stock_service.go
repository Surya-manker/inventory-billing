package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
)

type StockService interface {
	// GetAlerts returns every product whose current stock is at or below its
	// individual low_stock_threshold. Callers can poll this endpoint or hook it
	// into a notification pipeline.
	GetAlerts(ctx context.Context) ([]domain.Product, error)

	// GetLogs returns a paginated, filtered view of the stock_logs table.
	GetLogs(ctx context.Context, filter domain.StockLogFilter, limit, offset int) ([]domain.StockLog, int64, error)

	// GetProductLogs returns the stock history for a single product.
	GetProductLogs(ctx context.Context, productID uuid.UUID, limit, offset int) ([]domain.StockLog, int64, error)
}

type stockService struct {
	productRepo  repository.ProductRepository
	stockLogRepo repository.StockLogRepository
}

func NewStockService(productRepo repository.ProductRepository, stockLogRepo repository.StockLogRepository) StockService {
	return &stockService{productRepo: productRepo, stockLogRepo: stockLogRepo}
}

func (s *stockService) GetAlerts(ctx context.Context) ([]domain.Product, error) {
	return s.productRepo.ListLowStock(ctx)
}

func (s *stockService) GetLogs(ctx context.Context, filter domain.StockLogFilter, limit, offset int) ([]domain.StockLog, int64, error) {
	return s.stockLogRepo.List(ctx, filter, limit, offset)
}

func (s *stockService) GetProductLogs(ctx context.Context, productID uuid.UUID, limit, offset int) ([]domain.StockLog, int64, error) {
	return s.stockLogRepo.ListByProduct(ctx, productID, limit, offset)
}
