package service

import "time"

// IssueType is a machine-readable identifier for a known remediation scenario.
type IssueType string

const (
	IssueUnencryptedPII       IssueType = "unencrypted_pii"
	IssueExcessiveRetention   IssueType = "excessive_retention"
	IssueMissingConsentTag    IssueType = "missing_consent_tag"
	IssueNoRBAC               IssueType = "no_rbac"
	IssueUnclassifiedSensitive IssueType = "unclassified_sensitive"
	IssueOrphanAsset          IssueType = "orphan_asset"
	IssueLineageCycle         IssueType = "lineage_cycle"
	IssuePIISpike             IssueType = "pii_volume_spike"
	IssueMissingDPO           IssueType = "missing_dpo_assignment"
	IssueMissingPurpose       IssueType = "missing_declared_purpose"
	IssueAuditChainBreak      IssueType = "audit_chain_broken"
	IssueStaleAadhaarStore    IssueType = "stale_aadhaar_store"
)

// SeverityLevel for a SOP.
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "critical"
	SeverityHigh     SeverityLevel = "high"
	SeverityMedium   SeverityLevel = "medium"
	SeverityLow      SeverityLevel = "low"
)

// SOP is a Standard Operating Procedure definition for a known issue type.
// Stored in-memory — no DB round-trip needed; these are static compliance requirements.
type SOP struct {
	IssueType       IssueType       `json:"issue_type"`
	Title           string          `json:"title"`
	DPDPASection    string          `json:"dpdpa_section"`   // e.g. "Sec 6", "Sec 17"
	Severity        SeverityLevel   `json:"severity"`
	EscalateAfter   time.Duration   `json:"-"`               // escalation threshold
	EscalateAfterHR string          `json:"escalate_after"`  // human-readable
	Steps           []SOPStep       `json:"steps"`
	AutoRemediation *AutoRemediation `json:"auto_remediation,omitempty"`
}

// SOPStep is one action in a SOP runbook.
type SOPStep struct {
	Order       int    `json:"order"`
	Actor       string `json:"actor"`       // "dpo" | "engineer" | "automated"
	Action      string `json:"action"`
	SuccessCriteria string `json:"success_criteria"`
}

// AutoRemediation describes how the system can auto-fix the issue.
type AutoRemediation struct {
	Supported bool   `json:"supported"`
	Method    string `json:"method"`
	RiskLevel string `json:"risk_level"` // "safe" | "moderate" | "high"
}

