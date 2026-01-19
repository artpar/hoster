-- Remove proxy_port column from deployments
-- Note: SQLite doesn't support DROP COLUMN directly before 3.35
-- This migration assumes SQLite 3.35+ or will need to recreate the table

ALTER TABLE deployments DROP COLUMN proxy_port;
