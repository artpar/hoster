-- Migration 005: Add nodes and ssh_keys tables for creator worker nodes
-- Following specs/domain/node.md

-- SSH keys table (stored separately for security)
CREATE TABLE IF NOT EXISTS ssh_keys (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL,
    name TEXT NOT NULL,
    private_key_encrypted BLOB NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(creator_id, name)
);

-- Nodes table
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    creator_id TEXT NOT NULL,
    ssh_host TEXT NOT NULL,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    ssh_user TEXT NOT NULL,
    ssh_key_id TEXT,
    docker_socket TEXT DEFAULT '/var/run/docker.sock',
    status TEXT NOT NULL DEFAULT 'offline',
    capabilities TEXT DEFAULT '["standard"]',  -- JSON array
    capacity_cpu_cores REAL DEFAULT 0,
    capacity_memory_mb INTEGER DEFAULT 0,
    capacity_disk_mb INTEGER DEFAULT 0,
    capacity_cpu_used REAL DEFAULT 0,
    capacity_memory_used_mb INTEGER DEFAULT 0,
    capacity_disk_used_mb INTEGER DEFAULT 0,
    location TEXT DEFAULT '',
    last_health_check TEXT,
    error_message TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(creator_id, name),
    FOREIGN KEY (ssh_key_id) REFERENCES ssh_keys(id) ON DELETE SET NULL
);

-- Add required_capabilities to templates table
ALTER TABLE templates ADD COLUMN required_capabilities TEXT DEFAULT '[]';

-- Note: node_id column already exists in deployments table from migration 001
-- No need to add it again

-- Index for node scheduling queries
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_creator ON nodes(creator_id);

-- Index for deployments by node
CREATE INDEX IF NOT EXISTS idx_deployments_node ON deployments(node_id);
