package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Vendor ────────────────────────────────────────────────────────────────────

type VendorRepository interface {
	Create(ctx context.Context, v *domain.Vendor) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Vendor, error)
	FindByEmail(ctx context.Context, email string) (*domain.Vendor, error)
	Update(ctx context.Context, v *domain.Vendor) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter domain.VendorFilter, limit, offset int) ([]domain.Vendor, int64, error)
}

type vendorRepository struct{ db *gorm.DB }

func NewVendorRepository(db *gorm.DB) VendorRepository {
	return &vendorRepository{db: db}
}

func (r *vendorRepository) Create(ctx context.Context, v *domain.Vendor) error {
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *vendorRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Vendor, error) {
	var v domain.Vendor
	if err := r.db.WithContext(ctx).First(&v, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *vendorRepository) FindByEmail(ctx context.Context, email string) (*domain.Vendor, error) {
	var v domain.Vendor
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *vendorRepository) Update(ctx context.Context, v *domain.Vendor) error {
	return r.db.WithContext(ctx).Model(v).Updates(map[string]any{
		"name":               v.Name,
		"email":              v.Email,
		"phone":              v.Phone,
		"address":            v.Address,
		"company":            v.Company,
		"gstin":              v.GSTIN,
		"contact_person":     v.ContactPerson,
		"payment_terms_days": v.PaymentTermsDays,
		"is_active":          v.IsActive,
	}).Error
}

func (r *vendorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Vendor{}, "id = ?", id).Error
}

func (r *vendorRepository) List(ctx context.Context, filter domain.VendorFilter, limit, offset int) ([]domain.Vendor, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Vendor{})
	if filter.Search != "" {
		p := "%" + filter.Search + "%"
		q = q.Where("name LIKE ? OR email LIKE ? OR company LIKE ?", p, p, p)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var vendors []domain.Vendor
	if err := q.Order("name ASC").Limit(limit).Offset(offset).Find(&vendors).Error; err != nil {
		return nil, 0, err
	}
	return vendors, total, nil
}

// ── Purchase Order ────────────────────────────────────────────────────────────

// POStockCredit tells CreateTx to credit stock for a received PO item.
type POStockCredit struct {
	ProductID uuid.UUID
	UserID    uuid.UUID
	Quantity  int
	UnitCost  float64
	PONumber  string
}

type PurchaseRepository interface {
	Create(ctx context.Context, po *domain.PurchaseOrder) (*domain.PurchaseOrder, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error)
	Update(ctx context.Context, po *domain.PurchaseOrder) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, vendorID *uuid.UUID, status string, limit, offset int) ([]domain.PurchaseOrder, int64, error)
	// ReceiveTx atomically marks the PO as received, credits stock for every item,
	// and writes a StockLog of type "purchase" for each line.
	ReceiveTx(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.PurchaseOrder, error)
}

type purchaseRepository struct{ db *gorm.DB }

func NewPurchaseRepository(db *gorm.DB) PurchaseRepository {
	return &purchaseRepository{db: db}
}

func (r *purchaseRepository) Create(ctx context.Context, po *domain.PurchaseOrder) (*domain.PurchaseOrder, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		counter := domain.POCounter{}
		if err := tx.Create(&counter).Error; err != nil {
			return err
		}
		po.PONumber = fmt.Sprintf("PO-%s-%07d", time.Now().UTC().Format("200601"), counter.ID)
		return tx.Create(po).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, po.ID)
}

func (r *purchaseRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error) {
	var po domain.PurchaseOrder
	err := r.db.WithContext(ctx).
		Preload("Vendor").
		Preload("Creator").
		Preload("Items.Product").
		First(&po, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *purchaseRepository) Update(ctx context.Context, po *domain.PurchaseOrder) error {
	return r.db.WithContext(ctx).Model(po).Updates(map[string]any{
		"status":      po.Status,
		"notes":       po.Notes,
		"expected_at": po.ExpectedAt,
	}).Error
}

func (r *purchaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PurchaseOrder{}, "id = ?", id).Error
}

func (r *purchaseRepository) List(ctx context.Context, vendorID *uuid.UUID, status string, limit, offset int) ([]domain.PurchaseOrder, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.PurchaseOrder{})
	if vendorID != nil {
		q = q.Where("vendor_id = ?", *vendorID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var pos []domain.PurchaseOrder
	err := q.Preload("Vendor").
		Order(utils.OrderClause("", "", nil, "created_at")).
		Limit(limit).Offset(offset).Find(&pos).Error
	return pos, total, err
}

// ReceiveTx locks product rows in sorted order (same deadlock-prevention pattern
// as invoice_repository), credits each item's quantity to stock, and appends a
// StockLog row of type "purchase".
func (r *purchaseRepository) ReceiveTx(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.PurchaseOrder, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po domain.PurchaseOrder
		if err := tx.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
			return domain.ErrNotFound
		}
		if po.Status != domain.POConfirmed {
			return domain.ErrInvalidPOTransition
		}

		// Sort items by product UUID to lock in deterministic order.
		items := make([]domain.POItem, len(po.Items))
		copy(items, po.Items)
		sort.Slice(items, func(i, j int) bool {
			return items[i].ProductID.String() < items[j].ProductID.String()
		})

		pids := make([]uuid.UUID, len(items))
		for i, it := range items {
			pids[i] = it.ProductID
		}

		var products []domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", pids).Find(&products).Error; err != nil {
			return err
		}
		stockMap := map[uuid.UUID]int{}
		for _, p := range products {
			stockMap[p.ID] = p.Stock
		}

		now := time.Now().UTC()
		for _, item := range items {
			before := stockMap[item.ProductID]
			after := before + item.Quantity

			if err := tx.Model(&domain.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("stock", after).Error; err != nil {
				return err
			}

			sl := domain.StockLog{
				ProductID:      item.ProductID,
				UserID:         actorID,
				ChangeType:     domain.StockPurchase,
				QuantityBefore: before,
				QuantityChange: item.Quantity,
				QuantityAfter:  after,
				Note:           fmt.Sprintf("received via %s", po.PONumber),
			}
			if err := tx.Create(&sl).Error; err != nil {
				return err
			}
			stockMap[item.ProductID] = after

			// Mark received quantity on the item.
			if err := tx.Model(&domain.POItem{}).
				Where("id = ?", item.ID).
				UpdateColumn("received_qty", item.Quantity).Error; err != nil {
				return err
			}
		}

		return tx.Model(&po).Updates(map[string]any{
			"status":      domain.POReceived,
			"received_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}
