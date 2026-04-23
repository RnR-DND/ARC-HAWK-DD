CREATE TABLE IF NOT EXISTS feedback_corrections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    finding_id      UUID REFERENCES findings(id) ON DELETE SET NULL,
    pattern_id      UUID REFERENCES custom_patterns(id) ON DELETE SET NULL,
    pattern_code    TEXT NOT NULL,
    column_name     TEXT,
    table_name      TEXT,
    schema_name     TEXT,
    connector_type  TEXT,
    correction_type TEXT NOT NULL CHECK (correction_type IN ('false_positive','false_negative','confirmed')),
    corrected_by    UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_feedback_pattern_tenant ON feedback_corrections(pattern_code, tenant_id);
CREATE INDEX idx_feedback_created ON feedback_corrections(created_at DESC);
