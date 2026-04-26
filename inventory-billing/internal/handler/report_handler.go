package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type ReportHandler struct {
	svc service.ReportService
}

func NewReportHandler(svc service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// parseRange extracts ?from=&to= as time.Time (date-only, e.g. "2025-01-01").
// Defaults to the current calendar month when omitted.
func parseRange(c *gin.Context) (from, to time.Time) {
	now := time.Now().UTC()
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to = now

	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.DateOnly, v); err == nil {
			from = t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.DateOnly, v); err == nil {
			to = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}
	return
}

// Revenue godoc
// GET /api/v1/reports/revenue?from=2025-01-01&to=2025-01-31
func (h *ReportHandler) Revenue(c *gin.Context) {
	from, to := parseRange(c)
	data, err := h.svc.Revenue(c.Request.Context(), from, to)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", data)
}

// TopProducts godoc
// GET /api/v1/reports/top-products?from=&to=&limit=10
func (h *ReportHandler) TopProducts(c *gin.Context) {
	from, to := parseRange(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	data, err := h.svc.TopProducts(c.Request.Context(), from, to, limit)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", data)
}

// InvoiceSummary godoc
// GET /api/v1/reports/invoices?from=&to=
func (h *ReportHandler) InvoiceSummary(c *gin.Context) {
	from, to := parseRange(c)
	data, err := h.svc.InvoiceSummary(c.Request.Context(), from, to)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", data)
}

// InventoryValue godoc
// GET /api/v1/reports/inventory-value
func (h *ReportHandler) InventoryValue(c *gin.Context) {
	value, err := h.svc.InventoryValue(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not calculate inventory value")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", gin.H{"inventory_value": value})
}

// AgingReport godoc
// GET /api/v1/reports/aging
// Returns outstanding receivables bucketed by days overdue: current, 1-30, 31-60, 61-90, 90+.
// Only pending invoices are included.
func (h *ReportHandler) AgingReport(c *gin.Context) {
	rows, err := h.svc.AgingReport(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not generate aging report")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", rows)
}
