package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type StockHandler struct {
	svc service.StockService
}

func NewStockHandler(svc service.StockService) *StockHandler {
	return &StockHandler{svc: svc}
}

// GetAlerts godoc
// GET /api/v1/stock/alerts
//
// Returns every product whose current stock is at or below its
// individual low_stock_threshold. No pagination — the list is
// expected to be small enough for a single response.
func (h *StockHandler) GetAlerts(c *gin.Context) {
	products, err := h.svc.GetAlerts(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch stock alerts")
		return
	}

	type alertItem struct {
		ID                uuid.UUID `json:"id"`
		Name              string    `json:"name"`
		SKU               string    `json:"sku"`
		Stock             int       `json:"stock"`
		LowStockThreshold int       `json:"low_stock_threshold"`
		Deficit           int       `json:"deficit"` // threshold - stock (how many to reorder)
	}

	items := make([]alertItem, 0, len(products))
	for _, p := range products {
		items = append(items, alertItem{
			ID:                p.ID,
			Name:              p.Name,
			SKU:               p.SKU,
			Stock:             p.Stock,
			LowStockThreshold: p.LowStockThreshold,
			Deficit:           p.LowStockThreshold - p.Stock,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "low stock alerts", gin.H{
		"count": len(items),
		"items": items,
	})
}

// GetLogs godoc
// GET /api/v1/stock/logs
//
// Query params:
//
//	product_id   uuid
//	change_type  sale | purchase | adjustment | return
//	from_date    RFC3339
//	to_date      RFC3339
//	limit        int (max 100, default 20)
//	offset       int
func (h *StockHandler) GetLogs(c *gin.Context) {
	limit, offset := utils.Pagination(c)

	filter := domain.StockLogFilter{
		ChangeType: c.Query("change_type"),
	}
	if v := c.Query("product_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ProductID = &id
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

	logs, total, err := h.svc.GetLogs(c.Request.Context(), filter, limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch stock logs")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", logs, total, limit, offset)
}

// GetProductLogs godoc
// GET /api/v1/stock/logs/product/:id
func (h *StockHandler) GetProductLogs(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid product id")
		return
	}

	limit, offset := utils.Pagination(c)
	logs, total, err := h.svc.GetProductLogs(c.Request.Context(), productID, limit, offset)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch product stock logs")
		return
	}
	utils.PaginatedResponse(c, http.StatusOK, "success", logs, total, limit, offset)
}
