# 📋 architecture/ingestion-sop.md - Ingestion SOP

**Version:** 1.0.0
**Date:** 2026-01-22
**Status:** Draft

---

## 🎯 Purpose

This SOP defines the data ingestion workflow from Scanner SDK to Backend API. Ensures data integrity and proper validation.

---

## 📊 Input

### Scanner Output Payload

```json
{
  "fs": [
    {
      "host": "string",
      "file_path": "string",
      "pattern_name": "IN_AADHAAR",
      "matches": ["string"],
      "sample_text": "string",
      "profile": "string",
      "data_source": "fs",
      "severity": "Critical|High|Medium|Low",
      "file_data": {}
    }
  ],
  "postgresql": [],
  "mongodb": [],
  "s3": [],
  "gcs": []
}
```

---

## 🔄 Workflow Steps

### Step 1: Receive Payload

**Endpoint:** `POST /api/v1/scans/ingest-verified`

```go
func (h *SDKIngestHandler) IngestVerified(c *gin.Context) {
    var payload map[string][]Finding
    
    if err := c.ShouldBindJSON(&payload); err != nil {
        c.JSON(400, gin.H{"error": "Invalid payload format"})
        return
    }
    
    // Step 2: Validate payload
    if !h.validatePayload(payload) {
        c.JSON(400, gin.H{"error": "No findings provided"})
        return
    }
    
    // Step 3: Process each source
    for sourceType, findings := range payload {
        count, err := h.ingestionService.Ingest(findings)
        if err != nil {
            log.Printf("Error ingesting %s: %v", sourceType, err)
            continue
        }
        log.Printf("Ingested %d findings from %s", count, sourceType)
    }
}
```

### Step 2: Validate Payload

**Rules:**
1. Payload MUST contain at least one finding
2. Each finding MUST have `verified: true`
3. Each finding MUST have `pattern_name`
4. Each finding MUST have `data_source`

**Validation Logic:**
```go
func (h *IngestionService) validatePayload(findings []Finding) bool {
    for _, f := range findings {
        if !f.Verified {
            return false // REJECT - only verified findings
        }
        if f.PatternName == "" {
            return false
        }
        if f.DataSource == "" {
            return false
        }
    }
    return true
}
```

### Step 3: Enrich Findings

Add metadata before storage:

```go
func (h *IngestionService) enrichFinding(finding *Finding) {
    finding.ID = uuid.New().String()
    finding.CreatedAt = time.Now().UTC()
    finding.ScanRunID = h.getCurrentScanID()
    finding.TenantID = h.getCurrentTenantID()
}
```

### Step 4: Deduplicate

Check for existing findings:

```go
func (h *IngestionService) isDuplicate(finding Finding) bool {
    var count int64
    h.db.Model(&Finding{}).
        Where("asset_id = ?", finding.AssetID).
        Where("pattern_name = ?", finding.PatternName).
        Where("matches = ?", finding.Matches).
        Count(&count)
    return count > 0
}
```

### Step 5: Store in PostgreSQL

```go
func (h *IngestionService) storeFinding(finding *Finding) error {
    return h.db.Create(finding).Error
}
```

### Step 6: Update Classification

```go
func (h *IngestionService) updateClassification(finding *Finding) {
    classification := Classification{
        ID: uuid.New().String(),
        FindingID: finding.ID,
        ClassificationType: finding.PatternName,
        DPDPACategory: mapPIItoDPDPA(finding.PatternName),
        RequiresConsent: requiresConsent(finding.PatternName),
        Verified: true,
    }
    h.db.Create(&classification)
}
```

### Step 7: Link to Asset

```go
func (h *IngestionService) linkToAsset(finding *Finding) {
    asset, err := h.findOrCreateAsset(finding)
    if err != nil {
        log.Printf("Error linking asset: %v", err)
        return
    }
    finding.AssetID = asset.ID
}
```

---

## ⚠️ Error Handling

| Error Type | HTTP Code | Message |
|------------|-----------|---------|
| Invalid Payload | 400 | "Invalid payload format" |
| No Findings | 400 | "No findings provided" |
| Unverified Finding | 400 | "All findings must be verified by SDK" |
| Database Error | 500 | "Internal server error" |
| Asset Not Found | 404 | "Referenced asset not found" |

---

## 📈 Metrics

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Ingestion Rate | 500-1,000 findings/sec | < 200 findings/sec |
| Success Rate | 99.9% | < 99% |
| Latency (p95) | < 100ms | > 500ms |

---

## 🔗 Related Documents

- `scanning-sop.md` - Scanner workflow
- `ingestion-sop.md` - This SOP
- `lineage-sop.md` - Graph building workflow
- `gemini.md` - Data schemas

---

*This SOP is the source of truth for ingestion operations. Update before modifying backend code.*
