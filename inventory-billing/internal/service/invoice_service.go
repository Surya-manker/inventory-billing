package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"gorm.io/gorm"
)

// ── request input types ───────────────────────────────────────────────────────

type InvoiceItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type CreateInvoiceInput struct {
	CustomerID uint      // auto-increment uint matching Customer.ID
	CreatedBy  uuid.UUID // from JWT — never from the request body
	Items      []InvoiceItemInput
	Discount   float64 // flat discount amount, default 0
	Notes      string
	DueAt      time.Time
}

// ── service interface ─────────────────────────────────────────────────────────

type InvoiceService interface {
	Create(ctx context.Context, input CreateInvoiceInput) (*domain.Invoice, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByNumber(ctx context.Context, number string) (*domain.Invoice, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, newStatus domain.InvoiceStatus, actorID uuid.UUID) (*domain.Invoice, error)
	Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
	List(ctx context.Context, filter domain.InvoiceFilter, limit, offset int) ([]domain.Invoice, int64, error)
}

// ── implementation ────────────────────────────────────────────────────────────

type invoiceService struct {
	invoiceRepo    repository.InvoiceRepository
	productRepo    repository.ProductRepository
	customerRepo   repository.CustomerRepository
	accountingSvc  AccountingService
	stateCode      string // seller GST state code for intra/inter-state detection
}

func NewInvoiceService(
	invoiceRepo repository.InvoiceRepository,
	productRepo repository.ProductRepository,
	customerRepo repository.CustomerRepository,
	accountingSvc AccountingService,
	sellerStateCode string,
) InvoiceService {
	return &invoiceService{
		invoiceRepo:   invoiceRepo,
		productRepo:   productRepo,
		customerRepo:  customerRepo,
		accountingSvc: accountingSvc,
		stateCode:     sellerStateCode,
	}
}

// Create validates all inputs, computes per-item and invoice-level financials
// (including GST split), then delegates the atomic persistence to the repository.
func (s *invoiceService) Create(ctx context.Context, input CreateInvoiceInput) (*domain.Invoice, error) {
	if len(input.Items) == 0 {
		return nil, errors.New("invoice must contain at least one item")
	}
	if input.DueAt.IsZero() || input.DueAt.UTC().Before(time.Now().UTC()) {
		return nil, errors.New("due_at must be a future date")
	}

	// Verify customer exists and retrieve GSTIN for intra/inter-state detection.
	customer, err := s.customerRepo.FindByID(ctx, input.CustomerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	intraState := utils.IsIntraState(s.stateCode, customer.GSTIN)

	// ── Build line items and accumulate totals ────────────────────────────────
	var (
		items      []domain.InvoiceItem
		deductions []repository.StockDeduction
		subTotal   float64
		taxAmount  float64
	)

	for _, req := range input.Items {
		product, err := s.productRepo.FindByID(ctx, req.ProductID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}

		// Pre-flight stock check — the repository re-validates inside the tx.
		if product.Stock < req.Quantity {
			return nil, fmt.Errorf("%w: %s (available: %d, requested: %d)",
				domain.ErrInsufficientStock, product.Name, product.Stock, req.Quantity)
		}

		lineSubTotal := utils.Round2(product.Price * float64(req.Quantity))
		lineTax := utils.Round2(lineSubTotal * product.TaxRate / 100)
		lineTotal := utils.Round2(lineSubTotal + lineTax)

		items = append(items, domain.InvoiceItem{
			ProductID: product.ID,
			Quantity:  req.Quantity,
			UnitPrice: product.Price,
			CostPrice: product.CostPrice, // snapshot at sale time for COGS tracking
			SubTotal:  lineSubTotal,
			TaxRate:   product.TaxRate,
			TaxAmount: lineTax,
			Total:     lineTotal,
		})

		deductions = append(deductions, repository.StockDeduction{
			ProductID: product.ID,
			UserID:    input.CreatedBy,
			Quantity:  req.Quantity,
		})

		subTotal += lineSubTotal
		taxAmount += lineTax
	}

	subTotal = utils.Round2(subTotal)
	taxAmount = utils.Round2(taxAmount)

	if input.Discount < 0 {
		return nil, errors.New("discount cannot be negative")
	}
	if input.Discount > subTotal {
		return nil, domain.ErrDiscountExceedsSubtotal
	}

	discount := utils.Round2(input.Discount)
	taxableAmount := utils.Round2(subTotal - discount)
	totalPrice := utils.Round2(taxableAmount + taxAmount)

	// ── GST split (CGST+SGST for intra-state, IGST for inter-state) ──────────
	// Compute the effective blended tax rate across all line items so the invoice
	// summary shows a meaningful rate alongside the amount.
	var effectiveTaxRate float64
	if subTotal > 0 {
		effectiveTaxRate = utils.Round2(taxAmount / subTotal * 100)
	}
	gst := utils.SplitGST(taxAmount, effectiveTaxRate, intraState)

	invoice := &domain.Invoice{
		CustomerID:    input.CustomerID,
		CreatedBy:     input.CreatedBy,
		Items:         items,
		SubTotal:      subTotal,
		Discount:      discount,
		TaxableAmount: taxableAmount,
		TaxAmount:     taxAmount,
		TotalPrice:    totalPrice,
		IntraState:    intraState,
		CGSTAmount:    gst.CGSTAmount,
		SGSTAmount:    gst.SGSTAmount,
		IGSTAmount:    gst.IGSTAmount,
		Status:        domain.InvoicePending,
		Notes:         input.Notes,
		DueAt:         input.DueAt,
	}

	created, err := s.invoiceRepo.CreateTx(ctx, repository.CreateInvoiceTxInput{
		Invoice:    invoice,
		Deductions: deductions,
	})
	if err != nil {
		return nil, err
	}

	// Post accounting entry after the invoice transaction commits.
	// Runs in its own transaction — a failure here is logged by the repository
	// layer but does not roll back the already-committed invoice.
	_ = s.accountingSvc.PostInvoice(ctx, created, input.CreatedBy)

	return created, nil
}

func (s *invoiceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return invoice, nil
}

func (s *invoiceService) GetByNumber(ctx context.Context, number string) (*domain.Invoice, error) {
	invoice, err := s.invoiceRepo.FindByNumber(ctx, number)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return invoice, nil
}

// UpdateStatus enforces the valid transition table from domain.ValidTransitions.
func (s *invoiceService) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus domain.InvoiceStatus, actorID uuid.UUID) (*domain.Invoice, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if !isValidTransition(invoice.Status, newStatus) {
		return nil, domain.ErrInvalidStatusTransition
	}

	switch newStatus {
	case domain.InvoicePaid:
		return s.invoiceRepo.MarkPaid(ctx, id)
	case domain.InvoiceCanceled:
		return s.invoiceRepo.CancelTx(ctx, id, actorID)
	default:
		return nil, domain.ErrInvalidStatusTransition
	}
}

func (s *invoiceService) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	return s.invoiceRepo.DeleteTx(ctx, id, actorID)
}

func (s *invoiceService) List(ctx context.Context, filter domain.InvoiceFilter, limit, offset int) ([]domain.Invoice, int64, error) {
	return s.invoiceRepo.List(ctx, filter, limit, offset)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isValidTransition(current, next domain.InvoiceStatus) bool {
	allowed, ok := domain.ValidTransitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}
