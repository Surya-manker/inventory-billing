package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type CreditNoteHandler struct {
	svc service.CreditNoteService
}

func NewCreditNoteHandler(svc service.CreditNoteService) *CreditNoteHandler {
	return &CreditNoteHandler{svc: svc}
}

// Issue godoc
// POST /api/v1/invoices/:id/credit-notes  (admin)
func (h *CreditNoteHandler) Issue(c *gin.Context) {
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
		Items []struct {
			ProductID string `json:"product_id" binding:"required"`
			Quantity  int    `json:"quantity"   binding:"required,gt=0"`
		} `json:"items" binding:"required,min=1"`
		Reason string `json:"reason" binding:"required"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	items := make([]service.CNItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		pid, err := uuid.Parse(it.ProductID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid product_id: "+it.ProductID)
			return
		}
		items = append(items, service.CNItemInput{ProductID: pid, Quantity: it.Quantity})
	}

	cn, err := h.svc.Issue(c.Request.Context(), service.IssueCreditNoteInput{
		InvoiceID: invoiceID,
		IssuedBy:  actorID,
		Items:     items,
		Reason:    req.Reason,
		Notes:     req.Notes,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "invoice not found")
		case errors.Is(err, domain.ErrCreditNoteInvoiceNotPaid):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrCreditNoteProductNotInInv):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrCreditNoteQuantityExceeds):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not issue credit note")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "credit note issued", cn)
}

// GetByID godoc
// GET /api/v1/credit-notes/:id
func (h *CreditNoteHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid credit note id")
		return
	}
	cn, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "credit note not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch credit note")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", cn)
}

// Void godoc
// PATCH /api/v1/credit-notes/:id/void  (admin)
func (h *CreditNoteHandler) Void(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid credit note id")
		return
	}
	actorID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	cn, err := h.svc.Void(c.Request.Context(), id, actorID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "credit note not found")
		case errors.Is(err, domain.ErrCreditNoteAlreadyVoided):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not void credit note")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "credit note voided", cn)
}

// ListByInvoice godoc
// GET /api/v1/invoices/:id/credit-notes
func (h *CreditNoteHandler) ListByInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid invoice id")
		return
	}
	cns, err := h.svc.ListByInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch credit notes")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", cns)
}

// ListByCustomer godoc
// GET /api/v1/customers/:id/credit-notes
func (h *CreditNoteHandler) ListByCustomer(c *gin.Context) {
	limit, offset := utils.Pagination(c)
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || n == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid customer id")
		return
	}
	cns, total, err := h.svc.ListByCustomer(c.Request.Context(), uint(n), limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch credit notes")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", cns, total, limit, offset)
}
