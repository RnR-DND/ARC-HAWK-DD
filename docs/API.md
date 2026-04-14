# API Documentation

Complete API reference for the ARC-Hawk Backend. Base URL: `http://localhost:8080/api/v1`

## Table of Contents

- [Authentication](#authentication)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Scanning API](#scanning-api)
- [Custom Patterns API](#custom-patterns-api)
- [Findings API](#findings-api)
- [Assets API](#assets-api)
- [Discovery API](#discovery-api)
- [Lineage API](#lineage-api)
- [Connections API](#connections-api)
- [Compliance API](#compliance-api)
- [Remediation API](#remediation-api)
- [Analytics API](#analytics-api)
- [WebSocket API](#websocket-api)

---

## Authentication

> **Note**: Authentication is not yet implemented. All endpoints are currently public.

When authentication is implemented, the API will support:

- **JWT Bearer Token**: `Authorization: Bearer <token>`
- **API Key**: `X-API-Key: <api_key>`

---

## Response Format

All API responses follow a consistent JSON structure:

### Success Response

```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 200,
    "total_pages": 4
  },
  "message": "Operation completed successfully"
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": { ... }
  },
  "request_id": "req-uuid-123"
}
```

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource conflict |
| 422 | Unprocessable Entity | Validation error |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 503 | Service Unavailable | Service temporarily unavailable |

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Request format is invalid |
| `VALIDATION_ERROR` | Request validation failed |
| `NOT_FOUND` | Resource not found |
| `ALREADY_EXISTS` | Resource already exists |
| `UNAUTHORIZED` | Authentication required |
| `FORBIDDEN` | Insufficient permissions |
| `RATE_LIMITED` | Rate limit exceeded |
| `INTERNAL_ERROR` | Internal server error |
| `SERVICE_UNAVAILABLE` | Service temporarily unavailable |

---

## Scanning API

### Trigger a Scan

Initiates a new scan for an asset.

```http
POST /api/v1/scans/trigger
Content-Type: application/json
```

**Request Body:**

```json
{
  "asset_id": "asset-uuid-123",
  "scan_type": "full",
  "priority": "normal",
  "options": {
    "incremental": false,
    "dry_run": false
  }
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `asset_id` | string | Yes | UUID of the asset to scan |
| `scan_type` | string | No | Type of scan: `full`, `incremental`, `sample` (default: `full`) |
| `priority` | string | No | Priority: `low`, `normal`, `high` (default: `normal`) |
| `options` | object | No | Additional scan options |
| `options.incremental` | boolean | No | Only scan changed data |
| `options.dry_run` | boolean | No | Preview scan without saving |

**Response:**

```json
{
  "success": true,
  "data": {
    "scan_id": "scan-uuid-456",
    "workflow_id": "workflow-uuid-789",
    "status": "pending",
    "created_at": "2026-01-15T10:30:00Z",
    "estimated_duration": "5m"
  },
  "message": "Scan triggered successfully"
}
```

### Get Scan Status

Retrieves the current status of a scan.

```http
GET /api/v1/scans/{scan_id}/status
```

**Response:**

```json
{
  "success": true,
  "data": {
    "scan_id": "scan-uuid-456",
    "status": "running",
    "progress": 65,
    "started_at": "2026-01-15T10:30:05Z",
    "estimated_completion": "2026-01-15T10:35:00Z",
    "files_scanned": 650,
    "files_total": 1000,
    "findings_count": 12
  }
}
```

**Status Values:**

- `pending`: Waiting to start
- `running`: Currently scanning
- `completed`: Scan finished successfully
- `failed`: Scan failed
- `cancelled`: Scan was cancelled

### List Scans

Retrieves a list of all scans.

```http
GET /api/v1/scans?status=completed&page=1&limit=50
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status |
| `asset_id` | string | Filter by asset |
| `from_date` | string | Filter by start date (ISO 8601) |
| `to_date` | string | Filter by end date (ISO 8601) |
| `page` | integer | Page number (default: 1) |
| `limit` | integer | Items per page (default: 50, max: 100) |

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "scan_id": "scan-uuid-456",
      "asset_id": "asset-uuid-123",
      "asset_name": "Production PostgreSQL",
      "status": "completed",
      "scan_type": "full",
      "started_at": "2026-01-15T10:30:00Z",
      "completed_at": "2026-01-15T10:35:00Z",
      "duration": "5m",
      "findings_count": 12,
      "files_scanned": 1000
    }
  ],
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 25,
    "total_pages": 1
  }
}
```

### Cancel Scan

Cancels a running scan.

```http
POST /api/v1/scans/{scan_id}/cancel
```

**Response:**

```json
{
  "success": true,
  "data": {
    "scan_id": "scan-uuid-456",
    "status": "cancelled",
    "cancelled_at": "2026-01-15T10:32:00Z"
  },
  "message": "Scan cancelled successfully"
}
```

---

## Findings API

### List Findings

Retrieves a list of PII findings with filtering options.

```http
GET /api/v1/findings?status=open&severity=high&page=1&limit=50
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status: `open`, `resolved`, `false_positive`, `under_review` |
| `severity` | string | Filter by severity: `critical`, `high`, `medium`, `low` |
| `pii_type` | string | Filter by PII type: `aadhaar`, `pan`, `passport`, etc. |
| `asset_id` | string | Filter by asset |
| `scan_id` | string | Filter by scan |
| `from_date` | string | Filter by detection date (ISO 8601) |
| `to_date` | string | Filter by detection date (ISO 8601) |
| `page` | integer | Page number |
| `limit` | integer | Items per page |
| `sort_by` | string | Sort field: `detected_at`, `severity`, `pii_type` |
| `sort_order` | string | Sort order: `asc`, `desc` |

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "finding-uuid-001",
      "scan_id": "scan-uuid-456",
      "asset_id": "asset-uuid-123",
      "asset_name": "Production PostgreSQL",
      "status": "open",
      "severity": "critical",
      "pii_type": "aadhaar",
      "confidence": 1.0,
      "value": "1234-5678-9012",
      "location": {
        "path": "users.table",
        "line": 45,
        "column": "aadhaar_number"
      },
      "context": "... context around the finding ...",
      "detected_at": "2026-01-15T10:32:15Z",
      "remediation_status": null
    }
  ],
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 150,
    "total_pages": 3
  }
}
```

### Get Finding Details

Retrieves detailed information about a specific finding.

```http
GET /api/v1/findings/{finding_id}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "finding-uuid-001",
    "scan_id": "scan-uuid-456",
    "asset_id": "asset-uuid-123",
    "asset_name": "Production PostgreSQL",
    "status": "open",
    "severity": "critical",
    "pii_type": "aadhaar",
    "confidence": 1.0,
    "value": "1234-5678-9012",
    "validation": {
      "algorithm": "verhoeff",
      "is_valid": true,
      "checksum_verified": true
    },
    "location": {
      "path": "users.table",
      "line": 45,
      "column": "aadhaar_number",
      "file_type": "database"
    },
    "context": {
      "before": "... 3 lines before ...",
      "match": "aadhaar_number = 1234-5678-9012",
      "after": "... 3 lines after ..."
    },
    "compliance": {
      "dpdpa_category": "sensitive_personal_data",
      "consent_status": "unknown",
      "retention_days": 365
    },
    "detected_at": "2026-01-15T10:32:15Z",
    "updated_at": "2026-01-15T10:32:15Z",
    "remediation_history": []
  }
}
```

### Update Finding Status

Updates the status of a finding.

```http
PATCH /api/v1/findings/{finding_id}
Content-Type: application/json
```

**Request Body:**

```json
{
  "status": "false_positive",
  "feedback": "This is test data, not actual PII",
  "user_id": "user-uuid-123"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "finding-uuid-001",
    "status": "false_positive",
    "updated_at": "2026-01-15T11:00:00Z"
  },
  "message": "Finding updated successfully"
}
```

### Submit Feedback

Submits feedback about a finding (false positive, incorrect type, etc.).

```http
POST /api/v1/findings/{finding_id}/feedback
Content-Type: application/json
```

**Request Body:**

```json
{
  "is_false_positive": true,
  "reason": "test_data",
  "correct_pii_type": null,
  "comments": "This is synthetic test data used in development",
  "user_id": "user-uuid-123"
}
```

**Feedback Reasons:**

- `test_data`: Finding is test/synthetic data
- `not_pii`: Value is not actually PII
- `wrong_type`: PII type is incorrect
- `incorrect_location`: Location information is wrong
- `other`: Other reason

**Response:**

```json
{
  "success": true,
  "data": {
    "feedback_id": "feedback-uuid-789",
    "finding_id": "finding-uuid-001",
    "status": "submitted",
    "submitted_at": "2026-01-15T11:00:00Z"
  },
  "message": "Feedback submitted successfully"
}
```

### Bulk Update Findings

Updates multiple findings at once.

```http
POST /api/v1/findings/bulk-update
Content-Type: application/json
```

**Request Body:**

```json
{
  "finding_ids": ["finding-uuid-001", "finding-uuid-002"],
  "status": "resolved",
  "reason": "remediated"
}
```

---

## Assets API

### List Assets

Retrieves a list of all assets.

```http
GET /api/v1/assets?page=1&limit=50&source_type=postgresql
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `source_type` | string | Filter by source type |
| `status` | string | Filter by status: `active`, `inactive`, `error` |
| `page` | integer | Page number |
| `limit` | integer | Items per page |

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "asset-uuid-123",
      "name": "Production PostgreSQL",
      "source_type": "postgresql",
      "connection_id": "conn-uuid-456",
      "status": "active",
      "last_scan_at": "2026-01-15T10:35:00Z",
      "total_findings": 45,
      "open_findings": 12,
      "metadata": {
        "host": "localhost",
        "port": 5432,
        "database": "production"
      },
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-15T10:35:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 10
  }
}
```

### Create Asset

Creates a new asset.

```http
POST /api/v1/assets
Content-Type: application/json
```

**Request Body:**

```json
{
  "name": "Staging PostgreSQL",
  "source_type": "postgresql",
  "connection_id": "conn-uuid-789",
  "metadata": {
    "host": "staging-db.company.com",
    "port": 5432,
    "database": "staging"
  },
  "tags": ["staging", "postgresql"]
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "asset-uuid-new",
    "name": "Staging PostgreSQL",
    "source_type": "postgresql",
    "status": "active",
    "created_at": "2026-01-15T12:00:00Z"
  },
  "message": "Asset created successfully"
}
```

### Get Asset Details

Retrieves detailed information about an asset.

```http
GET /api/v1/assets/{asset_id}
```

### Update Asset

Updates an existing asset.

```http
PUT /api/v1/assets/{asset_id}
Content-Type: application/json
```

### Delete Asset

Deletes an asset.

```http
DELETE /api/v1/assets/{asset_id}
```

### Get Asset Scans

Retrieves scan history for an asset.

```http
GET /api/v1/assets/{asset_id}/scans
```

### Get Asset Findings

Retrieves findings for an asset.

```http
GET /api/v1/assets/{asset_id}/findings
```

---

## Lineage API

### Get Lineage Graph

Retrieves the complete lineage graph.

```http
GET /api/v1/lineage/v2
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `system_id` | string | Filter by system |
| `asset_id` | string | Filter by asset |
| `pii_type` | string | Filter by PII type |
| `depth` | integer | Graph depth (default: 3) |

