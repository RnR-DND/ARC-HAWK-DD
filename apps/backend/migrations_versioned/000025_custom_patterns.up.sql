-- Custom PII patterns — user-defined regex rules for PII detection
CREATE TABLE IF NOT EXISTS custom_patterns (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        VARCHAR(128) NOT NULL,
    display_name VARCHAR(256) NOT NULL,
    regex       TEXT NOT NULL,
    category    VARCHAR(64) NOT NULL DEFAULT 'Custom',
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_by  VARCHAR(256),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_custom_patterns_tenant ON custom_patterns(tenant_id);
CREATE INDEX IF NOT EXISTS idx_custom_patterns_active ON custom_patterns(tenant_id, is_active);
