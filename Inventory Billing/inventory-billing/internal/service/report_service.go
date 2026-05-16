package service

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/inventory-billing/internal/repository"
)

type ReportService interface {
	Revenue(ctx context.Context, from, to time.Time) ([]repository.RevenuePoint, error)
	TopProducts(ctx context.Context, from, to time.Time, limit int) ([]repository.TopProduct, error)
	InvoiceSummary(ctx context.Context, from, to time.Time) (map[string]any, error)
	InventoryValue(ctx context.Context) (float64, error)
	AgingReport(ctx context.Context) ([]repository.AgingRow, error)
}

type reportService struct {
	repo repository.ReportRepository
}

func NewReportService(repo repository.ReportRepository) ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) Revenue(ctx context.Context, from, to time.Time) ([]repository.RevenuePoint, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}
	return s.repo.Revenue(ctx, from, to)
}

func (s *reportService) TopProducts(ctx context.Context, from, to time.Time, limit int) ([]repository.TopProduct, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.repo.TopProducts(ctx, from, to, limit)
}

func (s *reportService) InvoiceSummary(ctx context.Context, from, to time.Time) (map[string]any, error) {
	if err := validateDateRange(from, to); err != nil {
		return nil, err
	}
	return s.repo.InvoiceSummary(ctx, from, to)
}

func (s *reportService) InventoryValue(ctx context.Context) (float64, error) {
	return s.repo.InventoryValue(ctx)
}

func (s *reportService) AgingReport(ctx context.Context) ([]repository.AgingRow, error) {
	return s.repo.AgingReport(ctx)
}

func validateDateRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() {
		return errors.New("from and to dates are required")
	}
	if to.Before(from) {
		return errors.New("to must be after from")
	}
	if to.Sub(from).Hours() > 24*366 {
		return errors.New("date range cannot exceed one year")
	}
	return nil
}