**Response:**

```json
{
  "success": true,
  "data": {
    "nodes": [
      {
        "id": "system-prod",
        "type": "system",
        "label": "Production Environment",
        "properties": {
          "name": "Production",
          "environment": "production"
        }
      },
      {
        "id": "asset-users-db",
        "type": "asset",
        "label": "users table",
        "properties": {
          "name": "users",
          "type": "table",
          "system_id": "system-prod"
        }
      },
      {
        "id": "pii-aadhaar",
        "type": "pii_type",
        "label": "Aadhaar",
        "properties": {
          "type": "aadhaar",
          "category": "identity"
        }
      }
    ],
    "edges": [
      {
        "id": "edge-1",
        "source": "system-prod",
        "target": "asset-users-db",
        "type": "SYSTEM_OWNS_ASSET",
        "properties": {}
      },
      {
        "id": "edge-2",
        "source": "asset-users-db",
        "target": "pii-aadhaar",
        "type": "EXPOSES",
        "properties": {
          "column": "aadhaar_number",
          "confidence": 1.0
        }
      }
    ]
  }
}
```

### Get Asset Lineage

Retrieves lineage for a specific asset.

```http
GET /api/v1/lineage/asset/{asset_id}
```

### Get PII Distribution

Retrieves PII type distribution across the lineage.

