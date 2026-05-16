package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateCNTxInput carries the pre-validated credit note and the stock credits
// to apply inside a single atomic transaction.
type CreateCNTxInput struct {
	CreditNote *domain.CreditNote
	// StockCredits maps product_id → quantity to restore.
	StockCredits []StockDeduction // reuse StockDeduction (Quantity is positive here)
	ActorID      uuid.UUID
}

type CreditNoteRepository interface {
	CreateTx(ctx context.Context, in CreateCNTxInput) (*domain.CreditNote, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.CreditNote, error)
	VoidTx(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.CreditNote, error)
	ListByCustomer(ctx context.Context, customerID uint, limit, offset int) ([]domain.CreditNote, int64, error)
	ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.CreditNote, error)
}

type creditNoteRepository struct{ db *gorm.DB }

func NewCreditNoteRepository(db *gorm.DB) CreditNoteRepository {
	return &creditNoteRepository{db: db}
}

func (r *creditNoteRepository) CreateTx(ctx context.Context, in CreateCNTxInput) (*domain.CreditNote, error) {
	cn := in.CreditNote

	// Sort credits by ProductID for deterministic lock ordering (same pattern used
	// throughout — prevents AB/BA deadlocks under concurrent transactions).
	sorted := make([]StockDeduction, len(in.StockCredits))
	copy(sorted, in.StockCredits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ProductID.String() < sorted[j].ProductID.String()
	})

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		counter := domain.CNCounter{}
		if err := tx.Create(&counter).Error; err != nil {
			return err
		}
		cn.CNNumber = fmt.Sprintf("CN-%s-%07d", time.Now().UTC().Format("200601"), counter.ID)
		cn.IssuedAt = time.Now().UTC()

		if err := tx.Create(cn).Error; err != nil {
			return err
		}

		if len(sorted) == 0 {
			return nil
		}

		pids := make([]uuid.UUID, len(sorted))
		for i, s := range sorted {
			pids[i] = s.ProductID
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

		for _, credit := range sorted {
			before := stockMap[credit.ProductID]
			after := before + credit.Quantity

			if err := tx.Model(&domain.Product{}).
				Where("id = ?", credit.ProductID).
				UpdateColumn("stock", after).Error; err != nil {
				return err
			}

			sl := domain.StockLog{
				ProductID:      credit.ProductID,
				UserID:         in.ActorID,
				ChangeType:     domain.StockReturn,
				QuantityBefore: before,
				QuantityChange: credit.Quantity,
				QuantityAfter:  after,
				Note:           fmt.Sprintf("returned via credit note %s", cn.CNNumber),
			}
			if err := tx.Create(&sl).Error; err != nil {
				return err
			}
			stockMap[credit.ProductID] = after
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, cn.ID)
}

// VoidTx cancels the credit note and re-deducts the stock that was restored when
// it was issued. Only "issued" credit notes can be voided.
func (r *creditNoteRepository) VoidTx(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*domain.CreditNote, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cn domain.CreditNote
		if err := tx.Preload("Items").First(&cn, "id = ?", id).Error; err != nil {
			return domain.ErrNotFound
		}
		if cn.Status == domain.CNVoided {
			return domain.ErrCreditNoteAlreadyVoided
		}

		// Re-deduct the stock that was returned when the CN was created.
		pids := make([]uuid.UUID, 0, len(cn.Items))
		seen := map[uuid.UUID]bool{}
		for _, item := range cn.Items {
			if !seen[item.ProductID] {
				pids = append(pids, item.ProductID)
				seen[item.ProductID] = true
			}
		}
		sort.Slice(pids, func(i, j int) bool { return pids[i].String() < pids[j].String() })

		var products []domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", pids).Find(&products).Error; err != nil {
			return err
		}
		stockMap := map[uuid.UUID]int{}
		for _, p := range products {
			stockMap[p.ID] = p.Stock
		}

		for _, item := range cn.Items {
			before := stockMap[item.ProductID]
			after := before - item.Quantity
			if after < 0 {
				after = 0
			}
			if err := tx.Model(&domain.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("stock", after).Error; err != nil {
				return err
			}
			sl := domain.StockLog{
				ProductID:      item.ProductID,
				UserID:         actorID,
				ChangeType:     domain.StockAdjustment,
				QuantityBefore: before,
				QuantityChange: -(item.Quantity),
				QuantityAfter:  after,
				Note:           fmt.Sprintf("credit note %s voided — stock re-deducted", cn.CNNumber),
			}
			if err := tx.Create(&sl).Error; err != nil {
				return err
			}
			stockMap[item.ProductID] = after
		}

		now := time.Now().UTC()
		return tx.Model(&cn).Updates(map[string]any{
			"status":    domain.CNVoided,
			"voided_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *creditNoteRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.CreditNote, error) {
	var cn domain.CreditNote
	err := r.db.WithContext(ctx).
		Preload("Invoice").
		Preload("Customer").
		Preload("Issuer").
		Preload("Items.Product").
		First(&cn, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &cn, nil
}

func (r *creditNoteRepository) ListByCustomer(ctx context.Context, customerID uint, limit, offset int) ([]domain.CreditNote, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.CreditNote{}).
		Where("customer_id = ?", customerID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var cns []domain.CreditNote
	err := r.db.WithContext(ctx).
		Preload("Issuer").
		Where("customer_id = ?", customerID).
		Order("issued_at DESC").
		Limit(limit).Offset(offset).
		Find(&cns).Error
	return cns, total, err
}

func (r *creditNoteRepository) ListByInvoice(ctx context.Context, invoiceID uuid.UUID) ([]domain.CreditNote, error) {
	var cns []domain.CreditNote
	err := r.db.WithContext(ctx).
		Preload("Items.Product").
		Where("invoice_id = ?", invoiceID).
		Order("issued_at ASC").
		Find(&cns).Error
	return cns, err
}
