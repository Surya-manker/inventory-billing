package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"gorm.io/gorm"
)

type ProductService interface {
	Create(ctx context.Context, input CreateProductInput) (*domain.Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error)
	AdjustStock(ctx context.Context, input domain.StockAdjustInput) (*domain.StockLog, error)
}

// CreateProductInput bundles all fields required to create a new product.
// Using a struct keeps the service interface stable when fields are added later.
type CreateProductInput struct {
	Name        string
	Description string
	SKU         string
	Price       float64
	Stock       int // initial stock — recorded as a "purchase" log entry if > 0
}

// UpdateProductInput holds the mutable fields of a product.
// SKU is intentionally excluded — it is an immutable identifier after creation.
type UpdateProductInput struct {
	Name        string
	Description string
	Price       float64
}

type productService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) Create(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
	if input.Price <= 0 {
		return nil, errors.New("price must be greater than zero")
	}
	if input.Stock < 0 {
		return nil, errors.New("initial stock cannot be negative")
	}

	existing, err := s.repo.FindBySKU(ctx, input.SKU)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrSKUTaken
	}

	product := &domain.Product{
		Name:        input.Name,
		Description: input.Description,
		SKU:         input.SKU,
		Price:       input.Price,
		Stock:       input.Stock,
	}
	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *productService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *productService) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	product, err := s.repo.FindBySKU(ctx, sku)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *productService) Update(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*domain.Product, error) {
	if input.Price <= 0 {
		return nil, errors.New("price must be greater than zero")
	}

	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	product.Name = input.Name
	product.Description = input.Description
	product.Price = input.Price

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

// Delete refuses to remove a product that is referenced by any invoice line item.
func (s *productService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}

	inUse, err := s.repo.IsProductInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrProductInUse
	}

	return s.repo.Delete(ctx, id)
}

func (s *productService) List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error) {
	return s.repo.List(ctx, filter, limit, offset)
}

// AdjustStock delegates to the repository's atomic operation and validates
// that manual adjustments never use the "sale" change type (that is reserved
// for the invoice flow).
func (s *productService) AdjustStock(ctx context.Context, input domain.StockAdjustInput) (*domain.StockLog, error) {
	if input.ChangeType == domain.StockSale {
		return nil, errors.New("change_type 'sale' is reserved for the invoice flow")
	}
	if input.QuantityChange == 0 {
		return nil, errors.New("quantity_change must be non-zero")
	}

	return s.repo.AdjustStock(ctx, input)
}
