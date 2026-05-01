package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

type AccountingRepository interface {
	// FindAccountByCode looks up a chart-of-accounts entry by its code (e.g. "1001").
	FindAccountByCode(ctx context.Context, code string) (*domain.Account, error)

	// CreateJournalEntry persists a journal entry and all its ledger lines atomically.
	// It generates the entry number internally using the JECounter sequence table.
	CreateJournalEntry(ctx context.Context, entry *domain.JournalEntry) error

	// SeedSystemAccounts inserts the standard chart-of-accounts rows if they don't
	// already exist. Safe to call repeatedly on startup (idempotent).
	SeedSystemAccounts(ctx context.Context) error
}

type accountingRepository struct{ db *gorm.DB }

func NewAccountingRepository(db *gorm.DB) AccountingRepository {
	return &accountingRepository{db: db}
}

func (r *accountingRepository) FindAccountByCode(ctx context.Context, code string) (*domain.Account, error) {
	var acc domain.Account
	err := r.db.WithContext(ctx).
		Where("code = ? AND is_active = ?", code, true).
		First(&acc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &acc, nil
}

func (r *accountingRepository) CreateJournalEntry(ctx context.Context, entry *domain.JournalEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Generate a monotonic entry number using the JECounter auto-increment table.
		counter := domain.JECounter{}
		if err := tx.Create(&counter).Error; err != nil {
			return fmt.Errorf("je counter: %w", err)
		}
		entry.EntryNumber = fmt.Sprintf("JE-%s-%07d", time.Now().UTC().Format("200601"), counter.ID)

		// GORM cascades into ledger_entries via the Lines association.
		return tx.Create(entry).Error
	})
}

func (r *accountingRepository) SeedSystemAccounts(ctx context.Context) error {
	seeds := []domain.Account{
		{Code: domain.AcctReceivable,   Name: "Accounts Receivable", Type: domain.AccountAsset,     IsSystem: true},
		{Code: domain.AcctCashBank,     Name: "Cash & Bank",          Type: domain.AccountAsset,     IsSystem: true},
		{Code: domain.AcctInventory,    Name: "Inventory",            Type: domain.AccountAsset,     IsSystem: true},
		{Code: domain.AcctPayable,      Name: "Accounts Payable",     Type: domain.AccountLiability, IsSystem: true},
		{Code: domain.AcctCGSTPayable,  Name: "CGST Payable",         Type: domain.AccountLiability, IsSystem: true},
		{Code: domain.AcctSGSTPayable,  Name: "SGST Payable",         Type: domain.AccountLiability, IsSystem: true},
		{Code: domain.AcctIGSTPayable,  Name: "IGST Payable",         Type: domain.AccountLiability, IsSystem: true},
		{Code: domain.AcctEquity,       Name: "Owner's Equity",       Type: domain.AccountEquity,    IsSystem: true},
		{Code: domain.AcctSalesRevenue, Name: "Sales Revenue",        Type: domain.AccountRevenue,   IsSystem: true},
		{Code: domain.AcctCOGS,         Name: "Cost of Goods Sold",   Type: domain.AccountExpense,   IsSystem: true},
		{Code: domain.AcctPurchase,     Name: "Purchase Expense",     Type: domain.AccountExpense,   IsSystem: true},
	}

	for i := range seeds {
		var existing domain.Account
		err := r.db.WithContext(ctx).Where("code = ?", seeds[i].Code).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := r.db.WithContext(ctx).Create(&seeds[i]).Error; err != nil {
				return fmt.Errorf("seed account %s: %w", seeds[i].Code, err)
			}
		}
	}
	return nil
}
