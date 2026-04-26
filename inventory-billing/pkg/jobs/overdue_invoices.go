package jobs

import (
	"context"
	"time"

	"github.com/yourusername/inventory-billing/internal/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OverdueInvoiceJob runs once per day and logs all pending invoices whose
// due date has passed.  In a full product, this drives payment-reminder emails
// to customers.
type OverdueInvoiceJob struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewOverdueInvoiceJob(db *gorm.DB, log *zap.Logger) *OverdueInvoiceJob {
	return &OverdueInvoiceJob{db: db, log: log}
}

func (j *OverdueInvoiceJob) Name() string            { return "overdue-invoice-alert" }
func (j *OverdueInvoiceJob) Schedule() time.Duration { return 24 * time.Hour }

func (j *OverdueInvoiceJob) Execute(ctx context.Context) error {
	type result struct {
		domain.Invoice
		CustomerName  string
		CustomerEmail string
	}

	var rows []result
	err := j.db.WithContext(ctx).
		Table("invoices i").
		Select("i.*, c.name AS customer_name, c.email AS customer_email").
		Joins("JOIN customers c ON c.id = i.customer_id").
		Where("i.status = ? AND i.due_at < ?", domain.InvoicePending, time.Now().UTC()).
		Order("i.due_at ASC").
		Scan(&rows).Error
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		j.log.Info("overdue-invoice-alert: no overdue invoices")
		return nil
	}

	for _, r := range rows {
		daysOverdue := int(time.Since(r.DueAt).Hours() / 24)
		j.log.Warn("OVERDUE INVOICE",
			zap.String("invoice_number", r.InvoiceNumber),
			zap.String("customer", r.CustomerName),
			zap.String("customer_email", r.CustomerEmail),
			zap.Float64("amount", r.TotalPrice),
			zap.Int("days_overdue", daysOverdue),
		)
	}
	j.log.Info("overdue-invoice-alert: scan complete",
		zap.Int("overdue_count", len(rows)))
	return nil
}
