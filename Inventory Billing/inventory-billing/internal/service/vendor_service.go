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

// ── Vendor ────────────────────────────────────────────────────────────────────

type CreateVendorInput struct {
	Name             string
	Email            string
	Phone            string
	Address          string
	Company          string
	GSTIN            string
	ContactPerson    string
	PaymentTermsDays int
}

type VendorService interface {
	Create(ctx context.Context, in CreateVendorInput) (*domain.Vendor, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Vendor, error)
	Update(ctx context.Context, id uuid.UUID, in CreateVendorInput) (*domain.Vendor, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter domain.VendorFilter, limit, offset int) ([]domain.Vendor, int64, error)
}

type vendorService struct {
	repo repository.VendorRepository
}

func NewVendorService(repo repository.VendorRepository) VendorService {
	return &vendorService{repo: repo}
}

func (s *vendorService) Create(ctx context.Context, in CreateVendorInput) (*domain.Vendor, error) {
	if existing, err := s.repo.FindByEmail(ctx, in.Email); err == nil && existing != nil {
		return nil, domain.ErrVendorEmailTaken
	}
	terms := in.PaymentTermsDays
	if terms <= 0 {
		terms = 30
	}
	v := &domain.Vendor{
		Name: in.Name, Email: in.Email, Phone: in.Phone,
		Address: in.Address, Company: in.Company, GSTIN: in.GSTIN,
		ContactPerson:    in.ContactPerson,
		PaymentTermsDays: terms,
		IsActive:         true,
	}
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *vendorService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vendor, error) {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return v, nil
}

func (s *vendorService) Update(ctx context.Context, id uuid.UUID, in CreateVendorInput) (*domain.Vendor, error) {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if existing, err := s.repo.FindByEmail(ctx, in.Email); err == nil && existing.ID != id {
		return nil, domain.ErrVendorEmailTaken
	}
	v.Name, v.Email, v.Phone = in.Name, in.Email, in.Phone
	v.Address, v.Company, v.GSTIN = in.Address, in.Company, in.GSTIN
	v.ContactPerson, v.PaymentTermsDays = in.ContactPerson, in.PaymentTermsDays
	return v, s.repo.Update(ctx, v)
}

func (s *vendorService) Deactivate(ctx context.Context, id uuid.UUID) error {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	v.IsActive = false
	return s.repo.Update(ctx, v)
}

func (s *vendorService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *vendorService) List(ctx context.Context, filter domain.VendorFilter, limit, offset int) ([]domain.Vendor, int64, error) {
	return s.repo.List(ctx, filter, limit, offset)
}

// ── Purchase Order ────────────────────────────────────────────────────────────

type POItemInput struct {
	ProductID uuid.UUID
	Quantity  int
	UnitCost  float64
}

type CreatePOInput struct {
	VendorID   uuid.UUID
	CreatedBy  uuid.UUID
	Items      []POItemInput
	Notes      string
	ExpectedAt time.Time
}

type PurchaseService interface {
	Create(ctx context.Context, in CreatePOInput) (*domain.PurchaseOrder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error)
	Confirm(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error)
	Receive(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.PurchaseOrder, error)
	Cancel(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, vendorID *uuid.UUID, status string, limit, offset int) ([]domain.PurchaseOrder, int64, error)
}

type purchaseService struct {
	repo       repository.PurchaseRepository
	vendorRepo repository.VendorRepository
}

func NewPurchaseService(repo repository.PurchaseRepository, vendorRepo repository.VendorRepository) PurchaseService {
	return &purchaseService{repo: repo, vendorRepo: vendorRepo}
}

func (s *purchaseService) Create(ctx context.Context, in CreatePOInput) (*domain.PurchaseOrder, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("purchase order must have at least one item")
	}
	vendor, err := s.vendorRepo.FindByID(ctx, in.VendorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if !vendor.IsActive {
		return nil, domain.ErrVendorInactive
	}

	var items []domain.POItem
	var subTotal, taxAmount float64
	for _, it := range in.Items {
		lineSub := it.UnitCost * float64(it.Quantity)
		// Use 18% GST on purchases by default; real app would look up product tax_rate
		lineTax := 0.0
		items = append(items, domain.POItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			UnitCost:  it.UnitCost,
			TaxRate:   0,
			TaxAmount: lineTax,
			TotalCost: lineSub + lineTax,
		})
		subTotal += lineSub
		taxAmount += lineTax
	}

	po := &domain.PurchaseOrder{
		VendorID:    in.VendorID,
		CreatedBy:   in.CreatedBy,
		Items:       items,
		SubTotal:    subTotal,
		TaxAmount:   taxAmount,
		TotalAmount: subTotal + taxAmount,
		Status:      domain.PODraft,
		Notes:       in.Notes,
		ExpectedAt:  in.ExpectedAt,
	}
	return s.repo.Create(ctx, po)
}

func (s *purchaseService) GetByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error) {
	po, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return po, nil
}

func (s *purchaseService) transition(ctx context.Context, id uuid.UUID, to domain.POStatus) (*domain.PurchaseOrder, error) {
	po, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	allowed := domain.ValidPOTransitions[po.Status]
	ok := false
	for _, s := range allowed {
		if s == to {
			ok = true
			break
		}
	}
	if !ok {
		return nil, domain.ErrInvalidPOTransition
	}
	po.Status = to
	return po, s.repo.Update(ctx, po)
}

func (s *purchaseService) Confirm(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error) {
	return s.transition(ctx, id, domain.POConfirmed)
}

func (s *purchaseService) Receive(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.PurchaseOrder, error) {
	return s.repo.ReceiveTx(ctx, id, actorID)
}

func (s *purchaseService) Cancel(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error) {
	return s.transition(ctx, id, domain.POCanceled)
}

func (s *purchaseService) Delete(ctx context.Context, id uuid.UUID) error {
	po, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	if po.Status == domain.POConfirmed || po.Status == domain.POReceived {
		return domain.ErrCannotDeleteConfirmedPO
	}
	return s.repo.Delete(ctx, id)
}

func (s *purchaseService) List(ctx context.Context, vendorID *uuid.UUID, status string, limit, offset int) ([]domain.PurchaseOrder, int64, error) {
	return s.repo.List(ctx, vendorID, status, limit, offset)
}
