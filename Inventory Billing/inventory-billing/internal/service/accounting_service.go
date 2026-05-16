package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
)

type AccountingService interface {
	// PostInvoice creates a double-entry journal for a newly issued invoice:
	//   DR Accounts Receivable   invoice.TotalPrice
	//     CR Sales Revenue           invoice.TaxableAmount
	//     CR CGST/SGST/IGST Payable  invoice.TaxAmount (split per GST type)
	PostInvoice(ctx context.Context, invoice *domain.Invoice, createdBy uuid.UUID) error

	// PostPayment creates a double-entry journal for a received payment:
	//   DR Cash & Bank          payment.Amount
	//     CR Accounts Receivable  payment.Amount
	PostPayment(ctx context.Context, payment *domain.Payment) error
}

type accountingService struct {
	repo repository.AccountingRepository
}

func NewAccountingService(repo repository.AccountingRepository) AccountingService {
	return &accountingService{repo: repo}
}

func (s *accountingService) PostInvoice(ctx context.Context, invoice *domain.Invoice, createdBy uuid.UUID) error {
	arAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctReceivable)
	if err != nil {
		return fmt.Errorf("accounting PostInvoice: AR account: %w", err)
	}
	revAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctSalesRevenue)
	if err != nil {
		return fmt.Errorf("accounting PostInvoice: revenue account: %w", err)
	}

	lines := []domain.LedgerEntry{
		{
			AccountID:   arAcct.ID,
			Debit:       invoice.TotalPrice,
			Description: fmt.Sprintf("Receivable for invoice %s", invoice.InvoiceNumber),
		},
		{
			AccountID:   revAcct.ID,
			Credit:      invoice.TaxableAmount,
			Description: fmt.Sprintf("Sales revenue for invoice %s", invoice.InvoiceNumber),
		},
	}

	// Add GST liability lines based on intra/inter-state flag.
	if invoice.IntraState {
		if invoice.CGSTAmount > 0 {
			cgstAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctCGSTPayable)
			if err != nil {
				return fmt.Errorf("accounting PostInvoice: CGST account: %w", err)
			}
			lines = append(lines, domain.LedgerEntry{
				AccountID:   cgstAcct.ID,
				Credit:      invoice.CGSTAmount,
				Description: "CGST payable",
			})
		}
		if invoice.SGSTAmount > 0 {
			sgstAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctSGSTPayable)
			if err != nil {
				return fmt.Errorf("accounting PostInvoice: SGST account: %w", err)
			}
			lines = append(lines, domain.LedgerEntry{
				AccountID:   sgstAcct.ID,
				Credit:      invoice.SGSTAmount,
				Description: "SGST payable",
			})
		}
	} else if invoice.IGSTAmount > 0 {
		igstAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctIGSTPayable)
		if err != nil {
			return fmt.Errorf("accounting PostInvoice: IGST account: %w", err)
		}
		lines = append(lines, domain.LedgerEntry{
			AccountID:   igstAcct.ID,
			Credit:      invoice.IGSTAmount,
			Description: "IGST payable",
		})
	}

	if err := validateBalance(lines); err != nil {
		return fmt.Errorf("accounting PostInvoice %s: %w", invoice.InvoiceNumber, err)
	}

	entry := &domain.JournalEntry{
		Date:        invoice.IssuedAt,
		Description: fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
		RefType:     "invoice",
		RefID:       invoice.ID.String(),
		CreatedBy:   createdBy,
		Lines:       lines,
	}
	return s.repo.CreateJournalEntry(ctx, entry)
}

func (s *accountingService) PostPayment(ctx context.Context, payment *domain.Payment) error {
	cashAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctCashBank)
	if err != nil {
		return fmt.Errorf("accounting PostPayment: cash account: %w", err)
	}
	arAcct, err := s.repo.FindAccountByCode(ctx, domain.AcctReceivable)
	if err != nil {
		return fmt.Errorf("accounting PostPayment: AR account: %w", err)
	}

	lines := []domain.LedgerEntry{
		{
			AccountID:   cashAcct.ID,
			Debit:       payment.Amount,
			Description: fmt.Sprintf("Payment received via %s", payment.Method),
		},
		{
			AccountID:   arAcct.ID,
			Credit:      payment.Amount,
			Description: fmt.Sprintf("Receivable settled (payment %s)", payment.ID),
		},
	}

	if err := validateBalance(lines); err != nil {
		return fmt.Errorf("accounting PostPayment %s: %w", payment.ID, err)
	}

	entry := &domain.JournalEntry{
		Date:        payment.PaidAt,
		Description: fmt.Sprintf("Payment received via %s — ₹%.2f", payment.Method, payment.Amount),
		RefType:     "payment",
		RefID:       payment.ID.String(),
		CreatedBy:   payment.RecordedBy,
		Lines:       lines,
	}
	return s.repo.CreateJournalEntry(ctx, entry)
}

// validateBalance ensures sum(debits) == sum(credits) to within a 1-paisa tolerance.
func validateBalance(lines []domain.LedgerEntry) error {
	var totalDebit, totalCredit float64
	for _, l := range lines {
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	if math.Abs(totalDebit-totalCredit) > 0.01 {
		return errors.New("journal entry is not balanced: debits and credits do not match")
	}
	return nil
}
