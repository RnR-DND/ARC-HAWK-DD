CREATE TABLE IF NOT EXISTS obligation_regressions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id),
    scan_id      TEXT        NOT NULL,
    pii_category TEXT        NOT NULL,
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, pii_category)
);

CREATE INDEX idx_obligation_regressions_tenant ON obligation_regressions(tenant_id, detected_at DESC);
