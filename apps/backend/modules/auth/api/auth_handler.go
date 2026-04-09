package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/arc-platform/backend/modules/auth/entity"
	"github.com/arc-platform/backend/modules/auth/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	userService *service.UserService
	jwtService  *service.JWTService
	repo        *persistence.PostgresRepository
	auditLogger interfaces.AuditLogger
}

func NewAuthHandler(repo *persistence.PostgresRepository, db *sql.DB, auditLogger interfaces.AuditLogger) *AuthHandler {
	return &AuthHandler{
		userService: service.NewUserService(repo, db),
		jwtService:  service.NewJWTService(db),
		repo:        repo,
		auditLogger: auditLogger,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	TenantID string `json:"tenant_id" binding:"required"`
}

type LoginResponse struct {
	User         *entity.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	TokenType    string       `json:"token_type"`
}

type RegisterRequest struct {
	TenantName string `json:"tenant_name" binding:"required,min=3,max=100"`
	TenantSlug string `json:"tenant_slug" binding:"required,alpha,lowercase,min=3,max=50"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	if _, err := uuid.Parse(req.TenantID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid tenant_id format",
		})
		return
	}

	user, accessToken, refreshToken, err := h.userService.Authenticate(c.Request.Context(), req.Email, req.Password, req.TenantID)
	if err != nil {
		if h.auditLogger != nil {
			_ = h.auditLogger.Record(c.Request.Context(), "LOGIN_FAILED", "user", req.Email, map[string]interface{}{
				"ip":        c.ClientIP(),
				"tenant_id": req.TenantID,
				"reason":    err.Error(),
			})
		}
		status := http.StatusUnauthorized
		message := "Invalid credentials"
		if err == service.ErrUserInactive {
			message = "User account is inactive"
		}
		c.JSON(status, ErrorResponse{
			Error:   "authentication_error",
			Message: message,
		})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "LOGIN_SUCCESS", "user", user.ID.String(), map[string]interface{}{
			"ip":        c.ClientIP(),
			"tenant_id": req.TenantID,
			"email":     req.Email,
		})
	}

	c.JSON(http.StatusOK, LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
		TokenType:    "Bearer",
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	tenant := &entity.Tenant{
		ID:          uuid.New(),
		Name:        req.TenantName,
		Slug:        req.TenantSlug,
		Description: "Organization created during registration",
		IsActive:    true,
	}

	if err := h.repo.CreateTenant(c.Request.Context(), tenant); err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "tenant_exists",
			Message: "Tenant with this slug already exists",
		})
		return
	}

	user, err := h.userService.CreateUser(
		c.Request.Context(),
		tenant.ID,
		req.Email,
		req.Password,
		req.FirstName,
		req.LastName,
		entity.RoleAdmin,
	)
	if err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "user_exists",
			Message: "User with this email already exists",
		})
		return
	}

	accessToken, refreshToken, err := h.jwtService.GenerateToken(user, uuid.New())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate tokens",
		})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "USER_REGISTERED", "user", user.ID.String(), map[string]interface{}{
			"ip":          c.ClientIP(),
			"tenant_id":   tenant.ID.String(),
			"tenant_slug": req.TenantSlug,
			"email":       req.Email,
		})
	}

	c.JSON(http.StatusCreated, LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
		TokenType:    "Bearer",
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	claims, err := h.jwtService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid or expired refresh token",
		})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid user ID in token",
		})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found or inactive",
		})
		return
	}

	accessToken, refreshToken, err := h.jwtService.GenerateToken(user, uuid.New())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate tokens",
		})
		return
	}

	c.JSON(http.StatusOK, RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    86400,
		TokenType:    "Bearer",
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	err := h.userService.ChangePassword(c.Request.Context(), userID.(uuid.UUID), req.CurrentPassword, req.NewPassword)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Failed to change password"
		if err == service.ErrInvalidPassword {
			status = http.StatusUnauthorized
			message = "Current password is incorrect"
		}
		c.JSON(status, ErrorResponse{
			Error:   "password_error",
			Message: message,
		})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "PASSWORD_CHANGED", "user", userID.(uuid.UUID).String(), map[string]interface{}{
			"ip": c.ClientIP(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant not found",
		})
		return
	}

	users, err := h.userService.GetUsersByTenant(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

// SettingsRequest struct for update payload
type SettingsRequest struct {
	Settings map[string]interface{} `json:"settings" binding:"required"`
}

func (h *AuthHandler) GetSettings(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant not found",
		})
		return
	}

	tenant, err := h.repo.GetTenantByID(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Tenant not found",
		})
		return
	}

	// Just return the raw JSON string if it's there
	// frontend expects a JSON object though, so let's verify if we need to marshal/unmarshal
	// Entity definition says Settings is string (text/jsonb).
	// Let's assume it is a JSON string.
	// We should return it as an object.

	// If empty, return empty object
	if tenant.Settings == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	// It's a string in DB, but we want to return JSON
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, tenant.Settings)
}

func (h *AuthHandler) UpdateSettings(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant not found",
		})
		return
	}

	// We bind raw body mostly because we want to store it as is, or validation?
	// The struct uses `map[string]interface{}` which is good for flexible JSON
	var req SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	tenant, err := h.repo.GetTenantByID(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Tenant not found",
		})
		return
	}

	settingsJSON, err := json.Marshal(req.Settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "json_error",
			Message: "Failed to marshal settings",
		})
		return
	}

	tenant.Settings = string(settingsJSON)

	if err := h.repo.UpdateTenant(c.Request.Context(), tenant); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "update_error",
			Message: "Failed to update settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
	})
}
