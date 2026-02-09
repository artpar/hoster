-- Migration 010 down: Revert to TEXT PKs
-- This is destructive - integer IDs are lost, reference_ids become IDs again.

PRAGMA foreign_keys = OFF;

-- Reverse order of creation

-- cloud_provisions
CREATE TABLE cloud_provisions_old (
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
    completed_at TEXT
);
INSERT INTO cloud_provisions_old SELECT cp.reference_id, u.reference_id, cc.reference_id,
    cp.provider, cp.status, cp.instance_name, cp.region, cp.size,
    cp.provider_instance_id, cp.public_ip, cp.node_id, cp.ssh_key_id,
    cp.current_step, cp.error_message, cp.created_at, cp.updated_at, cp.completed_at
    FROM cloud_provisions cp
    JOIN users u ON u.id = cp.creator_id
    JOIN cloud_credentials cc ON cc.id = cp.credential_id;
DROP TABLE cloud_provisions;
ALTER TABLE cloud_provisions_old RENAME TO cloud_provisions;

-- cloud_credentials
CREATE TABLE cloud_credentials_old (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    credentials_encrypted BLOB NOT NULL,
    default_region TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(creator_id, name)
);
INSERT INTO cloud_credentials_old SELECT cc.reference_id, u.reference_id,
    cc.name, cc.provider, cc.credentials_encrypted, cc.default_region, cc.created_at, cc.updated_at
    FROM cloud_credentials cc
    JOIN users u ON u.id = cc.creator_id;
DROP TABLE cloud_credentials;
ALTER TABLE cloud_credentials_old RENAME TO cloud_credentials;

-- nodes
CREATE TABLE nodes_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    creator_id TEXT NOT NULL,
    ssh_host TEXT NOT NULL,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    ssh_user TEXT NOT NULL,
    ssh_key_id TEXT,
    docker_socket TEXT DEFAULT '/var/run/docker.sock',
    status TEXT NOT NULL DEFAULT 'offline',
    capabilities TEXT DEFAULT '["standard"]',
    capacity_cpu_cores REAL DEFAULT 0,
    capacity_memory_mb INTEGER DEFAULT 0,
    capacity_disk_mb INTEGER DEFAULT 0,
    capacity_cpu_used REAL DEFAULT 0,
    capacity_memory_used_mb INTEGER DEFAULT 0,
    capacity_disk_used_mb INTEGER DEFAULT 0,
    location TEXT DEFAULT '',
    last_health_check TEXT,
    error_message TEXT DEFAULT '',
    provider_type TEXT DEFAULT 'manual',
    provision_id TEXT DEFAULT '',
    base_domain TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(creator_id, name)
);
INSERT INTO nodes_old SELECT n.reference_id, n.name, u.reference_id,
    n.ssh_host, n.ssh_port, n.ssh_user, COALESCE(sk.reference_id, ''),
    n.docker_socket, n.status, n.capabilities,
    n.capacity_cpu_cores, n.capacity_memory_mb, n.capacity_disk_mb,
    n.capacity_cpu_used, n.capacity_memory_used_mb, n.capacity_disk_used_mb,
    n.location, n.last_health_check, n.error_message,
    n.provider_type, n.provision_id, n.base_domain,
    n.created_at, n.updated_at
    FROM nodes n
    JOIN users u ON u.id = n.creator_id
    LEFT JOIN ssh_keys sk ON sk.id = n.ssh_key_id;
DROP TABLE nodes;
ALTER TABLE nodes_old RENAME TO nodes;

-- usage_events
CREATE TABLE usage_events_old (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    quantity INTEGER DEFAULT 1,
    metadata TEXT,
    timestamp TEXT NOT NULL,
    reported_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO usage_events_old SELECT ue.reference_id, u.reference_id,
    ue.event_type, ue.resource_id, ue.resource_type,
    ue.quantity, ue.metadata, ue.timestamp, ue.reported_at, ue.created_at
    FROM usage_events ue
    JOIN users u ON u.id = ue.user_id;
DROP TABLE usage_events;
ALTER TABLE usage_events_old RENAME TO usage_events;

-- container_events
CREATE TABLE container_events_old (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    type TEXT NOT NULL,
    container TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO container_events_old SELECT ce.reference_id, d.reference_id,
    ce.type, ce.container, ce.message, ce.timestamp, ce.created_at
    FROM container_events ce
    JOIN deployments d ON d.id = ce.deployment_id;
DROP TABLE container_events;
ALTER TABLE container_events_old RENAME TO container_events;

-- deployments
CREATE TABLE deployments_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL,
    template_version TEXT NOT NULL DEFAULT '',
    customer_id TEXT NOT NULL,
    node_id TEXT DEFAULT '',
    status TEXT NOT NULL,
    variables TEXT,
    domains TEXT,
    containers TEXT,
    resources_cpu_cores REAL NOT NULL DEFAULT 0,
    resources_memory_mb INTEGER NOT NULL DEFAULT 0,
    resources_disk_mb INTEGER NOT NULL DEFAULT 0,
    proxy_port INTEGER,
    error_message TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    started_at TEXT,
    stopped_at TEXT,
    FOREIGN KEY (template_id) REFERENCES templates(id)
);
INSERT INTO deployments_old SELECT d.reference_id, d.name, t.reference_id, d.template_version,
    u.reference_id, d.node_id, d.status, d.variables, d.domains, d.containers,
    d.resources_cpu_cores, d.resources_memory_mb, d.resources_disk_mb,
    d.proxy_port, d.error_message, d.created_at, d.updated_at, d.started_at, d.stopped_at
    FROM deployments d
    JOIN templates t ON t.id = d.template_id
    JOIN users u ON u.id = d.customer_id;
DROP TABLE deployments;
ALTER TABLE deployments_old RENAME TO deployments;

-- templates
CREATE TABLE templates_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    version TEXT NOT NULL,
    compose_spec TEXT NOT NULL,
    variables TEXT,
    config_files TEXT,
    resources_cpu_cores REAL NOT NULL DEFAULT 0,
    resources_memory_mb INTEGER NOT NULL DEFAULT 0,
    resources_disk_mb INTEGER NOT NULL DEFAULT 0,
    price_monthly_cents INTEGER DEFAULT 0,
    category TEXT DEFAULT '',
    tags TEXT,
    required_capabilities TEXT DEFAULT '[]',
    published INTEGER DEFAULT 0,
    creator_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO templates_old SELECT t.reference_id, t.name, t.slug, t.description, t.version,
    t.compose_spec, t.variables, t.config_files,
    t.resources_cpu_cores, t.resources_memory_mb, t.resources_disk_mb,
    t.price_monthly_cents, t.category, t.tags, t.required_capabilities,
    t.published, u.reference_id, t.created_at, t.updated_at
    FROM templates t
    JOIN users u ON u.id = t.creator_id;
DROP TABLE templates;
ALTER TABLE templates_old RENAME TO templates;

-- ssh_keys
CREATE TABLE ssh_keys_old (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL,
    name TEXT NOT NULL,
    private_key_encrypted BLOB NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(creator_id, name)
);
INSERT INTO ssh_keys_old SELECT sk.reference_id, u.reference_id,
    sk.name, sk.private_key_encrypted, sk.fingerprint, sk.created_at
    FROM ssh_keys sk
    JOIN users u ON u.id = sk.creator_id;
DROP TABLE ssh_keys;
ALTER TABLE ssh_keys_old RENAME TO ssh_keys;

-- Drop users table
DROP TABLE users;

PRAGMA foreign_keys = ON;
