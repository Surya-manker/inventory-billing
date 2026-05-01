package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountType classifies a chart-of-accounts entry.
type AccountType string

const (
	AccountAsset     AccountType = "asset"
	AccountLiability AccountType = "liability"
	AccountEquity    AccountType = "equity"
	AccountRevenue   AccountType = "revenue"
	AccountExpense   AccountType = "expense"
)

// System account codes — referenced by the accounting service to post entries.
const (
	AcctReceivable   = "1001" // Accounts Receivable (Asset)
	AcctCashBank     = "1002" // Cash & Bank (Asset)
	AcctInventory    = "1003" // Inventory (Asset)
	AcctPayable      = "2001" // Accounts Payable (Liability)
	AcctCGSTPayable  = "2002" // CGST Payable (Liability)
	AcctSGSTPayable  = "2003" // SGST Payable (Liability)
	AcctIGSTPayable  = "2004" // IGST Payable (Liability)
	AcctEquity       = "3001" // Owner's Equity (Equity)
	AcctSalesRevenue = "4001" // Sales Revenue (Revenue)
	AcctCOGS         = "5001" // Cost of Goods Sold (Expense)
	AcctPurchase     = "5002" // Purchase Expense (Expense)
)

// Account is one entry in the chart of accounts.
// System accounts (IsSystem=true) are seeded at startup and cannot be deleted.
type Account struct {
	ID       uuid.UUID   `gorm:"type:varchar(36);primaryKey"            json:"id"`
	Code     string      `gorm:"type:varchar(20);uniqueIndex;not null"   json:"code"`
	Name     string      `gorm:"type:varchar(100);not null"              json:"name"`
	Type     AccountType `gorm:"type:varchar(20);not null;index"         json:"type"`
	ParentID *uuid.UUID  `gorm:"type:varchar(36);index"                  json:"parent_id,omitempty"`
	IsSystem bool        `gorm:"not null;default:false"                  json:"is_system"`
	IsActive bool        `gorm:"not null;default:true"                   json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// JournalEntry is the voucher header for a double-entry bookkeeping transaction.
// Every financial event (invoice, payment, credit note) produces one journal entry.
type JournalEntry struct {
	ID          uuid.UUID     `gorm:"type:varchar(36);primaryKey"                      json:"id"`
	EntryNumber string        `gorm:"type:varchar(50);uniqueIndex;not null"            json:"entry_number"` // JE-202401-0000001
	Date        time.Time     `gorm:"index;not null"                                   json:"date"`
	Description string        `gorm:"type:varchar(255);not null"                       json:"description"`
	RefType     string        `gorm:"type:varchar(30);index:idx_je_ref"                json:"ref_type"` // "invoice"|"payment"|"credit_note"
	RefID       string        `gorm:"type:varchar(36);index:idx_je_ref"                json:"ref_id"`
	CreatedBy   uuid.UUID     `gorm:"type:varchar(36);not null"                        json:"created_by"`
	IsReversed  bool          `gorm:"not null;default:false"                           json:"is_reversed"`
	ReversedBy  *uuid.UUID    `gorm:"type:varchar(36)"                                 json:"reversed_by,omitempty"`
	Lines       []LedgerEntry `gorm:"foreignKey:JournalID"                             json:"lines,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

func (j *JournalEntry) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// LedgerEntry is a single debit or credit line within a journal entry.
// Constraint: sum of all Debit lines == sum of all Credit lines for a given JournalID.
type LedgerEntry struct {
	ID          uuid.UUID `gorm:"type:varchar(36);primaryKey"                          json:"id"`
	JournalID   uuid.UUID `gorm:"type:varchar(36);not null;index"                      json:"journal_id"`
	AccountID   uuid.UUID `gorm:"type:varchar(36);not null;index:idx_le_account_date"  json:"account_id"`
	Account     Account   `gorm:"foreignKey:AccountID"                                 json:"account,omitempty"`
	Debit       float64   `gorm:"type:decimal(15,2);not null;default:0"                json:"debit"`
	Credit      float64   `gorm:"type:decimal(15,2);not null;default:0"                json:"credit"`
	Description string    `gorm:"type:varchar(255)"                                    json:"description"`
	CreatedAt   time.Time `gorm:"index:idx_le_account_date"                            json:"created_at"`
}

func (l *LedgerEntry) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// JECounter is an append-only auto-increment table used to generate monotonic
// journal entry numbers without a DB sequence (MySQL InnoDB compatible).
type JECounter struct {
	ID uint `gorm:"primaryKey;autoIncrement"`
}
