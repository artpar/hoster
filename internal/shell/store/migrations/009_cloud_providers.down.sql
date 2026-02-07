-- Rollback migration 009: Remove cloud provider provisioning and domain management

DROP INDEX IF EXISTS idx_cloud_provisions_credential;
DROP INDEX IF EXISTS idx_cloud_provisions_status;
DROP INDEX IF EXISTS idx_cloud_provisions_creator;
DROP INDEX IF EXISTS idx_cloud_credentials_creator;

DROP TABLE IF EXISTS cloud_provisions;
DROP TABLE IF EXISTS cloud_credentials;

-- Note: SQLite does not support DROP COLUMN directly.
-- The columns provider_type, provision_id, base_domain on nodes table
-- will remain but are harmless with their defaults.
