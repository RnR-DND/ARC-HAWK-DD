-- Migration: 000041_gro_cross_border
-- Description: DPDP Act 2023 Sec 11 (GRO) + Sec 10 SDF flag + Sec 12 (cross-border data residency)

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS gro_name VARCHAR(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS gro_email VARCHAR(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS gro_phone VARCHAR(50);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS is_significant_data_fiduciary BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE assets ADD COLUMN IF NOT EXISTS data_residency_country VARCHAR(10) DEFAULT 'IN';