```http
GET /api/v1/lineage/pii-distribution
```

**Response:**

```json
{
  "success": true,
  "data": {
    "distribution": [
      {
        "pii_type": "aadhaar",
        "count": 45,
        "percentage": 35.2,
        "assets": ["asset-1", "asset-2"]
      },
      {
        "pii_type": "pan",
        "count": 38,
        "percentage": 29.7,
        "assets": ["asset-1", "asset-3"]
      }
    ],
    "total_findings": 128
  }
}
```

### Get Impact Analysis

Analyzes the impact of changes to an asset.

```http
GET /api/v1/lineage/impact/{asset_id}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "asset_id": "asset-uuid-123",
    "upstream_dependencies": [],
    "downstream_dependencies": [
      {
        "asset_id": "asset-uuid-456",
        "relationship": "consumed_by",
        "pii_types": ["aadhaar", "pan"]
      }
    ],
    "affected_systems": ["system-prod"],
    "total_affected_assets": 3
  }
}
```

---

## Connections API

### List Connections

Retrieves all configured connections.

```http
GET /api/v1/connections
```

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "conn-uuid-456",
      "name": "Production PostgreSQL",
      "type": "postgresql",
      "status": "connected",
      "last_tested_at": "2026-01-15T10:00:00Z",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### Create Connection

