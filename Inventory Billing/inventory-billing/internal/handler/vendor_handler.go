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

// ── Vendor handler ────────────────────────────────────────────────────────────

type VendorHandler struct {
	svc service.VendorService
}

func NewVendorHandler(svc service.VendorService) *VendorHandler {
	return &VendorHandler{svc: svc}
}

func (h *VendorHandler) Create(c *gin.Context) {
	var req struct {
		Name             string `json:"name"              binding:"required"`
		Email            string `json:"email"             binding:"required,email"`
		Phone            string `json:"phone"`
		Address          string `json:"address"`
		Company          string `json:"company"`
		GSTIN            string `json:"gstin"`
		ContactPerson    string `json:"contact_person"`
		PaymentTermsDays int    `json:"payment_terms_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	vendor, err := h.svc.Create(c.Request.Context(), service.CreateVendorInput{
		Name: req.Name, Email: req.Email, Phone: req.Phone,
		Address: req.Address, Company: req.Company, GSTIN: req.GSTIN,
		ContactPerson:    req.ContactPerson,
		PaymentTermsDays: req.PaymentTermsDays,
	})
	if err != nil {
		if errors.Is(err, domain.ErrVendorEmailTaken) {
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not create vendor")
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "vendor created", vendor)
}

func (h *VendorHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid vendor id")
		return
	}
	vendor, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "vendor not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch vendor")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", vendor)
}

func (h *VendorHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid vendor id")
		return
	}
	var req struct {
		Name             string `json:"name"              binding:"required"`
		Email            string `json:"email"             binding:"required,email"`
		Phone            string `json:"phone"`
		Address          string `json:"address"`
		Company          string `json:"company"`
		GSTIN            string `json:"gstin"`
		ContactPerson    string `json:"contact_person"`
		PaymentTermsDays int    `json:"payment_terms_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	vendor, err := h.svc.Update(c.Request.Context(), id, service.CreateVendorInput{
		Name: req.Name, Email: req.Email, Phone: req.Phone,
		Address: req.Address, Company: req.Company, GSTIN: req.GSTIN,
		ContactPerson:    req.ContactPerson,
		PaymentTermsDays: req.PaymentTermsDays,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "vendor not found")
		case errors.Is(err, domain.ErrVendorEmailTaken):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update vendor")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "vendor updated", vendor)
}

func (h *VendorHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid vendor id")
		return
	}
	if err := h.svc.Deactivate(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "vendor not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not deactivate vendor")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "vendor deactivated", nil)
}

func (h *VendorHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid vendor id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "vendor not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete vendor")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "vendor deleted", nil)
}

func (h *VendorHandler) List(c *gin.Context) {
	limit, offset := utils.Pagination(c)
	filter := domain.VendorFilter{Search: c.Query("search")}
	if c.Query("active") == "true" {
		t := true
		filter.IsActive = &t
	} else if c.Query("active") == "false" {
		f := false
		filter.IsActive = &f
	}
	vendors, total, err := h.svc.List(c.Request.Context(), filter, limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not list vendors")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", vendors, total, limit, offset)
}

// ── Purchase Order handler ────────────────────────────────────────────────────

type PurchaseHandler struct {
	svc service.PurchaseService
}

func NewPurchaseHandler(svc service.PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{svc: svc}
}

func (h *PurchaseHandler) Create(c *gin.Context) {
	var req struct {
		VendorID string `json:"vendor_id" binding:"required"`
		Items    []struct {
			ProductID string  `json:"product_id" binding:"required"`
			Quantity  int     `json:"quantity"   binding:"required,gt=0"`
			UnitCost  float64 `json:"unit_cost"  binding:"required,gt=0"`
		} `json:"items" binding:"required,min=1"`
		Notes      string `json:"notes"`
		ExpectedAt string `json:"expected_at" binding:"required"` // RFC3339
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	vendorID, err := uuid.Parse(req.VendorID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid vendor_id")
		return
	}
	expectedAt, err := time.Parse(time.RFC3339, req.ExpectedAt)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "expected_at must be RFC3339")
		return
	}
	actorID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	items := make([]service.POItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		pid, err := uuid.Parse(it.ProductID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid product_id: "+it.ProductID)
			return
		}
		items = append(items, service.POItemInput{ProductID: pid, Quantity: it.Quantity, UnitCost: it.UnitCost})
	}

	po, err := h.svc.Create(c.Request.Context(), service.CreatePOInput{
		VendorID: vendorID, CreatedBy: actorID, Items: items,
		Notes: req.Notes, ExpectedAt: expectedAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "vendor not found")
		case errors.Is(err, domain.ErrVendorInactive):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not create purchase order")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "purchase order created", po)
}

func (h *PurchaseHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}
	po, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "purchase order not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch purchase order")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", po)
}

func (h *PurchaseHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Status domain.POStatus `json:"status" binding:"required"`
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

	var (
		po    *domain.PurchaseOrder
		svcErr error
	)
	switch req.Status {
	case domain.POConfirmed:
		po, svcErr = h.svc.Confirm(c.Request.Context(), id)
	case domain.POReceived:
		po, svcErr = h.svc.Receive(c.Request.Context(), id, actorID)
	case domain.POCanceled:
		po, svcErr = h.svc.Cancel(c.Request.Context(), id)
	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status — use: confirmed | received | canceled")
		return
	}

	if svcErr != nil {
		switch {
		case errors.Is(svcErr, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "purchase order not found")
		case errors.Is(svcErr, domain.ErrInvalidPOTransition):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, svcErr.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update purchase order")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "status updated", po)
}

func (h *PurchaseHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "purchase order not found")
		case errors.Is(err, domain.ErrCannotDeleteConfirmedPO):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete purchase order")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "purchase order deleted", nil)
}

func (h *PurchaseHandler) List(c *gin.Context) {
	limit, offset := utils.Pagination(c)
	var vendorID *uuid.UUID
	if v := c.Query("vendor_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			vendorID = &id
		}
	}
	pos, total, err := h.svc.List(c.Request.Context(), vendorID, c.Query("status"), limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not list purchase orders")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", pos, total, limit, offset)
}
