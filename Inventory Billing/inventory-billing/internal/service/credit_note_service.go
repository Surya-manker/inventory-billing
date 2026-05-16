package service

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"gorm.io/gorm"
)

type CNItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type IssueCreditNoteInput struct {
	InvoiceID uuid.UUID
	IssuedBy  uuid.UUID
	Items     []CNItemInput
	Reason    string
	Notes     string
}

type CreditNoteService interface {
	Issue(ctx context.Context, in IssueCreditNoteInput) (*domain.CreditNote, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CreditNote, error)
	Void(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.CreditNote, error)
	ListByCustomer(ctx context.Context, customerID uint, limit, offset int) ([]domain.CreditNote, int64, error)
	ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.CreditNote, error)
}

type creditNoteService struct {
	cnRepo      repository.CreditNoteRepository
	invoiceRepo repository.InvoiceRepository
}

func NewCreditNoteService(cnRepo repository.CreditNoteRepository, invRepo repository.InvoiceRepository) CreditNoteService {
	return &creditNoteService{cnRepo: cnRepo, invoiceRepo: invRepo}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// Issue validates the return request, computes financials, then delegates
// the atomic DB work to the repository.
func (s *creditNoteService) Issue(ctx context.Context, in IssueCreditNoteInput) (*domain.CreditNote, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("credit note must contain at least one item")
	}
	if in.Reason == "" {
		return nil, errors.New("reason is required")
	}

	invoice, err := s.invoiceRepo.FindByID(ctx, in.InvoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if invoice.Status != domain.InvoicePaid {
		return nil, domain.ErrCreditNoteInvoiceNotPaid
	}

	// Build a lookup: product_id → original InvoiceItem for validation and price snapshot.
	origItems := map[uuid.UUID]domain.InvoiceItem{}
	for _, item := range invoice.Items {
		origItems[item.ProductID] = item
	}

	var cnItems []domain.CreditNoteItem
	var credits []repository.StockDeduction
	var subTotal, taxAmount float64

	for _, req := range in.Items {
		orig, ok := origItems[req.ProductID]
		if !ok {
			return nil, domain.ErrCreditNoteProductNotInInv
		}
		if req.Quantity <= 0 || req.Quantity > orig.Quantity {
			return nil, domain.ErrCreditNoteQuantityExceeds
		}

		lineSub := round2(orig.UnitPrice * float64(req.Quantity))
		lineTax := round2(lineSub * orig.TaxRate / 100)

		cnItems = append(cnItems, domain.CreditNoteItem{
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			UnitPrice: orig.UnitPrice,
			SubTotal:  lineSub,
			TaxRate:   orig.TaxRate,
			TaxAmount: lineTax,
			Total:     round2(lineSub + lineTax),
		})
		credits = append(credits, repository.StockDeduction{
			ProductID: req.ProductID,
			UserID:    in.IssuedBy,
			Quantity:  req.Quantity,
		})
		subTotal += lineSub
		taxAmount += lineTax
	}

	subTotal   = round2(subTotal)
	taxAmount  = round2(taxAmount)

	cn := &domain.CreditNote{
		InvoiceID:   in.InvoiceID,
		CustomerID:  invoice.CustomerID,
		IssuedBy:    in.IssuedBy,
		Items:       cnItems,
		SubTotal:    subTotal,
		TaxAmount:   taxAmount,
		TotalAmount: round2(subTotal + taxAmount),
		Status:      domain.CNIssued,
		Reason:      in.Reason,
		Notes:       in.Notes,
	}

	return s.cnRepo.CreateTx(ctx, repository.CreateCNTxInput{
		CreditNote:   cn,
		StockCredits: credits,
		ActorID:      in.IssuedBy,
	})
}

func (s *creditNoteService) GetByID(ctx context.Context, id uuid.UUID) (*domain.CreditNote, error) {
	cn, err := s.cnRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return cn, nil
}

func (s *creditNoteService) Void(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.CreditNote, error) {
	return s.cnRepo.VoidTx(ctx, id, actorID)
}

func (s *creditNoteService) ListByCustomer(ctx context.Context, customerID uint, limit, offset int) ([]domain.CreditNote, int64, error) {
	return s.cnRepo.ListByCustomer(ctx, customerID, limit, offset)
}

func (s *creditNoteService) ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.CreditNote, error) {
	return s.cnRepo.ListByInvoice(ctx, invoiceID)
}