Creates a new connection profile.

```http
POST /api/v1/connections
Content-Type: application/json
```

**Request Body:**

```json
{
  "name": "Production PostgreSQL",
  "type": "postgresql",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "production"
  },
  "credentials": {
    "username": "scanner",
    "password": "secure_password"
  }
}
```

### Test Connection

Tests a connection without saving.

```http
POST /api/v1/connections/test
Content-Type: application/json
```

**Request Body:**

```json
{
  "type": "postgresql",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "production"
  },
  "credentials": {
    "username": "scanner",
    "password": "secure_password"
  }
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "connected": true,
    "latency_ms": 15,
    "message": "Connection successful"
  }
}
```

### Get Connection Details

```http
GET /api/v1/connections/{connection_id}
```

### Update Connection

```http
PUT /api/v1/connections/{connection_id}
Content-Type: application/json
```

### Delete Connection

```http
DELETE /api/v1/connections/{connection_id}
```

---

## Compliance API

### Get Compliance Overview

Retrieves compliance dashboard data.

```http
GET /api/v1/compliance/overview
```

**Response:**

```json
{
  "success": true,
  "data": {
    "overall_score": 75,
    "dpdpa_compliance": {
      "status": "partial",
      "score": 72,
      "requirements": {
        "total": 10,
        "met": 7,
        "in_progress": 2,
        "not_met": 1
      }
    },
    "consent_tracking": {
      "total_findings": 150,
      "with_consent": 80,
      "without_consent": 45,
      "unknown": 25
    },
    "retention_compliance": {
      "compliant": 120,
      "expiring_soon": 20,
      "expired": 10
    }
  }
}
```

### Get DPDPA Report

Generates a DPDPA compliance report.

