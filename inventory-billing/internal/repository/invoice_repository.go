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

// StockDeduction tells the repository how many units to deduct from a product
// inside an invoice transaction.
type StockDeduction struct {
	ProductID uuid.UUID
	UserID    uuid.UUID
	Quantity  int // positive units to deduct
}

// CreateInvoiceTxInput bundles the pre-computed invoice (built by the service)
// and the list of stock movements to execute atomically alongside it.
type CreateInvoiceTxInput struct {
	Invoice    *domain.Invoice
	Deductions []StockDeduction
}

type InvoiceRepository interface {
	// CreateTx atomically:
	//   1. Locks product rows FOR UPDATE in deterministic order (prevents deadlocks)
	//   2. Re-validates stock availability (handles concurrent requests)
	//   3. Generates an invoice number via DB sequence
	//   4. Inserts invoice + items
	//   5. Deducts stock and writes a StockLog for each item
	CreateTx(ctx context.Context, input CreateInvoiceTxInput) (*domain.Invoice, error)

	FindByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	FindByNumber(ctx context.Context, number string) (*domain.Invoice, error)

	// MarkPaid sets status = 'paid' and records paid_at.
	MarkPaid(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)

	// CancelTx atomically sets status = 'canceled' and restores stock for every item.
	CancelTx(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Invoice, error)

	// DeleteTx checks that the invoice is not paid, restores stock if pending,
	// then hard-deletes (items cascade).
	DeleteTx(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	List(ctx context.Context, filter domain.InvoiceFilter, limit, offset int) ([]domain.Invoice, int64, error)
}

type invoiceRepository struct {
	db *gorm.DB
}

func NewInvoiceRepository(db *gorm.DB) InvoiceRepository {
	return &invoiceRepository{db: db}
}

// ── write operations ──────────────────────────────────────────────────────────

func (r *invoiceRepository) CreateTx(ctx context.Context, input CreateInvoiceTxInput) (*domain.Invoice, error) {
	invoice := input.Invoice

	// Sort deductions by ProductID so all concurrent transactions lock in the
	// same order — this prevents the classic AB/BA deadlock pattern.
	sorted := make([]StockDeduction, len(input.Deductions))
	copy(sorted, input.Deductions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ProductID.String() < sorted[j].ProductID.String()
	})

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock all referenced products and re-verify stock.
		productIDs := make([]uuid.UUID, len(sorted))
		for i, d := range sorted {
			productIDs[i] = d.ProductID
		}

		var products []domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", productIDs).
			Find(&products).Error; err != nil {
			return err
		}

		stockMap := make(map[uuid.UUID]int, len(products))
		for _, p := range products {
			stockMap[p.ID] = p.Stock
		}

		for _, d := range sorted {
			stock, ok := stockMap[d.ProductID]
			if !ok {
				return domain.ErrNotFound
			}
			if stock < d.Quantity {
				return domain.ErrInsufficientStock
			}
		}

		// 2. Generate invoice number using the InvoiceCounter auto-increment table.
		// INSERT + last_insert_id() is atomic in MySQL InnoDB — no sequences needed.
		counter := domain.InvoiceCounter{}
		if err := tx.Create(&counter).Error; err != nil {
			return err
		}
		invoice.InvoiceNumber = fmt.Sprintf("INV-%s-%07d", time.Now().UTC().Format("200601"), counter.ID)
		invoice.IssuedAt = time.Now().UTC()

		// 3. Insert invoice — GORM cascades into invoice_items via the Items association.
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}

		// 4. Deduct stock and record the audit log for each line.
		for _, d := range sorted {
			before := stockMap[d.ProductID]
			after := before - d.Quantity

			if err := tx.Model(&domain.Product{}).
				Where("id = ?", d.ProductID).
				UpdateColumn("stock", after).Error; err != nil {
				return err
			}

			log := domain.StockLog{
				ProductID:      d.ProductID,
				UserID:         d.UserID,
				InvoiceID:      &invoice.ID,
				ChangeType:     domain.StockSale,
				QuantityBefore: before,
				QuantityChange: -d.Quantity,
				QuantityAfter:  after,
				Note:           fmt.Sprintf("sold via invoice %s", invoice.InvoiceNumber),
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.FindByID(ctx, invoice.ID)
}

func (r *invoiceRepository) MarkPaid(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	now := time.Now().UTC()
	// Also align paid_amount with total_price so OutstandingBalance() is zero
	// when an admin manually marks an invoice paid (e.g. write-off).
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv domain.Invoice
		if err := tx.First(&inv, "id = ?", id).Error; err != nil {
			return domain.ErrNotFound
		}
		return tx.Model(&domain.Invoice{}).Where("id = ?", id).Updates(map[string]any{
			"status":      domain.InvoicePaid,
			"paid_amount": inv.TotalPrice,
			"paid_at":     now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *invoiceRepository) CancelTx(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Invoice, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invoice domain.Invoice
		if err := tx.Preload("Items").First(&invoice, "id = ?", id).Error; err != nil {
			return domain.ErrNotFound
		}

		if err := r.restoreStock(tx, &invoice, userID); err != nil {
			return err
		}

		return tx.Model(&invoice).Update("status", domain.InvoiceCanceled).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *invoiceRepository) DeleteTx(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invoice domain.Invoice
		if err := tx.Preload("Items").First(&invoice, "id = ?", id).Error; err != nil {
			return domain.ErrNotFound
		}

		switch invoice.Status {
		case domain.InvoicePaid:
			return domain.ErrCannotDeletePaidInvoice
		case domain.InvoicePending, domain.InvoicePartial, domain.InvoiceOverdue:
			// Restore stock before removing the record.
			if err := r.restoreStock(tx, &invoice, userID); err != nil {
				return err
			}
		case domain.InvoiceCanceled:
			// Stock was already restored when the invoice was canceled — just delete.
		}

		return tx.Delete(&domain.Invoice{}, "id = ?", id).Error
	})
}

// ── read operations ───────────────────────────────────────────────────────────

func (r *invoiceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := r.db.WithContext(ctx).
		Preload("Customer").
		Preload("Creator").
		Preload("Items.Product").
		First(&invoice, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepository) FindByNumber(ctx context.Context, number string) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := r.db.WithContext(ctx).
		Preload("Customer").
		Preload("Items.Product").
		Where("invoice_number = ?", number).
		First(&invoice).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepository) List(ctx context.Context, filter domain.InvoiceFilter, limit, offset int) ([]domain.Invoice, int64, error) {
	var total int64
	if err := r.applyFilters(ctx, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := utils.OrderClause(filter.SortBy, filter.SortDir,
		[]string{"issued_at", "due_at", "total_price", "created_at"}, "issued_at")

	var invoices []domain.Invoice
	err := r.applyFilters(ctx, filter).
		Preload("Customer").
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&invoices).Error
	if err != nil {
		return nil, 0, err
	}
	return invoices, total, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// restoreStock creates a StockReturn log and adds back the quantity for every
// line item. Called inside an existing transaction.
func (r *invoiceRepository) restoreStock(tx *gorm.DB, invoice *domain.Invoice, userID uuid.UUID) error {
	// Collect unique product IDs and lock them in sorted order.
	ids := make([]uuid.UUID, 0, len(invoice.Items))
	seen := make(map[uuid.UUID]bool)
	for _, item := range invoice.Items {
		if !seen[item.ProductID] {
			ids = append(ids, item.ProductID)
			seen[item.ProductID] = true
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

	var products []domain.Product
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).Find(&products).Error; err != nil {
		return err
	}
	stockMap := make(map[uuid.UUID]int, len(products))
	for _, p := range products {
		stockMap[p.ID] = p.Stock
	}

	for _, item := range invoice.Items {
		before := stockMap[item.ProductID]
		after := before + item.Quantity

		if err := tx.Model(&domain.Product{}).
			Where("id = ?", item.ProductID).
			UpdateColumn("stock", after).Error; err != nil {
			return err
		}

		log := domain.StockLog{
			ProductID:      item.ProductID,
			UserID:         userID,
			InvoiceID:      &invoice.ID,
			ChangeType:     domain.StockReturn,
			QuantityBefore: before,
			QuantityChange: item.Quantity,
			QuantityAfter:  after,
			Note:           fmt.Sprintf("stock returned from invoice %s", invoice.InvoiceNumber),
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		stockMap[item.ProductID] = after // keep local map consistent for multi-line same product
	}
	return nil
}

func (r *invoiceRepository) applyFilters(ctx context.Context, f domain.InvoiceFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&domain.Invoice{})
	if f.CustomerID != nil && *f.CustomerID > 0 {
		q = q.Where("customer_id = ?", *f.CustomerID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.FromDate != nil {
		q = q.Where("issued_at >= ?", *f.FromDate)
	}
	if f.ToDate != nil {
		q = q.Where("issued_at <= ?", *f.ToDate)
	}
	return q
}

