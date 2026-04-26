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

	ErrCustomerEmailTaken        = errors.New("a customer with this email already exists")
	ErrInvalidStatusTransition   = errors.New("invalid invoice status transition")
	ErrCannotDeletePaidInvoice   = errors.New("paid invoices cannot be deleted")
	ErrDiscountExceedsSubtotal   = errors.New("discount cannot exceed subtotal")
)