// SOPRegistry is the in-memory catalogue of all known SOPs.
var SOPRegistry = map[IssueType]*SOP{
	IssueUnencryptedPII: {
		IssueType:     IssueUnencryptedPII,
		Title:         "PII stored without encryption at rest",
		DPDPASection:  "Sec 8 (Data Accuracy), Sec 4 (Lawful Processing)",
		Severity:      SeverityCritical,
		EscalateAfter: 7 * 24 * time.Hour,
		EscalateAfterHR: "7 days",
		Steps: []SOPStep{
			{1, "engineer", "Identify the storage layer (column/file) containing plaintext PII", "Column identified in findings report"},
			{2, "engineer", "Apply AES-256-GCM encryption using platform EncryptionService", "Encrypted value written to DB; plaintext column removed or migrated"},
			{3, "dpo", "Update data processing record to reflect encrypted storage", "DPDPA Art 8(6) requirement met"},
			{4, "automated", "Re-scan asset to confirm zero plaintext PII findings", "Scanner returns 0 unencrypted PII matches"},
		},
		AutoRemediation: &AutoRemediation{
			Supported: false,
			Method:    "manual — requires key management decisions",
			RiskLevel: "high",
		},
	},

	IssueExcessiveRetention: {
		IssueType:     IssueExcessiveRetention,
		Title:         "Data retained beyond declared retention period",
		DPDPASection:  "Sec 17 (Retention and Erasure)",
		Severity:      SeverityHigh,
		EscalateAfter: 14 * 24 * time.Hour,
		EscalateAfterHR: "14 days",
		Steps: []SOPStep{
			{1, "engineer", "Query retention_violations view for affected asset", "List of overdue records exported"},
			{2, "dpo", "Confirm whether extended retention is legally justified", "Legal basis documented or erasure authorised"},
			{3, "engineer", "Execute purge job or archive to cold storage", "Row count in retention_violations drops to 0 for asset"},
			{4, "automated", "Update retention_policy_days if legal basis changed", "Policy record updated in DB"},
		},
		AutoRemediation: &AutoRemediation{
			Supported: true,
			Method:    "POST /remediation/:id/execute — triggers erasure workflow",
			RiskLevel: "moderate",
		},
	},

	IssueMissingConsentTag: {
		IssueType:     IssueMissingConsentTag,
		Title:         "PII asset lacks linked consent record",
		DPDPASection:  "Sec 6 (Consent)",
		Severity:      SeverityHigh,
		EscalateAfter: 3 * 24 * time.Hour,
		EscalateAfterHR: "3 days",
		Steps: []SOPStep{
			{1, "dpo", "Determine the legal basis for processing this data", "One of: explicit consent, legitimate interest, contract, legal obligation"},
			{2, "dpo", "Create consent record via POST /compliance/consent", "Consent record ID returned and linked to asset"},
			{3, "automated", "DPDPA obligation check re-runs — Sec 6 should show pass", "Sec 6 gap count = 0 for this asset"},
		},
		AutoRemediation: &AutoRemediation{
			Supported: false,
			Method:    "Consent must be explicitly recorded by DPO",
			RiskLevel: "high",
		},
	},

	IssueNoRBAC: {
		IssueType:     IssueNoRBAC,
		Title:         "Data asset accessible without role-based access control",
		DPDPASection:  "Sec 8 (Accuracy and Security)",
		Severity:      SeverityHigh,
		EscalateAfter: 5 * 24 * time.Hour,
		EscalateAfterHR: "5 days",
		Steps: []SOPStep{
			{1, "engineer", "Identify access control policy for the data store", "Current ACL documented"},
			{2, "engineer", "Apply least-privilege RBAC: restrict to named roles only", "ACL updated; access log shows only authorised principals"},
			{3, "dpo", "Update data processing record with access restriction details", "Processing record updated"},
		},
		AutoRemediation: &AutoRemediation{Supported: false, Method: "manual", RiskLevel: "high"},
	},

	IssueUnclassifiedSensitive: {
		IssueType:     IssueUnclassifiedSensitive,
		Title:         "High-entropy data present without PII classification",
		DPDPASection:  "Sec 4 (Lawful Processing)",
		Severity:      SeverityMedium,
		EscalateAfter: 10 * 24 * time.Hour,
		EscalateAfterHR: "10 days",
		Steps: []SOPStep{
			{1, "automated", "Re-run scan with classification_mode=contextual to trigger LLM layer", "Classification result stored; classifier=llm"},
			{2, "dpo", "Review LLM classification result — approve or override", "Manual classification tag set"},
		},
		AutoRemediation: &AutoRemediation{
			Supported: true,
			Method:    "POST /scans/trigger with classification_mode=contextual",
			RiskLevel: "safe",
		},
	},

	IssueOrphanAsset: {
		IssueType:     IssueOrphanAsset,
		Title:         "Asset has no scan findings for >90 days (orphan)",
		DPDPASection:  "Sec 8 (Data Accuracy)",
		Severity:      SeverityMedium,
		EscalateAfter: 14 * 24 * time.Hour,
		EscalateAfterHR: "14 days",
		Steps: []SOPStep{
			{1, "engineer", "Verify if data source still exists and is reachable", "Connection test result"},
			{2, "engineer", "Re-trigger scan for this specific source", "Scan completes; is_orphan flag cleared"},
			{3, "dpo", "If source is decommissioned: delete asset record or archive", "Asset status updated"},
		},
		AutoRemediation: &AutoRemediation{Supported: false, Method: "manual verification required", RiskLevel: "moderate"},
	},

	IssueMissingDPO: {
		IssueType:     IssueMissingDPO,
		Title:         "High-risk asset (score ≥60) has no DPO assigned",
		DPDPASection:  "Sec 10 (Data Fiduciary Obligations)",
		Severity:      SeverityHigh,
		EscalateAfter: 7 * 24 * time.Hour,
		EscalateAfterHR: "7 days",
		Steps: []SOPStep{
			{1, "dpo", "Assign a DPO or DPO delegate to the asset", "dpo_assigned field set on asset"},
			{2, "dpo", "Document DPO responsibilities in the data processing record", "Processing record updated"},
		},
		AutoRemediation: &AutoRemediation{Supported: false, Method: "manual", RiskLevel: "high"},
	},

	IssueMissingPurpose: {
		IssueType:     IssueMissingPurpose,
		Title:         "Asset has no declared_purpose tag",
		DPDPASection:  "Sec 5 (Purpose Limitation)",
		Severity:      SeverityHigh,
		EscalateAfter: 7 * 24 * time.Hour,
		EscalateAfterHR: "7 days",
		Steps: []SOPStep{
			{1, "dpo", "Document the lawful purpose for collecting/processing this data", "declared_purpose column set on asset via PUT /assets/:id"},
			{2, "dpo", "Confirm purpose is included in the privacy notice", "Privacy notice reference ID documented"},
		},
		AutoRemediation: &AutoRemediation{Supported: false, Method: "manual — requires legal review", RiskLevel: "high"},
	},

	IssueAuditChainBreak: {
		IssueType:     IssueAuditChainBreak,
		Title:         "SHA-256 audit log chain is broken — tampering suspected",
		DPDPASection:  "Sec 8 (Security Safeguards)",
		Severity:      SeverityCritical,
		EscalateAfter: 0, // escalate immediately
		EscalateAfterHR: "immediate",
		Steps: []SOPStep{
			{1, "engineer", "Run python manage.py verify-audit-chain to identify break point", "Break index and entry ID identified"},
			{2, "dpo", "Preserve broken chain as forensic evidence — do not delete entries", "Snapshot exported to cold storage"},
			{3, "engineer", "Investigate who modified audit_logs table directly", "DB access logs reviewed; source identified"},
			{4, "dpo", "Report to CERT-In under DPDPA incident notification rules if PII was accessed", "Incident report filed"},
		},
		AutoRemediation: &AutoRemediation{Supported: false, Method: "forensic investigation required", RiskLevel: "high"},
	},
}

// LookupSOP returns the SOP for an issue type, or nil if not found.
func LookupSOP(issueType IssueType) *SOP {
	return SOPRegistry[issueType]
}

// ListSOPs returns all SOPs in the registry.
func ListSOPs() []*SOP {
	out := make([]*SOP, 0, len(SOPRegistry))
	for _, sop := range SOPRegistry {
		out = append(out, sop)
	}
	return out
}
