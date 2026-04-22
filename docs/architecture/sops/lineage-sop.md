# 📋 architecture/lineage-sop.md - Lineage SOP

**Version:** 1.0.0
**Date:** 2026-01-22
**Status:** Draft

---

## 🎯 Purpose

This SOP defines the semantic lineage graph building workflow. Maps PII flow from systems → assets → PII categories.

---

## 📊 Graph Schema

### Nodes

| Node Type | Properties | Example |
|-----------|------------|---------|
| **System** | `id`, `type`, `host`, `name` | `system-localhost` |
| **Asset** | `id`, `type`, `path`, `name`, `data_source` | `7ffd8211-e784-...` |
| **PII_Category** | `id`, `type`, `dpdpa_category` | `IN_AADHAAR` |

### Relationships

| Relationship | Direction | Properties |
|--------------|-----------|------------|
| **OWNS** | System → Asset | `metadata` |
| **CONTAINS** | Asset → PII_Category | `count`, `severity` |

---

## 🔄 Workflow Steps

### Step 1: Create/Update System Node

```go
func (h *LineageService) syncSystem(sourceType, host string) (*Node, error) {
    systemID := fmt.Sprintf("system-%s", host)
    
    // Check if exists
    existing, err := h.findNode(systemID)
    if err == nil && existing != nil {
        return existing, nil // Already exists
    }
    
    // Create or update
    node := &Node{
        ID: systemID,
        Type: "system",
        Label: host,
        Metadata: map[string]interface{}{
            "source_type": sourceType,
            "host": host,
        },
    }
    
    return h.upsertNode(node)
}
```

### Step 2: Create/Update Asset Node

```go
func (h *LineageService) syncAsset(finding Finding) (*Node, error) {
    assetID := finding.AssetID
    if assetID == "" {
        assetID = generateAssetID(finding)
    }
    
    node := &Node{
        ID: assetID,
        Type: "asset",
        Label: finding.FilePath,
        Metadata: map[string]interface{}{
            "path": finding.FilePath,
            "data_source": finding.DataSource,
            "profile": finding.Profile,
            "host": finding.Host,
        },
    }
    
    return h.upsertNode(node)
}
```

### Step 3: Create/Update PII Category Node

```go
func (h *LineageService) syncPIICategory(patternName string) (*Node, error) {
    piiID := fmt.Sprintf("pii-%s", patternName)
    
    node := &Node{
        ID: piiID,
        Type: "pii_category",
        Label: patternName,
        Metadata: map[string]interface{}{
            "dpdpa_category": mapToDPDPACategory(patternName),
            "requires_consent": requiresConsent(patternName),
        },
    }
    
    return h.upsertNode(node)
}
```

### Step 4: Create OWNS Relationship

```go
func (h *LineageService) linkSystemToAsset(systemID, assetID string) error {
    rel := &Relationship{
        Source: systemID,
        Target: assetID,
        Type: "OWNS",
        Metadata: map[string]interface{}{
            "created_at": time.Now().UTC(),
        },
    }
    return h.upsertRelationship(rel)
}
```

### Step 5: Create CONTAINS Relationship

```go
func (h *LineageService) linkAssetToPII(assetID, patternName string, count int) error {
    piiID := fmt.Sprintf("pii-%s", patternName)
    
    rel := &Relationship{
        Source: assetID,
        Target: piiID,
        Type: "CONTAINS",
        Metadata: map[string]interface{}{
            "count": count,
            "severity": getSeverity(patternName),
            "created_at": time.Now().UTC(),
        },
    }
    
    return h.upsertRelationship(rel)
}
```

### Step 6: Build Graph Query

```go
func (h *LineageService) GetSemanticGraph() (*Graph, error) {
    query := `
        MATCH (s:System)-[:OWNS]->(a:Asset)-[:CONTAINS]->(p:PII_Category)
        RETURN s, a, p
        ORDER BY s.id, a.id
    `
    
    results, err := h.db.Query(query)
    if err != nil {
        return nil, err
    }
    
    return buildGraphFromResults(results)
}
```

---

## ⚠️ Error Handling

| Error Type | Condition | Action |
|------------|-----------|--------|
| NodeCreationError | Failed to create node | Log error, continue |
| RelationshipError | Failed to create relationship | Log error, continue |
| CycleDetected | Circular dependency | Reject relationship |

---

## 📈 Metrics

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Graph Query Time | 50-150ms (p95) | > 500ms |
| Nodes Created | 1M max | > 900K |
| Relationships | 5M max | > 4.5M |

---

## 🔗 Related Documents

- `scanning-sop.md` - Scanner workflow
- `ingestion-sop.md` - Ingestion workflow
- `lineage-sop.md` - This SOP
- `gemini.md` - Graph schema

---

*This SOP is the source of truth for lineage operations. Update before modifying graph code.*
