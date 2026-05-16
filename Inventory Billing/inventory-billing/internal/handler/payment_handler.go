package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type PaymentHandler struct {
	svc service.PaymentService
}

func NewPaymentHandler(svc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// Record godoc
// POST /api/v1/invoices/:id/payments
func (h *PaymentHandler) Record(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid invoice id")
		return
	}
	actorID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Amount      float64               `json:"amount"       binding:"required,gt=0"`
		Method      domain.PaymentMethod  `json:"method"       binding:"required"`
		ReferenceNo string                `json:"reference_no"`
		Notes       string                `json:"notes"`
		PaidAt      string                `json:"paid_at"` // optional RFC3339; defaults to now
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	paidAt := time.Now().UTC()
	if req.PaidAt != "" {
		if t, err := time.Parse(time.RFC3339, req.PaidAt); err == nil {
			paidAt = t
		}
	}

	payment, err := h.svc.Record(c.Request.Context(), service.RecordPaymentInput{
		InvoiceID:   invoiceID,
		RecordedBy:  actorID,
		Amount:      req.Amount,
		Method:      req.Method,
		ReferenceNo: req.ReferenceNo,
		Notes:       req.Notes,
		PaidAt:      paidAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
		case errors.Is(err, domain.ErrPaymentExceedsBalance):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrInvoiceAlreadyPaid):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrCannotPayCanceledInvoice):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not record payment")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "payment recorded", payment)
}

// ListByInvoice godoc
// GET /api/v1/invoices/:id/payments
func (h *PaymentHandler) ListByInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid invoice id")
		return
	}
	payments, summary, err := h.svc.ListByInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch payments")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", gin.H{
		"payments": payments,
		"summary":  summary,
	})
}

// Delete godoc
// DELETE /api/v1/payments/:id  (admin)
func (h *PaymentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid payment id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "payment not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete payment")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "payment deleted", nil)
}
