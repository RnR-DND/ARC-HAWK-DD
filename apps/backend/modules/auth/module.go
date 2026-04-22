package auth

import (
	"log"

	"github.com/arc-platform/backend/modules/auth/api"
	"github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	sharedmiddleware "github.com/arc-platform/backend/modules/shared/middleware"
	"github.com/gin-gonic/gin"
)

type AuthModule struct {
	handler    *api.AuthHandler
	middleware *middleware.AuthMiddleware
	pgRepo     *persistence.PostgresRepository
	strictRL   *sharedmiddleware.RateLimiter
}

func NewAuthModule() *AuthModule {
	return &AuthModule{}
}

func (m *AuthModule) Name() string {
	return "auth"
}

func (m *AuthModule) Initialize(deps *interfaces.ModuleDependencies) error {
	log.Printf("📡 Initializing Auth Module...")

	m.pgRepo = persistence.NewPostgresRepository(deps.DB)
	m.handler = api.NewAuthHandler(m.pgRepo, deps.DB, deps.AuditLogger)
	m.middleware = middleware.NewAuthMiddleware(m.pgRepo, deps.DB)
	m.strictRL = sharedmiddleware.StrictRateLimiter()

	log.Printf("✅ Auth Module initialized")
	return nil
}

func (m *AuthModule) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		// R-06: strict rate limit (10 req/min per IP) on unauthenticated endpoints
		public := auth.Group("")
		public.Use(m.strictRL.Middleware())
		{
			public.POST("/login", m.handler.Login)
			public.POST("/register", m.handler.Register)
			public.POST("/refresh", m.handler.Refresh)
		}

		protected := auth.Group("")
		protected.Use(m.middleware.Authenticate())
		{
			protected.GET("/profile", m.handler.GetProfile)
			protected.POST("/change-password", m.handler.ChangePassword)
			protected.GET("/users", m.handler.ListUsers)

			// Settings
			protected.GET("/settings", m.handler.GetSettings)
			protected.PUT("/settings", m.handler.UpdateSettings)
		}
	}
}

func (m *AuthModule) Shutdown() error {
	log.Printf("🔌 Shutting down Auth Module...")
	if m.strictRL != nil {
		m.strictRL.Stop()
	}
	return nil
}

func (m *AuthModule) GetMiddleware() *middleware.AuthMiddleware {
	return m.middleware
}

func (m *AuthModule) GetRepository() *persistence.PostgresRepository {
	return m.pgRepo
}