```http
GET /api/v1/compliance/dpdpa-report?format=pdf
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `format` | string | Report format: `json`, `pdf`, `csv` |
| `from_date` | string | Start date (ISO 8601) |
| `to_date` | string | End date (ISO 8601) |

### Update Consent Status

Updates the consent status for a finding.

```http
PUT /api/v1/compliance/consent/{finding_id}
Content-Type: application/json
```

**Request Body:**

```json
{
  "consent_status": "granted",
  "consent_date": "2026-01-15T10:00:00Z",
  "consent_type": "explicit",
  "consent_purpose": "service_provision",
  "data_principal_id": "user-123"
}
```

**Consent Status Values:**

- `granted`: Consent obtained
- `withdrawn`: Consent withdrawn
- `not_required`: Consent not required
- `unknown`: Consent status unknown

### Get Data Principal Requests

Retrieves data principal requests (access, correction, deletion).

```http
GET /api/v1/compliance/dpdpa/requests
```

### Create Data Principal Request

Creates a new data principal request.

```http
POST /api/v1/compliance/dpdpa/requests
Content-Type: application/json
```

**Request Body:**

```json
{
  "data_principal_id": "user-123",
  "request_type": "access",
  "description": "Request access to all personal data",
  "contact_email": "user@example.com"
}
```

---

## Remediation API

### Get Available Actions

Retrieves available remediation actions for findings.

```http
GET /api/v1/remediation/actions?finding_id=finding-uuid-001
```

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "action_id": "mask_partial",
      "name": "Partial Mask",
      "description": "Mask all but last 4 digits",
      "available": true,
      "risk_level": "low",
      "estimated_duration": "1m"
    },
    {
      "action_id": "delete",
      "name": "Delete",
      "description": "Permanently delete the data",
      "available": true,
      "risk_level": "high",
      "requires_approval": true,
      "estimated_duration": "5m"
    }
  ]
}
```

### Execute Remediation

Executes a remediation action.

```http
POST /api/v1/remediation/execute
Content-Type: application/json
```

**Request Body:**

```json
{
  "finding_id": "finding-uuid-001",
  "action": "mask",
  "strategy": "partial",
  "parameters": {
    "keep_last": 4,
    "mask_char": "*"
  },
  "reason": "DPDPA compliance - expired retention period",
  "requested_by": "user-uuid-123",
  "approval_required": false
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "remediation_id": "rem-uuid-789",
    "status": "pending_approval",
    "workflow_id": "workflow-uuid-012",
    "estimated_completion": "2026-01-15T11:05:00Z"
  },
  "message": "Remediation initiated successfully"
}
```

### Get Remediation History

Retrieves remediation history.

```http
GET /api/v1/remediation/history?finding_id=finding-uuid-001&page=1&limit=50
```

### Preview Remediation

Previews the effect of a remediation action.

```http
POST /api/v1/remediation/preview
Content-Type: application/json
```

**Request Body:**

```json
{
  "finding_id": "finding-uuid-001",
  "action": "mask",
  "strategy": "partial"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "finding_id": "finding-uuid-001",
    "current_value": "1234-5678-9012",
    "preview_value": "****-****-9012",
    "rows_affected": 1,
    "action": "mask",
    "strategy": "partial"
  }
}
```

### Cancel Remediation

Cancels a pending remediation.

```http
POST /api/v1/remediation/{remediation_id}/cancel
```

---

## Analytics API

### Get Dashboard Stats

Retrieves dashboard statistics.

```http
GET /api/v1/analytics/dashboard
```

**Response:**

```json
{
  "success": true,
  "data": {
    "summary": {
      "total_findings": 150,
      "open_findings": 45,
      "resolved_findings": 95,
      "false_positives": 10,
      "total_assets": 12,
      "total_scans": 25
    },
    "risk_score": 72,
    "severity_breakdown": {
      "critical": 5,
      "high": 15,
      "medium": 45,
      "low": 85
    },
    "pii_type_distribution": {
      "aadhaar": 45,
      "pan": 38,
      "email": 25,
      "phone": 20,
      "credit_card": 12,
      "passport": 10
    },
    "recent_activity": [
      {
        "type": "scan_completed",
        "asset_name": "Production PostgreSQL",
        "timestamp": "2026-01-15T10:35:00Z",
        "details": "Found 12 new findings"
      }
    ],
    "trends": {
      "findings_last_7_days": [12, 8, 15, 10, 5, 20, 18],
      "risk_score_last_7_days": [75, 74, 72, 73, 72, 71, 72]
    }
  }
}
```

### Get Risk Score

Calculates the current risk score.

