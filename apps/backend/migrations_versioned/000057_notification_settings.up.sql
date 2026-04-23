CREATE TABLE IF NOT EXISTS notification_settings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL UNIQUE,
    email_enabled       BOOLEAN NOT NULL DEFAULT FALSE,
    email_recipients    TEXT[] NOT NULL DEFAULT '{}',
    slack_enabled       BOOLEAN NOT NULL DEFAULT FALSE,
    slack_webhook_url   TEXT,
    notify_on_scan_complete   BOOLEAN NOT NULL DEFAULT TRUE,
    notify_on_high_severity   BOOLEAN NOT NULL DEFAULT TRUE,
    notify_on_stale_connector BOOLEAN NOT NULL DEFAULT FALSE,
    severity_threshold  TEXT NOT NULL DEFAULT 'High' CHECK (severity_threshold IN ('Critical', 'High', 'Medium', 'Low')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
