// Package auth handles tenant authentication, JWT issuance, refresh tokens, and rate-limited login.
package auth

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/arc-platform/backend/modules/auth/api"
	"github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	sharedmiddleware "github.com/arc-platform/backend/modules/shared/middleware"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type AuthModule struct {
	handler    *api.AuthHandler
	middleware *middleware.AuthMiddleware
	pgRepo     *persistence.PostgresRepository
	strictRL   *sharedmiddleware.RateLimiter
	db         *sql.DB
}

func NewAuthModule() *AuthModule {
	return &AuthModule{}
}

func (m *AuthModule) Name() string {
	return "auth"
}

func (m *AuthModule) Initialize(deps *interfaces.ModuleDependencies) error {
	log.Printf("📡 Initializing Auth Module...")

	m.db = deps.DB
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

			// Notification settings
			protected.GET("/settings/notifications", func(c *gin.Context) {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
				defer cancel()
				tenantID, err := persistence.EnsureTenantID(ctx)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}
				type NotifSettings struct {
					ID                     string   `json:"id"`
					EmailEnabled           bool     `json:"email_enabled"`
					EmailRecipients        []string `json:"email_recipients"`
					SlackEnabled           bool     `json:"slack_enabled"`
					SlackWebhookURL        string   `json:"slack_webhook_url"`
					NotifyOnScanComplete   bool     `json:"notify_on_scan_complete"`
					NotifyOnHighSeverity   bool     `json:"notify_on_high_severity"`
					NotifyOnStaleConnector bool     `json:"notify_on_stale_connector"`
					SeverityThreshold      string   `json:"severity_threshold"`
				}
				var ns NotifSettings
				var slackURL sql.NullString
				err = m.db.QueryRowContext(ctx, `
					SELECT id, email_enabled, email_recipients, slack_enabled, slack_webhook_url,
					       notify_on_scan_complete, notify_on_high_severity, notify_on_stale_connector, severity_threshold
					FROM notification_settings WHERE tenant_id = $1`, tenantID).Scan(
					&ns.ID, &ns.EmailEnabled, pq.Array(&ns.EmailRecipients), &ns.SlackEnabled, &slackURL,
					&ns.NotifyOnScanComplete, &ns.NotifyOnHighSeverity, &ns.NotifyOnStaleConnector, &ns.SeverityThreshold,
				)
				if err == sql.ErrNoRows {
					// Return defaults if not configured yet
					c.JSON(http.StatusOK, gin.H{"settings": NotifSettings{
						EmailRecipients: []string{}, SeverityThreshold: "High",
						NotifyOnScanComplete: true, NotifyOnHighSeverity: true,
					}})
					return
				}
				if err != nil {
					log.Printf("ERROR: get notification settings: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
					return
				}
				if slackURL.Valid {
					ns.SlackWebhookURL = slackURL.String
				}
				if ns.EmailRecipients == nil {
					ns.EmailRecipients = []string{}
				}
				c.JSON(http.StatusOK, gin.H{"settings": ns})
			})

			protected.PUT("/settings/notifications", func(c *gin.Context) {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
				defer cancel()
				tenantID, err := persistence.EnsureTenantID(ctx)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}
				var body struct {
					EmailEnabled           bool     `json:"email_enabled"`
					EmailRecipients        []string `json:"email_recipients"`
					SlackEnabled           bool     `json:"slack_enabled"`
					SlackWebhookURL        *string  `json:"slack_webhook_url"`
					NotifyOnScanComplete   bool     `json:"notify_on_scan_complete"`
					NotifyOnHighSeverity   bool     `json:"notify_on_high_severity"`
					NotifyOnStaleConnector bool     `json:"notify_on_stale_connector"`
					SeverityThreshold      string   `json:"severity_threshold"`
				}
				if err := c.ShouldBindJSON(&body); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
					return
				}
				if body.EmailRecipients == nil {
					body.EmailRecipients = []string{}
				}
				validThresholds := map[string]bool{"Critical": true, "High": true, "Medium": true, "Low": true}
				if body.SeverityThreshold != "" && !validThresholds[body.SeverityThreshold] {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid severity_threshold"})
					return
				}
				if body.SeverityThreshold == "" {
					body.SeverityThreshold = "High"
				}
				_, err = m.db.ExecContext(ctx, `
					INSERT INTO notification_settings
					  (tenant_id, email_enabled, email_recipients, slack_enabled, slack_webhook_url,
					   notify_on_scan_complete, notify_on_high_severity, notify_on_stale_connector, severity_threshold)
					VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
					ON CONFLICT (tenant_id) DO UPDATE SET
					  email_enabled = EXCLUDED.email_enabled,
					  email_recipients = EXCLUDED.email_recipients,
					  slack_enabled = EXCLUDED.slack_enabled,
					  slack_webhook_url = EXCLUDED.slack_webhook_url,
					  notify_on_scan_complete = EXCLUDED.notify_on_scan_complete,
					  notify_on_high_severity = EXCLUDED.notify_on_high_severity,
					  notify_on_stale_connector = EXCLUDED.notify_on_stale_connector,
					  severity_threshold = EXCLUDED.severity_threshold,
					  updated_at = NOW()`,
					tenantID, body.EmailEnabled, pq.Array(body.EmailRecipients), body.SlackEnabled, body.SlackWebhookURL,
					body.NotifyOnScanComplete, body.NotifyOnHighSeverity, body.NotifyOnStaleConnector, body.SeverityThreshold,
				)
				if err != nil {
					log.Printf("ERROR: upsert notification settings: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"saved": true})
			})
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