```http
GET /api/v1/analytics/risk-score
```

**Response:**

```json
{
  "success": true,
  "data": {
    "overall_score": 72,
    "category": "medium",
    "breakdown": {
      "exposure": 65,
      "severity": 80,
      "volume": 70,
      "remediation": 75
    },
    "factors": [
      {
        "name": "Critical findings",
        "impact": -10,
        "description": "5 critical severity findings"
      },
      {
        "name": "Remediation rate",
        "impact": +5,
        "description": "63% remediation rate"
      }
    ]
  }
}
```

### Get PII Trends

Retrieves PII detection trends over time.

```http
GET /api/v1/analytics/trends?days=30&group_by=day
```

### Get Asset Statistics

Retrieves statistics for a specific asset.

```http
GET /api/v1/analytics/assets/{asset_id}/stats
```

### Generate Report

Generates a custom analytics report.

```http
POST /api/v1/analytics/reports
Content-Type: application/json
```

**Request Body:**

```json
{
  "name": "Monthly Compliance Report",
  "type": "compliance",
  "format": "pdf",
  "date_range": {
    "from": "2026-01-01",
    "to": "2026-01-31"
  },
  "include_charts": true
}
```

---

## WebSocket API

### Connect

Establish a WebSocket connection for real-time updates.

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('Connected to ARC-Hawk');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};
```

### Subscribe to Channels

Subscribe to specific event channels:

```javascript
// Subscribe to scan updates
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'scan:scan-uuid-456'
}));

// Subscribe to all findings
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'findings'
}));

// Subscribe to system events
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'system'
}));
```

### Message Format

**Incoming Messages:**

```json
{
  "type": "scan_progress",
  "channel": "scan:scan-uuid-456",
  "timestamp": "2026-01-15T10:32:00Z",
  "data": {
    "scan_id": "scan-uuid-456",
    "progress": 65,
    "status": "running",
    "files_scanned": 650,
    "files_total": 1000,
    "findings_count": 12
  }
}
```

**Message Types:**

- `scan_progress`: Scan progress updates
- `scan_completed`: Scan completion notification
- `finding_detected`: New finding detected
- `finding_updated`: Finding status changed
- `system_health`: System health status
- `error`: Error notification

### Unsubscribe

```javascript
ws.send(JSON.stringify({
  action: 'unsubscribe',
  channel: 'scan:scan-uuid-456'
}));
```

---

## Rate Limiting

> **Note**: Rate limiting is not yet implemented.

When implemented:

- **Limit**: 1000 requests per minute per API key
- **Headers**:
  - `X-RateLimit-Limit`: Maximum requests allowed
  - `X-RateLimit-Remaining`: Remaining requests
  - `X-RateLimit-Reset`: Timestamp when limit resets

---

## Pagination

List endpoints support pagination:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `limit` | integer | 50 | Items per page (max: 100) |

**Response includes:**

```json
{
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 200,
    "total_pages": 4,
    "has_next": true,
    "has_prev": false
  }
}
```

---

## Filtering

Most list endpoints support filtering:

```http
GET /api/v1/findings?status=open&severity=high&pii_type=aadhaar
```

**Filter Operators:**

- `field=value`: Equal
- `field__ne=value`: Not equal
- `field__gt=value`: Greater than
- `field__gte=value`: Greater than or equal
- `field__lt=value`: Less than
- `field__lte=value`: Less than or equal
- `field__in=value1,value2`: In list
- `field__contains=value`: Contains substring

---

## Sorting

```http
GET /api/v1/findings?sort_by=detected_at&sort_order=desc
```

**Sortable Fields:**
- `detected_at`: Detection timestamp
- `updated_at`: Last update timestamp
- `severity`: Severity level
- `pii_type`: PII type
- `confidence`: Confidence score

---

## SDK Examples

### JavaScript/TypeScript

```typescript
import { ArcHawkClient } from '@arc-hawk/sdk';

const client = new ArcHawkClient({
  baseUrl: 'http://localhost:8080/api/v1',
  apiKey: 'your-api-key'
});

