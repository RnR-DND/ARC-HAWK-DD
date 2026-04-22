# ARC-HAWK Webhook Reference

ARC-HAWK delivers real-time event notifications to a configured HTTP endpoint using HMAC-signed POST requests.

---

## Configuration

Set these environment variables on the backend service:

| Variable | Description |
|----------|-------------|
| `WEBHOOK_URL` | Full URL of your receiver endpoint |
| `WEBHOOK_SECRET` | Shared secret for HMAC-SHA256 signing (min 32 bytes) |

---

## Signing

Every webhook delivery carries the header:

```
X-ARC-Signature: sha256=<lowercase-hex-digest>
```

The digest is `HMAC-SHA256(raw_request_body_bytes, WEBHOOK_SECRET)`.

**Verify before processing:**

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "os"
)

func verifySignature(body []byte, sigHeader string) bool {
    secret := []byte(os.Getenv("WEBHOOK_SECRET"))
    mac := hmac.New(sha256.New, secret)
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sigHeader))
}
```

Never use `==` for comparison — always use `hmac.Equal` (constant-time) to prevent timing attacks.

---

## Retry Policy

- **Initial delivery**: Attempted immediately after the event is produced.
- **Retries**: Exponential backoff — 30 s, 2 min, 8 min, 30 min, 2 h, 8 h, 24 h.
- **Max duration**: Events are retried for up to **24 hours** from the original event time.
- **Success criteria**: Your receiver must return HTTP `2xx` within 10 s.
- **Non-retryable**: HTTP `4xx` from your receiver causes immediate abandonment (except `429` which is retried).

---

## Idempotency

Each event envelope includes `event_id` — a stable UUID generated when the event is first produced. Your receiver should store processed `event_id` values and skip duplicates, as retries deliver the same `event_id`.

---

## Envelope Schema

All events share a common envelope:

```json
{
  "event_id":   "550e8400-e29b-41d4-a716-446655440000",
  "event":      "<event-type>",
  "tenant_id":  "tenant-uuid",
  "timestamp":  "2026-04-22T10:30:00Z",
  "data":       { ... }
}
```

---

## Event Catalog

### `scan.started`

Fired when a scan transitions from `pending` to `running`.

```json
{
  "event_id": "evt-uuid",
  "event": "scan.started",
  "tenant_id": "tenant-uuid",
  "timestamp": "2026-04-22T10:00:00Z",
  "data": {
    "scan_id": "scan-uuid",
    "scan_name": "nightly-pii-scan",
    "connection_id": "conn-uuid",
    "connection_name": "prod-postgres",
    "source_type": "postgresql",
    "triggered_by": "user-uuid"
  }
}
```

---

### `scan.progress`

Fired at each 10% progress increment during an active scan.

```json
{
  "event_id": "evt-uuid",
  "event": "scan.progress",
  "tenant_id": "tenant-uuid",
  "timestamp": "2026-04-22T10:05:00Z",
  "data": {
    "scan_id": "scan-uuid",
    "scan_name": "nightly-pii-scan",
    "progress": 40,
    "findings_so_far": 312,
    "assets_scanned": 8
  }
}
```

`progress` is an integer 0–100.

---

### `scan.completed`

Fired when a scan finishes (status `completed` or `failed`).

```json
{
  "event_id": "evt-uuid",
  "event": "scan.completed",
  "tenant_id": "tenant-uuid",
  "timestamp": "2026-04-22T10:30:00Z",
  "data": {
    "scan_id": "scan-uuid",
    "scan_name": "nightly-pii-scan",
    "status": "completed",
    "total_findings": 1888,
    "total_assets": 24,
    "duration_seconds": 1740,
    "error_message": null
  }
}
```

When `status` is `failed`, `error_message` contains the failure reason and `total_findings` may be partial.

---

### `finding.created`

Fired in near-real-time when a High or Critical severity finding is ingested during a scan. Low/Medium findings are batched and not individually surfaced via webhooks.

```json
{
  "event_id": "evt-uuid",
  "event": "finding.created",
  "tenant_id": "tenant-uuid",
  "timestamp": "2026-04-22T10:12:00Z",
  "data": {
    "finding_id": "finding-uuid",
    "scan_id": "scan-uuid",
    "asset_name": "users_table",
    "asset_path": "myapp.public.users",
    "field": "aadhaar_number",
    "pii_type": "IN_AADHAAR",
    "confidence": 1.0,
    "risk_score": 95,
    "risk": "Critical",
    "source_type": "Database"
  }
}
```

`risk` values: `Critical` | `High` | `Medium` | `Low` | `Info`

---

### `remediation.applied`

Fired after a remediation action is successfully executed.

```json
{
  "event_id": "evt-uuid",
  "event": "remediation.applied",
  "tenant_id": "tenant-uuid",
  "timestamp": "2026-04-22T11:00:00Z",
  "data": {
    "action_id": "action-uuid",
    "finding_id": "finding-uuid",
    "asset_name": "users_table",
    "action_type": "mask",
    "field": "aadhaar_number",
    "executed_by": "user-uuid",
    "rollback_available": true
  }
}
```

`action_type` values: `mask` | `delete` | `quarantine`

---

## Go Receiver Example

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
)

type WebhookEnvelope struct {
    EventID   string          `json:"event_id"`
    Event     string          `json:"event"`
    TenantID  string          `json:"tenant_id"`
    Timestamp string          `json:"timestamp"`
    Data      json.RawMessage `json:"data"`
}

func verifySignature(body []byte, sigHeader string) bool {
    secret := []byte(os.Getenv("WEBHOOK_SECRET"))
    mac := hmac.New(sha256.New, secret)
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sigHeader))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB max
    if err != nil {
        http.Error(w, "read error", http.StatusBadRequest)
        return
    }

    sig := r.Header.Get("X-ARC-Signature")
    if !verifySignature(body, sig) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    var envelope WebhookEnvelope
    if err := json.Unmarshal(body, &envelope); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    // Idempotency: check if event_id was already processed
    // (implementation depends on your store)
    log.Printf("received event: %s id=%s", envelope.Event, envelope.EventID)

    switch envelope.Event {
    case "scan.completed":
        // handle scan completion
    case "finding.created":
        // alert on high/critical findings
    case "remediation.applied":
        // update your records
    }

    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, `{"status":"ok"}`)
}

func main() {
    http.HandleFunc("/arc-hawk-events", webhookHandler)
    log.Fatal(http.ListenAndServe(":9000", nil))
}
```

---

## Related Documents

- [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) — Full integration walkthrough
