package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TenantContextKey struct{}

type TenantContext struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Role     string
}

func WithTenantContext(ctx context.Context, tenantID, userID uuid.UUID, role string) context.Context {
	return context.WithValue(ctx, TenantContextKey{}, TenantContext{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
	})
}

func GetTenantContext(ctx context.Context) (TenantContext, bool) {
	val, ok := ctx.Value(TenantContextKey{}).(TenantContext)
	return val, ok
}

func TenantMiddleware(pgRepo *persistence.PostgresRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := c.GetHeader("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = c.Query("tenant_id")
		}

		if tenantIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header or tenant_id query parameter is required"})
			c.Abort()
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
			c.Abort()
			return
		}

		userIDStr := c.GetHeader("X-User-ID")
		var userID uuid.UUID
		if userIDStr != "" {
			userID, err = uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
				c.Abort()
				return
			}
		}

		role := c.GetHeader("X-User-Role")
		if role == "" {
			role = "viewer"
		}

		tenantCtx := TenantContext{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
		}

		ctx := WithTenantContext(c.Request.Context(), tenantID, userID, role)
		c.Request = c.Request.WithContext(ctx)

		c.Set("tenant_context", tenantCtx)
		c.Next()
	}
}

func RequireTenant(fallbackRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantCtx, exists := c.Get("tenant_context")
		if !exists {
			if fallbackRole != "" {
				c.Set("tenant_context", TenantContext{
					TenantID: uuid.Nil,
					UserID:   uuid.Nil,
					Role:     fallbackRole,
				})
				c.Next()
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant context not set"})
			c.Abort()
			return
		}

		ctx := tenantCtx.(TenantContext)
		if ctx.TenantID == uuid.Nil && fallbackRole == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetTenantIDFromToken(c *gin.Context) uuid.UUID {
	tenantCtx, exists := c.Get("tenant_context")
	if !exists {
		return uuid.Nil
	}
	return tenantCtx.(TenantContext).TenantID
}

func GetUserIDFromToken(c *gin.Context) uuid.UUID {
	tenantCtx, exists := c.Get("tenant_context")
	if !exists {
		return uuid.Nil
	}
	return tenantCtx.(TenantContext).UserID
}

func GetUserRoleFromToken(c *gin.Context) string {
	tenantCtx, exists := c.Get("tenant_context")
	if !exists {
		return ""
	}
	return tenantCtx.(TenantContext).Role
}

func ExtractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}
