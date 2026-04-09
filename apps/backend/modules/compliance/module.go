package compliance

import (
	"log"

	"github.com/arc-platform/backend/modules/compliance/api"
	"github.com/arc-platform/backend/modules/compliance/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type ComplianceModule struct {
	complianceService   *service.ComplianceService
	consentService      *service.ConsentService
	retentionService    *service.RetentionService
	auditService        *service.AuditService
	obligationService   *service.DPDPAObligationService
	reportService       *service.ReportService

	complianceHandler      *api.ComplianceHandler
	consentHandler         *api.ConsentHandler
	consentRecordsHandler  *api.ConsentRecordsHandler
	retentionHandler       *api.RetentionHandler
	auditHandler           *api.AuditHandler
	dpdpaReportHandler     *api.DPDPAReportHandler

	deps *interfaces.ModuleDependencies
}

func (m *ComplianceModule) Name() string {
	return "compliance"
}

func (m *ComplianceModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("⚖️  Initializing Compliance Module...")

	repo := persistence.NewPostgresRepository(deps.DB)

	// Initialize services
	m.complianceService = service.NewComplianceService(repo, deps.Neo4jRepo)
	m.consentService = service.NewConsentService(deps.DB)
	m.retentionService = service.NewRetentionService(deps.DB)
	m.auditService = service.NewAuditService(deps.DB)
	m.obligationService = service.NewDPDPAObligationService(repo)
	m.reportService = service.NewReportService(m.obligationService)

	// Initialize handlers
	m.complianceHandler = api.NewComplianceHandler(m.complianceService)
	m.consentHandler = api.NewConsentHandler(m.consentService)
	m.consentRecordsHandler = api.NewConsentRecordsHandler(deps.DB)
	m.retentionHandler = api.NewRetentionHandler(m.retentionService)
	m.auditHandler = api.NewAuditHandler(m.auditService)
	m.dpdpaReportHandler = api.NewDPDPAReportHandler(m.obligationService, m.reportService)

	log.Printf("✅ Compliance Module initialized (6 services)")
	return nil
}

func (m *ComplianceModule) RegisterRoutes(router *gin.RouterGroup) {
	compliance := router.Group("/compliance")
	{
		compliance.GET("/overview", m.complianceHandler.GetComplianceOverview)
		compliance.GET("/violations", m.complianceHandler.GetConsentViolations)
		compliance.GET("/critical", m.complianceHandler.GetCriticalAssets)

		// DPDPA 2023 obligation mapping endpoints
		dpdpa := compliance.Group("/dpdpa")
		dpdpa.GET("/gaps", m.dpdpaReportHandler.GetObligationGaps)
		dpdpa.GET("/report", m.dpdpaReportHandler.GenerateHTMLReport)

		// Consent records (migration 000030 schema) — full lifecycle management.
		consentRec := compliance.Group("/consent")
		consentRec.GET("", m.consentRecordsHandler.ListConsentRecords)
		consentRec.POST("", m.consentRecordsHandler.CreateConsentRecord)
		consentRec.DELETE("/:id", m.consentRecordsHandler.WithdrawConsentRecord)
	}

	// Consent management routes
	consent := router.Group("/consent")
	{
		consent.POST("/records", m.consentHandler.RecordConsent)
		consent.GET("/records", m.consentHandler.ListConsentRecords)
		consent.POST("/withdraw/:id", m.consentHandler.WithdrawConsent)
		consent.GET("/status/:assetId/:piiType", m.consentHandler.GetConsentStatus)
		consent.GET("/violations", m.consentHandler.GetConsentViolations)
	}

	// Retention policy routes
	retention := router.Group("/retention")
	{
		retention.POST("/policies", m.retentionHandler.SetRetentionPolicy)
		retention.GET("/policies/:assetId", m.retentionHandler.GetRetentionPolicy)
		retention.GET("/violations", m.retentionHandler.GetRetentionViolations)
		retention.GET("/timeline/:assetId", m.retentionHandler.GetRetentionTimeline)
	}

	// Audit log routes
	audit := router.Group("/audit")
	{
		audit.GET("/logs", m.auditHandler.ListAuditLogs)
		audit.GET("/user/:userId", m.auditHandler.GetUserActivity)
		audit.GET("/resource/:resourceType/:resourceId", m.auditHandler.GetResourceHistory)
		audit.GET("/recent", m.auditHandler.GetRecentActivity)
	}

	log.Printf("⚖️  Compliance routes registered (20 endpoints)")
}

func (m *ComplianceModule) Shutdown() error {
	log.Printf("🔌 Shutting down Compliance Module...")
	return nil
}

func NewComplianceModule() *ComplianceModule {
	return &ComplianceModule{}
}
