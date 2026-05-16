package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type CustomerHandler struct {
	svc service.CustomerService
}

func NewCustomerHandler(svc service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func parseCustomerID(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || n == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid customer id")
		return 0, false
	}
	return uint(n), true
}

// Create godoc — POST /api/v1/customers  (admin)
func (h *CustomerHandler) Create(c *gin.Context) {
	var req struct {
		Name    string `json:"name"    binding:"required"`
		Email   string `json:"email"   binding:"required,email"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
		Company string `json:"company"`
		GSTIN   string `json:"gstin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	customer, err := h.svc.Create(c.Request.Context(), service.CreateCustomerInput{
		Name: req.Name, Email: req.Email, Phone: req.Phone,
		Address: req.Address, Company: req.Company, GSTIN: req.GSTIN,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCustomerEmailTaken) {
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not create customer")
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "customer created", customer)
}

// GetByID godoc — GET /api/v1/customers/:id
func (h *CustomerHandler) GetByID(c *gin.Context) {
	id, ok := parseCustomerID(c)
	if !ok {
		return
	}

	customer, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "customer not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch customer")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", customer)
}

// Update godoc — PUT /api/v1/customers/:id  (admin)
func (h *CustomerHandler) Update(c *gin.Context) {
	id, ok := parseCustomerID(c)
	if !ok {
		return
	}

	var req struct {
		Name    string `json:"name"    binding:"required"`
		Email   string `json:"email"   binding:"required,email"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
		Company string `json:"company"`
		GSTIN   string `json:"gstin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	customer, err := h.svc.Update(c.Request.Context(), id, service.UpdateCustomerInput{
		Name: req.Name, Email: req.Email, Phone: req.Phone,
		Address: req.Address, Company: req.Company, GSTIN: req.GSTIN,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "customer not found")
		case errors.Is(err, domain.ErrCustomerEmailTaken):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update customer")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "customer updated", customer)
}

// Delete godoc — DELETE /api/v1/customers/:id  (admin)
func (h *CustomerHandler) Delete(c *gin.Context) {
	id, ok := parseCustomerID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "customer not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete customer")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "customer deleted", nil)
}

// List godoc — GET /api/v1/customers?search=&limit=&offset=
func (h *CustomerHandler) List(c *gin.Context) {
	limit, offset := utils.Pagination(c)
	customers, total, err := h.svc.List(c.Request.Context(), c.Query("search"), limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not list customers")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", customers, total, limit, offset)
}
