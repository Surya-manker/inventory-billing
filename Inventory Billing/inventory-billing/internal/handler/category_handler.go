package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type CategoryHandler struct {
	svc service.CategoryService
}

func NewCategoryHandler(svc service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req struct {
		Name        string  `json:"name"        binding:"required"`
		Description string  `json:"description"`
		ParentID    *string `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid parent_id")
			return
		}
		parentID = &id
	}

	cat, err := h.svc.Create(c.Request.Context(), service.CreateCategoryInput{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    parentID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCategoryNameTaken) {
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not create category")
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "category created", cat)
}

func (h *CategoryHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid category id")
		return
	}
	cat, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "category not found")
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not fetch category")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", cat)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid category id")
		return
	}
	var req struct {
		Name        string  `json:"name"        binding:"required"`
		Description string  `json:"description"`
		ParentID    *string `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	var parentID *uuid.UUID
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid parent_id")
			return
		}
		parentID = &pid
	}
	cat, err := h.svc.Update(c.Request.Context(), id, service.CreateCategoryInput{
		Name: req.Name, Description: req.Description, ParentID: parentID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "category not found")
		case errors.Is(err, domain.ErrCategoryNameTaken):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update category")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "category updated", cat)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid category id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			utils.ErrorResponse(c, http.StatusNotFound, "category not found")
		case errors.Is(err, domain.ErrCategoryHasProducts):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not delete category")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "category deleted", nil)
}

func (h *CategoryHandler) List(c *gin.Context) {
	cats, err := h.svc.List(c.Request.Context())
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "could not list categories")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", cats)
}
