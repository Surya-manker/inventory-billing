package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/cache"
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

// pagedProducts is the cache-serialised shape for a paginated product list.
type pagedProducts struct {
	Items []domain.Product `json:"items"`
	Total int64            `json:"total"`
}

type productService struct {
	repo  repository.ProductRepository
	cache *cache.Cache // nil → caching disabled
}

// NewProductService creates a ProductService. Pass a non-nil *cache.Cache to
// enable transparent Redis caching of the product list (3-minute TTL).
func NewProductService(repo repository.ProductRepository, c *cache.Cache) ProductService {
	return &productService{repo: repo, cache: c}
}

// productListKey builds a deterministic cache key from filter + pagination.
// Changing any parameter produces a different key so stale data is never served.
func productListKey(f domain.ProductFilter, limit, offset int) string {
	cat := ""
	if f.CategoryID != nil {
		cat = f.CategoryID.String()
	}
	low := -1
	if f.LowStockMax != nil {
		low = *f.LowStockMax
	}
	return fmt.Sprintf("products:list:s=%s:c=%s:min=%.2f:max=%.2f:low=%d:by=%s:dir=%s:l=%d:o=%d",
		f.Search, cat, f.MinPrice, f.MaxPrice, low, f.SortBy, f.SortDir, limit, offset)
}

// invalidateProductCache removes all product list keys.
// We use a wildcard scan to be thorough — acceptable since mutations are rare.
func (s *productService) invalidateProductCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	// Best-effort; a cache miss on the next read is harmless.
	_ = s.cache.Del(ctx, "products:list:*")
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
	s.invalidateProductCache(ctx)
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
	s.invalidateProductCache(ctx)
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

	s.invalidateProductCache(ctx)
	return s.repo.Delete(ctx, id)
}

func (s *productService) List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error) {
	if s.cache == nil {
		return s.repo.List(ctx, filter, limit, offset)
	}
	key := productListKey(filter, limit, offset)
	result, err := cache.GetOrSet(ctx, s.cache, key, 3*time.Minute, func() (pagedProducts, error) {
		items, total, err := s.repo.List(ctx, filter, limit, offset)
		if err != nil {
			return pagedProducts{}, err
		}
		return pagedProducts{Items: items, Total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return result.Items, result.Total, nil
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

	log, err := s.repo.AdjustStock(ctx, input)
	if err == nil {
		s.invalidateProductCache(ctx)
	}
	return log, err
}
