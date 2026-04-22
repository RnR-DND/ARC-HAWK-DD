package compliance

import (
	"context"
	"log"

	compapi "github.com/arc-platform/backend/modules/compliance/api"
	compsvc "github.com/arc-platform/backend/modules/compliance/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// scanHookSetter is satisfied by *scanning.ScanningModule without importing it.
type scanHookSetter interface {
	SetScanCompletionHook(fn func(ctx context.Context, tenantID uuid.UUID, scanRunID string, totalFindings int))
}

type ComplianceModule struct {
	complianceService *compsvc.ComplianceService
	consentService    *compsvc.ConsentService
	retentionService  *compsvc.RetentionService
	auditService      *compsvc.AuditService
	obligationService *compsvc.DPDPAObligationService
	reportService     *compsvc.ReportService

	ledgerLogger        *audit.LedgerLogger
	regressionDetector  *compsvc.ObligationRegressionDetector
	evidencePackageSvc  *compsvc.EvidencePackageService

	complianceHandler     *compapi.ComplianceHandler
	consentHandler        *compapi.ConsentHandler
	consentRecordsHandler *compapi.ConsentRecordsHandler
	retentionHandler      *compapi.RetentionHandler
	auditHandler          *compapi.AuditHandler
	dpdpaReportHandler    *compapi.DPDPAReportHandler
	dataPrincipalHandler  *compapi.DataPrincipalHandler
	groHandler            *compapi.GROHandler
	evidenceHandler       *compapi.EvidenceHandler

	deps *interfaces.ModuleDependencies
}

func (m *ComplianceModule) Name() string {
	return "compliance"
}

func (m *ComplianceModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("⚖️  Initializing Compliance Module...")

	repo := persistence.NewPostgresRepository(deps.DB)

	// Core services
	m.complianceService = compsvc.NewComplianceService(repo, deps.Neo4jRepo)
	m.consentService = compsvc.NewConsentService(deps.DB)
	m.retentionService = compsvc.NewRetentionService(deps.DB)
	m.auditService = compsvc.NewAuditService(deps.DB)
	m.obligationService = compsvc.NewDPDPAObligationService(repo)
	m.reportService = compsvc.NewReportService(m.obligationService)

	// DPDP evidence services
	m.ledgerLogger = audit.NewLedgerLogger(deps.DB)
	m.regressionDetector = compsvc.NewObligationRegressionDetector(deps.DB, m.ledgerLogger)
	m.evidencePackageSvc = compsvc.NewEvidencePackageService(deps.DB, m.ledgerLogger)

	// Handlers
	m.complianceHandler = compapi.NewComplianceHandler(m.complianceService)
	m.consentHandler = compapi.NewConsentHandler(m.consentService)
	m.consentRecordsHandler = compapi.NewConsentRecordsHandler(deps.DB)
	m.retentionHandler = compapi.NewRetentionHandler(m.retentionService)
	m.auditHandler = compapi.NewAuditHandler(m.auditService)
	m.dpdpaReportHandler = compapi.NewDPDPAReportHandler(m.obligationService, m.reportService)
	m.dataPrincipalHandler = compapi.NewDataPrincipalHandler(deps.DB)
	m.groHandler = compapi.NewGROHandler(deps.DB)
	m.evidenceHandler = compapi.NewEvidenceHandler(m.evidencePackageSvc, m.ledgerLogger)

	// Wire scan completion hook into scanning module (duck-typed, no import needed).
	// Scanning is initialized before compliance, so it's already in the registry.
	if scanMod, ok := deps.Registry.Get("scanning"); ok {
		if setter, ok := scanMod.(scanHookSetter); ok {
			ledger := m.ledgerLogger
			detector := m.regressionDetector
			setter.SetScanCompletionHook(func(ctx context.Context, tenantID uuid.UUID, scanRunID string, totalFindings int) {
				_ = ledger.Log(ctx, audit.LogEntry{
					TenantID:     tenantID,
					EventType:    audit.EventScanCompleted,
					ResourceID:   scanRunID,
					ResourceType: "scan_run",
					Payload: map[string]interface{}{
						"total_findings": totalFindings,
					},
				})
				_ = detector.DetectRegressions(ctx, tenantID, scanRunID)
			})
			log.Printf("⚖️  Scan completion hook wired (audit ledger + obligation regression)")
		}
	}

	log.Printf("✅ Compliance Module initialized (9 services, evidence package + audit ledger)")
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

		// Consent records (migration 000030 schema)
		consentRec := compliance.Group("/consent")
		consentRec.GET("", m.consentRecordsHandler.ListConsentRecords)
		consentRec.POST("", m.consentRecordsHandler.CreateConsentRecord)
		consentRec.DELETE("/:id", m.consentRecordsHandler.WithdrawConsentRecord)

		// DPDPA Sec 7 — Data Principal Rights requests
		compliance.POST("/dpr", m.dataPrincipalHandler.SubmitRequest)
		compliance.GET("/dpr", m.dataPrincipalHandler.ListRequests)
		compliance.GET("/dpr/stats", m.dataPrincipalHandler.GetStats)
		compliance.PATCH("/dpr/:id/status", m.dataPrincipalHandler.UpdateStatus)

		// DPDPA Sec 11 — Grievance Redressal Officer settings
		compliance.GET("/gro-settings", m.groHandler.GetSettings)
		compliance.PUT("/gro-settings", m.groHandler.UpdateSettings)

		// DPDP Act 2023 evidence package + immutable audit trail
		compliance.POST("/evidence-package", m.evidenceHandler.GeneratePackage)
		compliance.GET("/audit-trail", m.evidenceHandler.GetAuditTrail)
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

	// Audit log routes (existing audit_logs table)
	auditGroup := router.Group("/audit")
	{
		auditGroup.GET("/logs", m.auditHandler.ListAuditLogs)
		auditGroup.GET("/user/:userId", m.auditHandler.GetUserActivity)
		auditGroup.GET("/resource/:resourceType/:resourceId", m.auditHandler.GetResourceHistory)
		auditGroup.GET("/recent", m.auditHandler.GetRecentActivity)
	}

	log.Printf("⚖️  Compliance routes registered (28 endpoints)")
}

func (m *ComplianceModule) Shutdown() error {
	log.Printf("🔌 Shutting down Compliance Module...")
	return nil
}

func NewComplianceModule() *ComplianceModule {
	return &ComplianceModule{}
}
