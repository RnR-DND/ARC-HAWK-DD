# 📋 architecture/compliance-sop.md - Compliance SOP

**Version:** 1.0.0
**Date:** 2026-01-22
**Status:** Draft

---

## 🎯 Purpose

This SOP defines the DPDPA 2023 compliance mapping, consent tracking, and retention policy enforcement.

---

## 📊 DPDPA Categories

| PII Type | DPDPA Category | Requires Consent |
|----------|----------------|------------------|
| IN_AADHAAR | Sensitive Personal Data | Yes |
| IN_PAN | Financial Identifier | Yes |
| CREDIT_CARD | Financial Identifier | Yes |
| IN_PASSPORT | Sensitive Personal Data | Yes |
| EMAIL | Contact Information | Yes |
| PHONE | Contact Information | Yes |
| BANK_ACCOUNT | Financial Identifier | Yes |
| IFSC | Financial Identifier | Yes |

---

## 🔄 Workflow Steps

### Step 1: Map PII to DPDPA Category

```go
func mapToDPDPACategory(patternName string) string {
    mapping := map[string]string{
        "IN_AADHAAR":      "Sensitive Personal Data",
        "IN_PAN":          "Financial Identifier",
        "CREDIT_CARD":     "Financial Identifier",
        "IN_PASSPORT":     "Sensitive Personal Data",
        "EMAIL":           "Contact Information",
        "PHONE":           "Contact Information",
        "BANK_ACCOUNT":    "Financial Identifier",
        "IFSC":            "Financial Identifier",
    }
    return mapping[patternName]
}
```

### Step 2: Check Consent Requirements

```go
func requiresConsent(patternName string) bool {
    consentRequired := map[string]bool{
        "IN_AADHAAR":      true,
        "IN_PAN":          true,
        "CREDIT_CARD":     true,
        "IN_PASSPORT":     true,
        "EMAIL":           true,
        "PHONE":           true,
        "BANK_ACCOUNT":    true,
        "IFSC":            true,
    }
    return consentRequired[patternName]
}
```

### Step 3: Record Consent

```go
func (h *ComplianceService) RecordConsent(record ConsentRecord) error {
    consent := &ConsentRecord{
        ID: uuid.New().String(),
        AssetID: record.AssetID,
        PIICategory: record.PIICategory,
        ConsentType: record.ConsentType, // "explicit", "implicit"
        Source: record.Source, // "form", "verbal", "contract"
        GrantedAt: time.Now().UTC(),
        ExpiresAt: record.ExpiresAt,
        Status: "active",
    }
    return h.db.Create(consent).Error
}
```

### Step 4: Check Consent Violations

```go
func (h *ComplianceService) GetConsentViolations() ([]Violation, error) {
    query := `
        SELECT DISTINCT f.asset_id, c.classification_type
        FROM findings f
        JOIN classifications c ON f.id = c.finding_id
        WHERE c.requires_consent = true
        AND NOT EXISTS (
            SELECT 1 FROM consent_records cr
            WHERE cr.asset_id = f.asset_id
            AND cr.pii_category = c.classification_type
            AND cr.status = 'active'
        )
    `
    
    var violations []Violation
    err := h.db.Raw(query).Scan(&violations).Error
    return violations, err
}
```

### Step 5: Set Retention Policy

```go
func (h *ComplianceService) SetRetentionPolicy(policy RetentionPolicy) error {
    retention := &RetentionPolicy{
        ID: uuid.New().String(),
        AssetID: policy.AssetID,
        PIICategory: policy.PIICategory,
        RetentionPeriodDays: policy.RetentionPeriodDays, // e.g., 365, 730, 1825
        LegalBasis: policy.LegalBasis, // "legitimate_use", "consent", "legal_obligation"
        CreatedAt: time.Now().UTC(),
    }
    return h.db.Create(retention).Error
}
```

### Step 6: Check Retention Violations

```go
func (h *ComplianceService) GetRetentionViolations() ([]Violation, error) {
    query := `
        SELECT asset_id, pii_category, created_at
        FROM findings
        WHERE created_at < NOW() - INTERVAL '1 day' * retention_period_days
        AND retention_period_days IS NOT NULL
    `
    
    var violations []Violation
    err := h.db.Raw(query).Scan(&violations).Error
    return violations, err
}
```

### Step 7: Generate Compliance Overview

```go
func (h *ComplianceService) GetComplianceOverview() (*ComplianceOverview, error) {
    overview := &ComplianceOverview{
        TotalAssets: h.countAssets(),
        CompliantAssets: h.countCompliantAssets(),
        NonCompliantAssets: h.countNonCompliantAssets(),
        ConsentViolations: h.countConsentViolations(),
        RetentionViolations: h.countRetentionViolations(),
        DPDPABreakdown: h.getDPDPABreakdown(),
    }
    
    overview.ComplianceScore = calculateScore(overview)
    return overview, nil
}
```

### Step 8: Audit Logging

```go
func (h *ComplianceService) LogAuditEvent(event AuditEvent) error {
    audit := &AuditLog{
        ID: uuid.New().String(),
        UserID: event.UserID,
        Action: event.Action, // "scan", "view", "export", "mask"
        ResourceType: event.ResourceType, // "finding", "asset", "report"
        ResourceID: event.ResourceID,
        Details: event.Details,
        Timestamp: time.Now().UTC(),
        IPAddress: event.IPAddress,
    }
    return h.db.Create(audit).Error
}
```

---

## ⚠️ Error Handling

| Error Type | Condition | Action |
|------------|-----------|--------|
| ConsentExpired | Consent record expired | Flag for renewal |
| RetentionExceeded | Data held too long | Flag for deletion |
| AuditLogFailed | Cannot write audit | Log to file fallback |

---

## 📈 Metrics

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Compliance Score | 80%+ | < 70% |
| Consent Coverage | 100% | < 95% |
| Retention Compliance | 100% | < 95% |
| Audit Log Latency | < 100ms | > 500ms |

---

## 🔗 Related Documents

- `scanning-sop.md` - Scanner workflow
- `ingestion-sop.md` - Ingestion workflow
- `lineage-sop.md` - Lineage workflow
- `compliance-sop.md` - This SOP
- `gemini.md` - DPDPA rules

---

*This SOP is the source of truth for compliance operations. Update before modifying compliance code.*
