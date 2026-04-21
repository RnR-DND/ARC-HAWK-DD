package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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

func NewAuthMiddleware(repo *persistence.PostgresRepository, db *sql.DB) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:   service.NewJWTService(db),
		userService:  service.NewUserService(repo, db),
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

// authenticateAPIKey checks the X-API-Key header and validates it against the
// api_keys table (migration 000028). Returns true and sets gin context values
// if the key is valid, active, and not expired.
func (m *AuthMiddleware) authenticateAPIKey(c *gin.Context) bool {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		return false
	}

	// Hash the presented key
	h := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(h[:])

	var (
		keyID     uuid.UUID
		tenantID  uuid.UUID
		scopes    []string
		expiresAt sql.NullTime
		revoked   bool
	)

	err := m.postgresRepo.GetDB().QueryRowContext(c.Request.Context(), `
		SELECT id, tenant_id, scopes, expires_at, revoked
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash).Scan(&keyID, &tenantID, &scopes, &expiresAt, &revoked)
	if err != nil {
		return false
	}

	if revoked {
		return false
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return false
	}

	// Update last_used_at (fire-and-forget — don't block the request)
	go func() {
		if _, err := m.postgresRepo.GetDB().ExecContext(context.Background(), `
			UPDATE api_keys SET last_used_at = NOW() WHERE id = $1
		`, keyID); err != nil {
			log.Printf("WARN: failed to update api_keys.last_used_at for %s: %v", keyID, err)
		}
	}()

	c.Set("user_id", uuid.Nil)
	c.Set("user_email", "apikey@service.internal")
	c.Set("user_role", "service")
	c.Set("tenant_id", tenantID)
	c.Set("api_key_id", keyID)
	c.Set("api_key_scopes", scopes)
	c.Set("authenticated", true)
	return true
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if m.skipAuthPaths[path] {
			c.Next()
			return
		}

		// Check API key first — allows service-to-service calls without a JWT
		if m.authenticateAPIKey(c) {
			c.Next()
			return
		}

		// Check scanner service token — allows the go-scanner to call the
		// backend's ingest-verified / complete / progress-event endpoints
		// without a user JWT. The per-route ScannerCallbackAuth middleware
		// re-validates and scopes tenant context. This branch only exists so
		// the global middleware doesn't reject the request with 401 before
		// the per-route middleware runs.
		if token := c.GetHeader("X-Scanner-Token"); token != "" {
			if expected := os.Getenv("SCANNER_SERVICE_TOKEN"); expected != "" && token == expected {
				c.Set("service_auth", "scanner")
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// In dev mode (AUTH_REQUIRED != "true"), allow anonymous access
			// This mirrors the global authMiddleware behavior in main.go
			if os.Getenv("AUTH_REQUIRED") == "false" {
				// Honor X-Tenant-ID when present (used by scanner→backend
				// machine-to-machine calls in dev). Falls back to the
				// dedicated DevSystemTenantID so EnsureTenantID is happy.
				tenantID := persistence.DevSystemTenantID
				if hdr := c.GetHeader("X-Tenant-ID"); hdr != "" {
					if parsed, err := uuid.Parse(hdr); err == nil && parsed != uuid.Nil {
						tenantID = parsed
					}
				}
				ctx := context.WithValue(c.Request.Context(), "tenant_id", tenantID)
				c.Request = c.Request.WithContext(ctx)
				c.Set("user_id", uuid.Nil)
				c.Set("user_email", "anonymous@dev.local")
				c.Set("user_role", "admin")
				c.Set("tenant_id", tenantID)
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

	var tenantID string
	switch v := ctx.Value("tenant_id").(type) {
	case string:
		tenantID = v
	case uuid.UUID:
		tenantID = v.String()
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
