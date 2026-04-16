-- Migration: 000041_gro_cross_border (rollback)

ALTER TABLE tenants DROP COLUMN IF EXISTS gro_name;
ALTER TABLE tenants DROP COLUMN IF EXISTS gro_email;
ALTER TABLE tenants DROP COLUMN IF EXISTS gro_phone;
ALTER TABLE tenants DROP COLUMN IF EXISTS is_significant_data_fiduciary;

ALTER TABLE assets DROP COLUMN IF EXISTS data_residency_country;
