package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"gorm.io/gorm"
)

type RecordPaymentInput struct {
	InvoiceID   uuid.UUID
	RecordedBy  uuid.UUID
	Amount      float64
	Method      domain.PaymentMethod
	ReferenceNo string
	Notes       string
	PaidAt      time.Time
}

type PaymentService interface {
	Record(ctx context.Context, in RecordPaymentInput) (*domain.Payment, error)
	ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.Payment, *domain.PaymentSummary, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type paymentService struct {
	repo          repository.PaymentRepository
	invoiceRepo   repository.InvoiceRepository
	accountingSvc AccountingService
}

func NewPaymentService(payRepo repository.PaymentRepository, invRepo repository.InvoiceRepository, accountingSvc AccountingService) PaymentService {
	return &paymentService{repo: payRepo, invoiceRepo: invRepo, accountingSvc: accountingSvc}
}

func (s *paymentService) Record(ctx context.Context, in RecordPaymentInput) (*domain.Payment, error) {
	if in.Amount <= 0 {
		return nil, errors.New("payment amount must be greater than zero")
	}
	if in.PaidAt.IsZero() {
		in.PaidAt = time.Now().UTC()
	}
	// Verify invoice exists before entering the transaction.
	if _, err := s.invoiceRepo.FindByID(ctx, in.InvoiceID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	p := &domain.Payment{
		InvoiceID:   in.InvoiceID,
		RecordedBy:  in.RecordedBy,
		Amount:      in.Amount,
		Method:      in.Method,
		ReferenceNo: in.ReferenceNo,
		Notes:       in.Notes,
		PaidAt:      in.PaidAt,
	}
	created, err := s.repo.CreateTx(ctx, p)
	if err != nil {
		return nil, err
	}

	// Post accounting entry after the payment transaction commits.
	_ = s.accountingSvc.PostPayment(ctx, created)

	return created, nil
}

func (s *paymentService) ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.Payment, *domain.PaymentSummary, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, err
	}

	payments, err := s.repo.ListByInvoice(ctx, invoiceID)
	if err != nil {
		return nil, nil, err
	}

	totalPaid, err := s.repo.SumByInvoice(ctx, invoiceID)
	if err != nil {
		return nil, nil, err
	}

	summary := &domain.PaymentSummary{
		TotalPaid:    totalPaid,
		TotalDue:     invoice.TotalPrice - totalPaid,
		Overpaid:     totalPaid > invoice.TotalPrice,
		FullySettled: totalPaid >= invoice.TotalPrice-0.01,
	}
	return payments, summary, nil
}

func (s *paymentService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}