// Trigger a scan
const scan = await client.scans.trigger({
  assetId: 'asset-uuid-123',
  scanType: 'full'
});

// List findings
const findings = await client.findings.list({
  status: 'open',
  severity: 'high'
});

// Get real-time updates
client.websocket.connect();
client.websocket.subscribe('scan:scan-uuid-456', (message) => {
  console.log('Progress:', message.data.progress);
});
```

### Python

```python
from arc_hawk import ArcHawkClient

client = ArcHawkClient(
    base_url='http://localhost:8080/api/v1',
    api_key='your-api-key'
)

# Trigger a scan
scan = client.scans.trigger(
    asset_id='asset-uuid-123',
    scan_type='full'
)

# List findings
findings = client.findings.list(
    status='open',
    severity='high'
)

# Get dashboard stats
stats = client.analytics.get_dashboard_stats()
print(f"Total findings: {stats['summary']['total_findings']}")
```

### Go

```go
import (
    "github.com/your-org/arc-hawk-go-sdk"
)

client := archawk.NewClient("http://localhost:8080/api/v1", "your-api-key")

// Trigger a scan
scan, err := client.Scans.Trigger(context.Background(), archawk.TriggerScanRequest{
    AssetID:  "asset-uuid-123",
    ScanType: "full",
})

