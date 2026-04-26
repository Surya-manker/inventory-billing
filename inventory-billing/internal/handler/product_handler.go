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

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// Create godoc
// POST /api/v1/products  (admin)
func (h *ProductHandler) Create(c *gin.Context) {
	var req struct {
		Name        string  `json:"name"        binding:"required"`
		Description string  `json:"description"`
		SKU         string  `json:"sku"         binding:"required"`
		Price       float64 `json:"price"       binding:"required,gt=0"`
		Stock       int     `json:"stock"       binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.svc.Create(c.Request.Context(), service.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Price:       req.Price,
		Stock:       req.Stock,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSKUTaken):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not create product")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "product created", product)
}

// GetByID godoc
// GET /api/v1/products/:id
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "product not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch product")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", product)
}

// Update godoc
// PUT /api/v1/products/:id  (admin)
// SKU is immutable — only name, description, and price can be changed.
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid product id")
		return
	}

	var req struct {
		Name        string  `json:"name"        binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price"       binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.svc.Update(c.Request.Context(), id, service.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "product not found")
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update product")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "product updated", product)
}

// Delete godoc
// DELETE /api/v1/products/:id  (admin)
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "product not found")
		case errors.Is(err, domain.ErrProductInUse):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete product")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "product deleted", nil)
}

// List godoc
// GET /api/v1/products
//
// Query params:
//
//	search      string   — partial match on name or SKU
//	min_price   float64
//	max_price   float64
//	low_stock   int      — show only products with stock ≤ this value
//	sort_by     string   — name | price | stock | created_at  (default: created_at)
//	sort_dir    string   — asc | desc  (default: desc)
//	limit       int      — max 100  (default: 20)
//	offset      int      — (default: 0)
func (h *ProductHandler) List(c *gin.Context) {
	limit, offset := utils.Pagination(c)

	filter := domain.ProductFilter{
		Search:  c.Query("search"),
		SortBy:  c.Query("sort_by"),
		SortDir: c.Query("sort_dir"),
	}

	if v := c.Query("min_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			filter.MinPrice = f
		}
	}
	if v := c.Query("max_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			filter.MaxPrice = f
		}
	}
	if v := c.Query("low_stock"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			filter.LowStockMax = &n
		}
	}

	products, total, err := h.svc.List(c.Request.Context(), filter, limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not list products")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", products, total, limit, offset)
}

// AdjustStock godoc
// PATCH /api/v1/products/:id/stock  (admin)
//
// Body:
//
//	quantity_change  int     — signed: positive = stock in, negative = stock out
//	change_type      string  — purchase | adjustment | return
//	note             string  — optional reason
func (h *ProductHandler) AdjustStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid product id")
		return
	}

	var req struct {
		QuantityChange int                    `json:"quantity_change" binding:"required"`
		ChangeType     domain.StockChangeType `json:"change_type"     binding:"required"`
		Note           string                 `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	log, err := h.svc.AdjustStock(c.Request.Context(), domain.StockAdjustInput{
		ProductID:      id,
		UserID:         userID,
		ChangeType:     req.ChangeType,
		QuantityChange: req.QuantityChange,
		Note:           req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "product not found")
		case errors.Is(err, domain.ErrInsufficientStock):
			utils.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not adjust stock")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "stock adjusted", log)
}
