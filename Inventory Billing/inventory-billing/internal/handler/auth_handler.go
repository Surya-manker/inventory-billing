package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/utils"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Name     string      `json:"name"     binding:"required"`
		Email    string      `json:"email"    binding:"required,email"`
		Password string      `json:"password" binding:"required,min=8"`
		Role     domain.Role `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Role == "" {
		req.Role = domain.RoleStaff
	}

	user, err := h.svc.Register(c.Request.Context(), req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailTaken):
			utils.ErrorResponse(c, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrWeakPassword):
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "registration failed")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "registered successfully", user)
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	user, tokens, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// always return the same message regardless of whether email or password is wrong
		utils.ErrorResponse(c, http.StatusUnauthorized, domain.ErrInvalidCredentials.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "login successful", gin.H{
		"user":   user,
		"tokens": tokens,
	})
}

// Refresh godoc
// POST /api/v1/auth/refresh
// Accepts a refresh token, invalidates it, and returns a new token pair (token rotation).
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, domain.ErrInvalidToken.Error())
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "token refreshed", tokens)
}

// Logout godoc
// POST /api/v1/auth/logout  (protected)
// Invalidates the refresh token. The access token expires on its own schedule.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// ignore error — if token didn't exist, the user is effectively already logged out
	_ = h.svc.Logout(c.Request.Context(), req.RefreshToken)
	utils.SuccessResponse(c, http.StatusOK, "logged out successfully", nil)
}

// Me godoc
// GET /api/v1/auth/me  (protected)
func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "user not found")
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "success", user)
}

// ChangePassword godoc
// PUT /api/v1/auth/change-password  (protected)
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, err := utils.UserIDFromCtx(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			utils.ErrorResponse(c, http.StatusUnauthorized, "current password is incorrect")
		case errors.Is(err, domain.ErrWeakPassword):
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(c, http.StatusInternalServerError, "could not update password")
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "password changed successfully", nil)
}
