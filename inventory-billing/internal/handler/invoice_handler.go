package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type InvoiceHandler struct {
	svc service.InvoiceService
}

func NewInvoiceHandler(svc service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// Create godoc
// POST /api/v1/invoices  (any authenticated user)
//
// The created_by field is always taken from the JWT — it is never accepted
// in the request body to prevent privilege escalation.
func (h *InvoiceHandler) Create(c *gin.Context) {
	var req struct {
		CustomerID uint `json:"customer_id" binding:"required"`
		Items      []struct {
			ProductID string `json:"product_id" binding:"required"`
			Quantity  int    `json:"quantity"   binding:"required,gt=0"`
		} `json:"items" binding:"required,min=1"`
		Discount float64   `json:"discount"`
		Notes    string    `json:"notes"`
		DueAt    time.Time `json:"due_at" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	actorID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	items := make([]service.InvoiceItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		pid, err := uuid.Parse(item.ProductID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid product_id: "+item.ProductID)
			return
		}
		items = append(items, service.InvoiceItemInput{ProductID: pid, Quantity: item.Quantity})
	}

	invoice, err := h.svc.Create(c.Request.Context(), service.CreateInvoiceInput{
		CustomerID: req.CustomerID,
		CreatedBy:  actorID,
		Items:      items,
		Discount:   req.Discount,
		Notes:      req.Notes,
		DueAt:      req.DueAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrInsufficientStock):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrDiscountExceedsSubtotal):
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not create invoice")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "invoice created", invoice)
}

// GetByID godoc
// GET /api/v1/invoices/:id
func (h *InvoiceHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid invoice id")
		return
	}

	invoice, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch invoice")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", invoice)
}

// GetByNumber godoc
// GET /api/v1/invoices/number/:number
func (h *InvoiceHandler) GetByNumber(c *gin.Context) {
	invoice, err := h.svc.GetByNumber(c.Request.Context(), c.Param("number"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch invoice")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", invoice)
}

// UpdateStatus godoc
// PATCH /api/v1/invoices/:id/status  (admin)
//
// Allowed transitions:
//
//	pending → paid       (sets paid_at, no stock change)
//	pending → canceled   (restores stock atomically)
func (h *InvoiceHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid invoice id")
		return
	}

	var req struct {
		Status domain.InvoiceStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	actorID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	invoice, err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status, actorID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
		case errors.Is(err, domain.ErrInvalidStatusTransition):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update status")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "status updated", invoice)
}

// Delete godoc
// DELETE /api/v1/invoices/:id  (admin)
//
// Pending invoices: stock is restored before deletion.
// Canceled invoices: deleted without stock changes (already restored on cancel).
// Paid invoices: rejected — use cancellation instead.
func (h *InvoiceHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid invoice id")
		return
	}

	actorID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, actorID); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
		case errors.Is(err, domain.ErrCannotDeletePaidInvoice):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete invoice")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "invoice deleted", nil)
}

// List godoc
// GET /api/v1/invoices
//
// Query params:
//
//	customer_id   uint
//	status        pending | paid | canceled
//	from_date     RFC3339
//	to_date       RFC3339
//	sort_by       issued_at | due_at | total_price | created_at
//	sort_dir      asc | desc
//	limit         int (max 100, default 20)
//	offset        int
func (h *InvoiceHandler) List(c *gin.Context) {
	limit, offset := utils.Pagination(c)

	filter := domain.InvoiceFilter{
		Status:  c.Query("status"),
		SortBy:  c.Query("sort_by"),
		SortDir: c.Query("sort_dir"),
	}

	if v := c.Query("customer_id"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n > 0 {
			id := uint(n)
			filter.CustomerID = &id
		}
	}
	if v := c.Query("from_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.FromDate = &t
		}
	}
	if v := c.Query("to_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.ToDate = &t
		}
	}

	invoices, total, err := h.svc.List(c.Request.Context(), filter, limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not list invoices")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", invoices, total, limit, offset)
}
