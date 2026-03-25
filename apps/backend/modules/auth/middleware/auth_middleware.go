package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/arc-platform/backend/modules/auth/entity"
	"github.com/arc-platform/backend/modules/auth/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthMiddleware struct {
	jwtService    *service.JWTService
	userService   *service.UserService
	postgresRepo  *persistence.PostgresRepository
	skipAuthPaths map[string]bool
	publicPaths   map[string]bool
}

func NewAuthMiddleware(repo *persistence.PostgresRepository) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:   service.NewJWTService(),
		userService:  service.NewUserService(repo),
		postgresRepo: repo,
		skipAuthPaths: map[string]bool{
			"/health":               true,
			"/api/v1/auth/login":    true,
			"/api/v1/auth/register": true,
			"/api/v1/auth/refresh":  true,
			"/docs":                 true,
			"/swagger":              true,
		},
		publicPaths: map[string]bool{
			"/api/v1/auth/login":    true,
			"/api/v1/auth/register": true,
			"/api/v1/auth/refresh":  true,
			"/api/v1/health":        true,
		},
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if m.skipAuthPaths[path] {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// In dev mode (AUTH_REQUIRED != "true"), allow anonymous access
			// This mirrors the global authMiddleware behavior in main.go
			if os.Getenv("AUTH_REQUIRED") != "true" {
				// Set anonymous defaults so handlers don't crash
				c.Set("user_id", uuid.Nil)
				c.Set("user_email", "anonymous@dev.local")
				c.Set("user_role", "admin")
				c.Set("tenant_id", uuid.Nil)
				c.Set("authenticated", false)
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header required",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid authorization header format. Use: Bearer <token>",
			})
			c.Abort()
			return
		}

		claims, err := m.jwtService.ValidateToken(parts[1])
		if err != nil {
			message := "Invalid token"
			if err == service.ErrTokenExpired {
				message = "Token expired"
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": message,
			})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid user ID in token",
			})
			c.Abort()
			return
		}

		user, err := m.userService.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User not found or inactive",
			})
			c.Abort()
			return
		}

		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User account is inactive",
			})
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_role", claims.Role)
		ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
		ctx = context.WithValue(ctx, "session_id", claims.SessionID)

		c.Request = c.Request.WithContext(ctx)
		c.Set("user_id", userID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user", user)

		c.Next()
	}
}

func (m *AuthMiddleware) RequirePermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User not authenticated",
			})
			c.Abort()
			return
		}

		userEntity, ok := user.(*entity.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Invalid user object",
			})
			c.Abort()
			return
		}

		if !m.userService.HasPermission(userEntity, entity.Permission(requiredPermission)) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":    "forbidden",
				"message":  "Insufficient permissions",
				"required": requiredPermission,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User not authenticated",
			})
			c.Abort()
			return
		}

		userEntity, ok := user.(*entity.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Invalid user object",
			})
			c.Abort()
			return
		}

		for _, permission := range permissions {
			if m.userService.HasPermission(userEntity, entity.Permission(permission)) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":    "forbidden",
			"message":  "Insufficient permissions",
			"required": strings.Join(permissions, " or "),
		})
		c.Abort()
	}
}

func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User not authenticated",
			})
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, allowedRole := range roles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":    "forbidden",
			"message":  "Role not authorized for this action",
			"required": strings.Join(roles, " or "),
		})
		c.Abort()
	}
}

type UserContext struct {
	UserID    uuid.UUID
	Email     string
	Role      string
	TenantID  uuid.UUID
	SessionID string
}

func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		return nil, false
	}

	email, ok := ctx.Value("user_email").(string)
	if !ok {
		email = ""
	}

	role, ok := ctx.Value("user_role").(string)
	if !ok {
		role = ""
	}

	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok {
		tenantID = ""
	}

	sessionID, ok := ctx.Value("session_id").(string)
	if !ok {
		sessionID = ""
	}

	var tenantUUID uuid.UUID
	if tenantID != "" {
		tenantUUID, _ = uuid.Parse(tenantID)
	}

	return &UserContext{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TenantID:  tenantUUID,
		SessionID: sessionID,
	}, true
}

func GetUserFromGin(c *gin.Context) (*UserContext, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, false
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return nil, false
	}

	email, _ := c.Get("user_email")
	role, _ := c.Get("user_role")
	tenantID, _ := c.Get("tenant_id")

	var tenantUUID uuid.UUID
	if tid, ok := tenantID.(string); ok {
		tenantUUID, _ = uuid.Parse(tid)
	}

	return &UserContext{
		UserID:   uid,
		Email:    email.(string),
		Role:     role.(string),
		TenantID: tenantUUID,
	}, true
}
