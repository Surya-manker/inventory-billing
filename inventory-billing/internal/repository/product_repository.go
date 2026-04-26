package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	FindBySKU(ctx context.Context, sku string) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error)

	// AdjustStock atomically updates the product's stock count and appends a
	// StockLog row — both inside a single DB transaction.
	AdjustStock(ctx context.Context, input domain.StockAdjustInput) (*domain.StockLog, error)

	// IsProductInUse returns true when at least one invoice_item references this product.
	// Used to guard deletion.
	IsProductInUse(ctx context.Context, id uuid.UUID) (bool, error)

	// ListLowStock returns products whose stock is at or below their own
	// low_stock_threshold. Used by the stock alert endpoint.
	ListLowStock(ctx context.Context) ([]domain.Product, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *productRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var product domain.Product
	if err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	var product domain.Product
	if err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", id).Error
}

// List applies optional filters, counts the total matching rows (for pagination
// metadata), then fetches the page. Two queries share the same filter builder so
// the WHERE clause is always consistent.
func (r *productRepository) List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error) {
	var total int64
	if err := r.applyFilters(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := r.resolveOrder(filter.SortBy, filter.SortDir)

	var products []domain.Product
	err := r.applyFilters(ctx, filter).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&products).Error
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// AdjustStock locks the product row (SELECT … FOR UPDATE), validates the result
// would not go negative, updates the stock column, and creates a StockLog row —
// all inside one transaction so no partial state is ever committed.
func (r *productRepository) AdjustStock(ctx context.Context, input domain.StockAdjustInput) (*domain.StockLog, error) {
	var log domain.StockLog

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&product, "id = ?", input.ProductID).Error; err != nil {
			return domain.ErrNotFound
		}

		after := product.Stock + input.QuantityChange
		if after < 0 {
			return domain.ErrInsufficientStock
		}

		if err := tx.Model(&product).UpdateColumn("stock", after).Error; err != nil {
			return err
		}

		log = domain.StockLog{
			ProductID:      input.ProductID,
			UserID:         input.UserID,
			InvoiceID:      input.InvoiceID,
			ChangeType:     input.ChangeType,
			QuantityBefore: product.Stock,
			QuantityChange: input.QuantityChange,
			QuantityAfter:  after,
			Note:           input.Note,
		}
		return tx.Create(&log).Error
	})
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *productRepository) IsProductInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.InvoiceItem{}).
		Where("product_id = ?", id).
		Count(&count).Error
	return count > 0, err
}

func (r *productRepository) ListLowStock(ctx context.Context) ([]domain.Product, error) {
	var products []domain.Product
	// stock <= low_stock_threshold covers both "at threshold" and "below"
	err := r.db.WithContext(ctx).
		Where("stock <= low_stock_threshold").
		Order("stock ASC").
		Find(&products).Error
	return products, err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (r *productRepository) applyFilters(ctx context.Context, f domain.ProductFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&domain.Product{})

	if f.Search != "" {
		pattern := "%" + f.Search + "%"
		// MySQL LIKE is case-insensitive with utf8mb4_general_ci collation
		q = q.Where("name LIKE ? OR sku LIKE ?", pattern, pattern)
	}
	if f.MinPrice > 0 {
		q = q.Where("price >= ?", f.MinPrice)
	}
	if f.MaxPrice > 0 {
		q = q.Where("price <= ?", f.MaxPrice)
	}
	if f.LowStockMax != nil {
		q = q.Where("stock <= ?", *f.LowStockMax)
	}
	return q
}

func (r *productRepository) resolveOrder(sortBy, sortDir string) string {
	allowed := map[string]bool{
		"name": true, "price": true, "stock": true, "created_at": true,
	}
	col := "created_at"
	if allowed[sortBy] {
		col = sortBy
	}

	dir := "DESC"
	if strings.EqualFold(sortDir, "asc") {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", col, dir)
}
