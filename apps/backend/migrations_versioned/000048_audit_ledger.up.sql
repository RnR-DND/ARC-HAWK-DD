CREATE TABLE IF NOT EXISTS audit_ledger (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id),
    event_type    TEXT        NOT NULL,
    actor_id      UUID        REFERENCES users(id),
    actor_email   TEXT,
    resource_id   TEXT,
    resource_type TEXT,
    payload       JSONB       NOT NULL DEFAULT '{}',
    ip_address    INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_ledger_tenant_created   ON audit_ledger(tenant_id, created_at DESC);
CREATE INDEX idx_audit_ledger_event_type        ON audit_ledger(event_type);
CREATE INDEX idx_audit_ledger_resource          ON audit_ledger(tenant_id, resource_type, resource_id);

COMMENT ON TABLE audit_ledger IS 'Append-only audit trail. Never UPDATE or DELETE rows. Required for DPDP Act 2023 compliance evidence.';
