package domain

import "errors"

var (
	ErrNotFound           = errors.New("record not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrForbidden          = errors.New("insufficient permissions")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")

	ErrSKUTaken          = errors.New("SKU already exists")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrProductInUse      = errors.New("product is referenced by existing invoices and cannot be deleted")

	ErrCustomerEmailTaken      = errors.New("a customer with this email already exists")
	ErrInvalidStatusTransition = errors.New("invalid invoice status transition")
	ErrCannotDeletePaidInvoice = errors.New("paid invoices cannot be deleted")
	ErrDiscountExceedsSubtotal = errors.New("discount cannot exceed subtotal")

	// vendor / purchase order
	ErrVendorEmailTaken         = errors.New("a vendor with this email already exists")
	ErrVendorInactive           = errors.New("vendor is inactive")
	ErrInvalidPOTransition      = errors.New("invalid purchase order status transition")
	ErrPOAlreadyReceived        = errors.New("purchase order has already been received")
	ErrCannotDeleteConfirmedPO  = errors.New("confirmed or received purchase orders cannot be deleted")

	// payments
	ErrPaymentExceedsBalance    = errors.New("payment amount exceeds outstanding invoice balance")
	ErrInvoiceAlreadyPaid       = errors.New("invoice is already fully paid")
	ErrCannotPayCanceledInvoice = errors.New("cannot record payment against a canceled invoice")

	// credit notes
	ErrCreditNoteInvoiceNotPaid   = errors.New("credit notes can only be issued against paid invoices")
	ErrCreditNoteQuantityExceeds  = errors.New("return quantity exceeds original invoice quantity for one or more items")
	ErrCreditNoteAlreadyVoided    = errors.New("credit note is already voided")
	ErrCreditNoteProductNotInInv  = errors.New("one or more products were not in the original invoice")

	// categories
	ErrCategoryNameTaken  = errors.New("a category with this name already exists")
	ErrCategoryHasProducts = errors.New("category has products assigned and cannot be deleted")
)
