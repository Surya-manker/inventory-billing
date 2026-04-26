package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"gorm.io/gorm"
)

// JobHandler exposes job status via a generic /jobs/:id endpoint.
// Clients receive a job_id when they trigger an async operation (e.g. invoice
// creation) and can poll this endpoint to check progress.
type JobHandler struct {
	repo repository.JobRepository
}

func NewJobHandler(repo repository.JobRepository) *JobHandler {
	return &JobHandler{repo: repo}
}

// GetByID returns the current status of a background job.
//
// GET /api/v1/jobs/:id
//
// 200 — job found (check .status for "pending"|"processing"|"completed"|"failed"|"dead")
// 404 — job ID not found
func (h *JobHandler) GetByID(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "job id is required")
		return
	}

	job, err := h.repo.FindByID(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "job not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch job")
		return
	}

	httpStatus := http.StatusOK
	if job.Status == domain.JobStatusPending || job.Status == domain.JobStatusProcessing {
		httpStatus = http.StatusAccepted // 202 — not done yet
	}

	utils.SuccessResponse(c, httpStatus, "job status", toJobResponse(job))
}

func toJobResponse(j *domain.JobRecord) gin.H {
	res := gin.H{
		"id":          j.ID,
		"type":        j.Type,
		"status":      j.Status,
		"entity_type": j.EntityType,
		"entity_id":   j.EntityID,
		"retry_count": j.RetryCount,
		"max_retries": j.MaxRetries,
		"enqueued_at": j.EnqueuedAt,
		"started_at":  j.StartedAt,
		"completed_at": j.CompletedAt,
	}
	if j.ErrorMsg != "" {
		res["error"] = j.ErrorMsg
	}
	if j.Result != "" {
		var raw json.RawMessage = json.RawMessage(j.Result)
		res["result"] = &raw
	}
	return res
}

// jsonUnmarshalStr is a convenience helper used by invoice_handler.go.
func jsonUnmarshalStr(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
