package repository

import (
	"context"
	"time"

	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

// RevenuePoint is one data point in a time-series revenue chart.
type RevenuePoint struct {
	Date    string  `json:"date"`    // "2025-04" for monthly, "2025-04-26" for daily
	Revenue float64 `json:"revenue"` // sum of total_price for paid invoices
	Count   int     `json:"count"`   // number of invoices
}

// TopProduct is one row in the best-sellers report.
type TopProduct struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	SKU          string  `json:"sku"`
	TotalQty     int     `json:"total_qty"`
	TotalRevenue float64 `json:"total_revenue"`
}

// AgingRow is one customer row in the receivables aging report.
// Buckets are defined by how many days past the due date each invoice is.
type AgingRow struct {
	CustomerID       uint    `json:"customer_id"`
	CustomerName     string  `json:"customer_name"`
	CustomerEmail    string  `json:"customer_email"`
	InvoiceCount     int     `json:"invoice_count"`
	TotalOutstanding float64 `json:"total_outstanding"`
	NotDue           float64 `json:"not_due"`       // due_at in the future
	Days1To30        float64 `json:"days_1_30"`
	Days31To60       float64 `json:"days_31_60"`
	Days61To90       float64 `json:"days_61_90"`
	DaysOver90       float64 `json:"days_over_90"`
}

// ReportRepository runs read-only aggregation queries used by the analytics endpoints.
// All queries hit paid invoices only so cancelled/pending records don't skew the numbers.
type ReportRepository interface {
	// Revenue returns daily revenue totals between from and to (inclusive).
	Revenue(ctx context.Context, from, to time.Time) ([]RevenuePoint, error)

	// TopProducts returns the N best-selling products by quantity in the period.
	TopProducts(ctx context.Context, from, to time.Time, limit int) ([]TopProduct, error)

	// InvoiceSummary returns aggregate counts and revenue split by status.
	InvoiceSummary(ctx context.Context, from, to time.Time) (map[string]any, error)

	// InventoryValue returns the current total stock value (price × stock for all products).
	InventoryValue(ctx context.Context) (float64, error)

	// AgingReport returns outstanding receivables bucketed by days overdue.
	// Only pending invoices are included; paid and canceled are excluded.
	AgingReport(ctx context.Context) ([]AgingRow, error)
}

type reportRepository struct{ db *gorm.DB }

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) Revenue(ctx context.Context, from, to time.Time) ([]RevenuePoint, error) {
	var rows []RevenuePoint
	err := r.db.WithContext(ctx).
		Model(&domain.Invoice{}).
		Select("DATE(issued_at) AS date, SUM(total_price) AS revenue, COUNT(*) AS count").
		Where("status = ? AND issued_at BETWEEN ? AND ?", domain.InvoicePaid, from, to).
		Group("DATE(issued_at)").
		Order("date ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *reportRepository) TopProducts(ctx context.Context, from, to time.Time, limit int) ([]TopProduct, error) {
	var rows []TopProduct
	err := r.db.WithContext(ctx).
		Table("invoice_items ii").
		Select(`
			ii.product_id,
			p.name  AS product_name,
			p.sku   AS sku,
			SUM(ii.quantity)  AS total_qty,
			SUM(ii.total)     AS total_revenue
		`).
		Joins("JOIN invoices inv ON inv.id = ii.invoice_id AND inv.status = ? AND inv.issued_at BETWEEN ? AND ?",
			domain.InvoicePaid, from, to).
		Joins("JOIN products p ON p.id = ii.product_id").
		Group("ii.product_id, p.name, p.sku").
		Order("total_qty DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *reportRepository) InvoiceSummary(ctx context.Context, from, to time.Time) (map[string]any, error) {
	type row struct {
		Status  string  `gorm:"column:status"`
		Count   int     `gorm:"column:count"`
		Revenue float64 `gorm:"column:revenue"`
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&domain.Invoice{}).
		Select("status, COUNT(*) AS count, COALESCE(SUM(total_price),0) AS revenue").
		Where("issued_at BETWEEN ? AND ?", from, to).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	summary := map[string]any{
		"from": from.Format(time.DateOnly),
		"to":   to.Format(time.DateOnly),
	}
	for _, r := range rows {
		summary[r.Status] = map[string]any{"count": r.Count, "revenue": r.Revenue}
	}
	return summary, nil
}

func (r *reportRepository) InventoryValue(ctx context.Context) (float64, error) {
	var value float64
	err := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Select("COALESCE(SUM(price * stock), 0)").
		Scan(&value).Error
	return value, err
}

// AgingReport uses conditional aggregation in a single SQL pass — far more
// efficient than per-bucket queries.  MySQL DATEDIFF(NOW(), due_at) is positive
// when the due date is in the past (overdue) and negative when it is in the future.
func (r *reportRepository) AgingReport(ctx context.Context) ([]AgingRow, error) {
	var rows []AgingRow
	err := r.db.WithContext(ctx).
		Table("invoices i").
		Select(`
			i.customer_id,
			c.name   AS customer_name,
			c.email  AS customer_email,
			COUNT(*)                                                                        AS invoice_count,
			COALESCE(SUM(i.total_price), 0)                                                AS total_outstanding,
			COALESCE(SUM(CASE WHEN DATEDIFF(NOW(), i.due_at) <= 0  THEN i.total_price ELSE 0 END), 0) AS not_due,
			COALESCE(SUM(CASE WHEN DATEDIFF(NOW(), i.due_at) BETWEEN  1 AND  30 THEN i.total_price ELSE 0 END), 0) AS days_1_to_30,
			COALESCE(SUM(CASE WHEN DATEDIFF(NOW(), i.due_at) BETWEEN 31 AND  60 THEN i.total_price ELSE 0 END), 0) AS days_31_to_60,
			COALESCE(SUM(CASE WHEN DATEDIFF(NOW(), i.due_at) BETWEEN 61 AND  90 THEN i.total_price ELSE 0 END), 0) AS days_61_to_90,
			COALESCE(SUM(CASE WHEN DATEDIFF(NOW(), i.due_at)        >  90       THEN i.total_price ELSE 0 END), 0) AS days_over_90
		`).
		Joins("JOIN customers c ON c.id = i.customer_id").
		Where("i.status = ?", domain.InvoicePending).
		Group("i.customer_id, c.name, c.email").
		Having("total_outstanding > 0").
		Order("total_outstanding DESC").
		Scan(&rows).Error
	return rows, err
}
