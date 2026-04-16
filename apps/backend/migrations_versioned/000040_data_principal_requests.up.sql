-- Migration: 000040_data_principal_requests
-- Description: DPDP Act 2023 Sec 7 — Data Principal Rights request tracking table

CREATE TABLE IF NOT EXISTS data_principal_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    request_type VARCHAR(50) NOT NULL CHECK (request_type IN ('ACCESS', 'CORRECTION', 'ERASURE', 'NOMINATION', 'GRIEVANCE')),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'REJECTED')),
    data_principal_id VARCHAR(255) NOT NULL,
    data_principal_email VARCHAR(255),
    request_details JSONB,
    response_details JSONB,
    due_date TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dpr_tenant_status ON data_principal_requests(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_dpr_tenant_created ON data_principal_requests(tenant_id, created_at);
