-- Migration 009: Add cloud provider provisioning and domain management support
-- Phase 1: Cloud Provider tables for AWS, DigitalOcean, Hetzner integration
-- Phase 2: Domain management extensions for custom domains + per-node base domains

-- Cloud credentials table (encrypted like ssh_keys)
CREATE TABLE IF NOT EXISTS cloud_credentials (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,                -- 'aws', 'digitalocean', 'hetzner'
    credentials_encrypted BLOB NOT NULL,   -- AES-256-GCM (same as SSH keys)
    default_region TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(creator_id, name)
);

-- Cloud provisions table (tracks async provisioning jobs)
CREATE TABLE IF NOT EXISTS cloud_provisions (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL,
    credential_id TEXT NOT NULL REFERENCES cloud_credentials(id),
    provider TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    instance_name TEXT NOT NULL,
    region TEXT NOT NULL,
    size TEXT NOT NULL,
    provider_instance_id TEXT DEFAULT '',
    public_ip TEXT DEFAULT '',
    node_id TEXT DEFAULT '',
    ssh_key_id TEXT DEFAULT '',
    current_step TEXT DEFAULT '',
    error_message TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    completed_at TEXT,
    FOREIGN KEY (credential_id) REFERENCES cloud_credentials(id) ON DELETE RESTRICT
);

-- Extend nodes table for cloud provider tracking
ALTER TABLE nodes ADD COLUMN provider_type TEXT DEFAULT 'manual';
ALTER TABLE nodes ADD COLUMN provision_id TEXT DEFAULT '';
ALTER TABLE nodes ADD COLUMN base_domain TEXT DEFAULT '';

-- Indexes
CREATE INDEX IF NOT EXISTS idx_cloud_credentials_creator ON cloud_credentials(creator_id);
CREATE INDEX IF NOT EXISTS idx_cloud_provisions_creator ON cloud_provisions(creator_id);
CREATE INDEX IF NOT EXISTS idx_cloud_provisions_status ON cloud_provisions(status);
CREATE INDEX IF NOT EXISTS idx_cloud_provisions_credential ON cloud_provisions(credential_id);
