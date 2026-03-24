package auth

import (
	"log"

	"github.com/arc-platform/backend/modules/auth/api"
	"github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type AuthModule struct {
	handler    *api.AuthHandler
	middleware *middleware.AuthMiddleware
	pgRepo     *persistence.PostgresRepository
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
	m.handler = api.NewAuthHandler(m.pgRepo)
	m.middleware = middleware.NewAuthMiddleware(m.pgRepo)

	log.Printf("✅ Auth Module initialized")
	return nil
}

func (m *AuthModule) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", m.handler.Login)
		auth.POST("/register", m.handler.Register)
		auth.POST("/refresh", m.handler.Refresh)

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
	return nil
}

func (m *AuthModule) GetMiddleware() *middleware.AuthMiddleware {
	return m.middleware
}

func (m *AuthModule) GetRepository() *persistence.PostgresRepository {
	return m.pgRepo
}
