package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	// CreateTx records the payment and, if the invoice is now fully settled,
	// atomically marks it as "paid" — all inside a single transaction.
	CreateTx(ctx context.Context, payment *domain.Payment) (*domain.Payment, error)

	FindByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.Payment, error)

	// SumByInvoice returns the total amount paid so far against an invoice.
	SumByInvoice(ctx context.Context, invoiceID uuid.UUID) (float64, error)

	Delete(ctx context.Context, id uuid.UUID) error
}

type paymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateTx(ctx context.Context, payment *domain.Payment) (*domain.Payment, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Re-fetch invoice inside the transaction with a row lock.
		var invoice domain.Invoice
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&invoice, "id = ?", payment.InvoiceID).Error; err != nil {
			return domain.ErrNotFound
		}
		if invoice.Status == domain.InvoiceCanceled {
			return domain.ErrCannotPayCanceledInvoice
		}
		if invoice.Status == domain.InvoicePaid {
			return domain.ErrInvoiceAlreadyPaid
		}

		// Sum existing payments inside the transaction for an accurate balance.
		var already float64
		tx.Model(&domain.Payment{}).
			Where("invoice_id = ?", payment.InvoiceID).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&already)

		if already+payment.Amount > invoice.TotalPrice+0.01 { // 1-paisa tolerance
			return domain.ErrPaymentExceedsBalance
		}

		if err := tx.Create(payment).Error; err != nil {
			return err
		}

		// Auto-close the invoice when fully settled.
		if already+payment.Amount >= invoice.TotalPrice-0.01 {
			if err := tx.Model(&invoice).Updates(map[string]any{
				"status":  domain.InvoicePaid,
				"paid_at": payment.PaidAt,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, payment.ID)
}

func (r *paymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	var p domain.Payment
	err := r.db.WithContext(ctx).
		Preload("Recorder").
		First(&p, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.Payment, error) {
	var payments []domain.Payment
	err := r.db.WithContext(ctx).
		Preload("Recorder").
		Where("invoice_id = ?", invoiceID).
		Order("paid_at ASC").
		Find(&payments).Error
	return payments, err
}

func (r *paymentRepository) SumByInvoice(ctx context.Context, invoiceID uuid.UUID) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}

func (r *paymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Payment{}, "id = ?", id).Error
}