// List findings
findings, err := client.Findings.List(context.Background(), archawk.ListFindingsRequest{
    Status:   "open",
    Severity: "high",
})
```

---

## Changelog

See [CHANGELOG.md](../../CHANGELOG.md) for API version history.

---

---

## Custom Patterns API

Manage user-defined PII patterns that extend the built-in 11 PII types.

> Patterns are validated against ReDoS heuristics at submission time. Patterns longer than 512 characters or containing nested quantifiers are rejected.

### List Patterns

```http
GET /api/v1/patterns
```

**Response:**
```json
{
  "patterns": [
    {
      "id": "pattern-uuid-123",
      "name": "Employee ID",
      "regex": "EMP-\\d{6}",
      "description": "Internal employee identifier format",
      "created_at": "2026-04-09T08:00:00Z"
    }
  ]
}
```

### Create Pattern

```http
POST /api/v1/patterns
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Employee ID",
  "regex": "EMP-\\d{6}",
  "description": "Internal employee identifier format"
}
```

**Validation errors (400):**
- Pattern exceeds 512 characters
- Pattern contains nested quantifiers (ReDoS risk)
- Pattern fails `re.compile()` (invalid regex)
- Name is blank or duplicate

### Delete Pattern

```http
DELETE /api/v1/patterns/:id
```

Returns 204 on success. Patterns are removed from subsequent scans immediately (hot-reload — no restart needed).

---

## Discovery API

Asset-level risk inventory with time-series scoring.

### Get Discovery Summary

```http
GET /api/v1/discovery
```

Returns per-source finding counts, risk scores, and spike indicators.

**Response:**
```json
{
  "assets": [
    {
      "asset_id": "asset-uuid-123",
      "asset_type": "database",
      "source_name": "prod-postgres",
      "risk_score": 87,
      "finding_count": 1420,
      "spike_detected": true,
      "last_scanned": "2026-04-09T07:00:00Z"
    }
  ],
  "heatmap": {
    "rows": [...],
    "columns": ["PAN", "Aadhaar", "Email", "Phone"]
  }
}
```

### Get Asset Risk History

```http
GET /api/v1/discovery/:asset_id/history
```

Returns time-series risk scores for the given asset.

---

---

## Backend-Only Routes (No Frontend UI)

These endpoints are fully implemented in the backend but have no corresponding frontend page or service call yet. They are available for CLI tooling, integrations, or future UI work.

| Route | Method | Handler | Purpose |
|-------|--------|---------|---------|
| `/discovery/inventory` | GET | `InventoryHandler.ListInventory` | Full asset inventory with source mapping |
| `/discovery/inventory/:assetId` | GET | `InventoryHandler.GetAssetInventory` | Per-asset inventory detail |
| `/discovery/snapshots` | GET | `SnapshotHandler.ListSnapshots` | List point-in-time snapshots |
| `/discovery/snapshots` | POST (trigger) | `SnapshotHandler.TriggerSnapshot` | Trigger a new snapshot |
| `/discovery/risk/overview` | GET | `RiskHandler.GetRiskOverview` | Aggregate risk overview for all assets |
| `/discovery/risk/hotspots` | GET | `RiskHandler.GetRiskHotspots` | Top-N riskiest assets |
| `/discovery/risk/scores/:assetId` | GET | `RiskHandler.GetAssetRiskHistory` | Time-series risk score for an asset |
| `/discovery/drift/since/:snapshotId` | GET | `DriftHandler.GetDriftSince` | Drift events since a snapshot |
| `/discovery/drift/timeline` | GET | `DriftHandler.GetDriftTimeline` | Full drift timeline |
| `/discovery/reports/generate` | POST | `ReportHandler.GenerateReport` | Generate a discovery report |
| `/discovery/reports` | GET | `ReportHandler.ListReports` | List generated reports |
| `/discovery/reports/:id/download` | GET | `ReportHandler.DownloadReport` | Download report file |
| `/discovery/glossary` | GET | `GlossaryHandler.GetGlossary` | Data glossary / term registry |
| `/assets/bulk-tag` | POST | `AssetHandler.BulkTagAssets` | Batch tag multiple assets |
| `/dataset/golden` | GET | `DatasetHandler.GetGoldenDataset` | Golden dataset for classifier training |
| `/lineage/sync` | POST | `LineageHandler.SyncLineage` | Manual lineage sync trigger |
| `/lineage/graph/semantic` | GET | `GraphHandler.GetSemanticGraph` | Full semantic lineage graph |
| `/masking/mask-asset` | POST | `MaskingHandler.MaskAsset` | Mask all PII fields in an asset |
| `/masking/status/:assetId` | GET | `MaskingHandler.GetMaskingStatus` | Masking status for an asset |
| `/masking/audit/:assetId` | GET | `MaskingHandler.GetMaskingAuditLog` | Masking audit log |
| `/remediation/history/:assetId` | GET | `RemediationHandler.GetRemediationHistory` | Per-asset remediation history |
| `/remediation/actions/:findingId` | GET | `RemediationHandler.GetRemediationActions` | Actions for a specific finding |
| `/remediation/rollback/:id` | POST | `RemediationHandler.RollbackRemediation` | Roll back a remediation action |
| `/remediation/export` | GET | `ExportHandler.ExportReport` | Export remediation report |
| `/remediation/escalation/preview` | GET | `EscalationHandler.Preview` | Preview escalation workflow |
| `/remediation/escalation/run` | POST | `EscalationHandler.Run` | Execute escalation workflow |
| `/remediation/sops` | GET | `SOPRegistry` | List all Standard Operating Procedures |
| `/remediation/sops/:issue_type` | GET | `SOPRegistry` | SOP for a specific issue type |
| `/scans/:id/delta` | GET | `ScanTriggerHandler.GetScanDelta` | Delta between this scan and previous |
| `/scans/:id/complete` | POST | `ScanStatusHandler.CompleteScan` | Mark a scan as complete (SDK use) |
| `/connections/sync` | POST | `ConnectionSyncHandler.SyncToScanner` | Sync connections to scanner config |
| `/connections/sync/validate` | GET | `ConnectionSyncHandler.ValidateSync` | Validate scanner config sync |
| `/connections/scans/scan-all` | POST | `ScanOrchestrationHandler.ScanAllAssets` | Trigger scan across all connections |
| `/analytics/risk-distribution` | GET | *(not yet implemented in backend)* | Risk distribution by tier |
| `/audit/resource/:resourceType/:resourceId` | GET | `AuditHandler.GetResourceHistory` | Audit history for a specific resource |
| `/audit/recent` | GET | `AuditHandler.GetRecentActivity` | Recent audit activity across all resources |

> **Note**: All routes above require a valid `Authorization: Bearer <token>` header.
> Routes listed as "no frontend" are fully functional via direct API calls or CLI tools.

---

**API Version**: 1.0  
**Last Updated**: April 2026  
**Documentation Version**: 3.1.0
