-- Migration 010: Normalize schema — add users table + integer PKs
-- SQLite doesn't support ALTER COLUMN, so we recreate tables.
-- All tables get: id INTEGER PRIMARY KEY AUTOINCREMENT, reference_id TEXT UNIQUE NOT NULL
-- All user FKs become INTEGER REFERENCES users(id)

PRAGMA foreign_keys = OFF;

-- =============================================================================
-- 1. Create users table
-- =============================================================================

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    email TEXT DEFAULT '',
    name TEXT DEFAULT '',
    plan_id TEXT DEFAULT 'free',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Collect distinct user IDs from all existing tables
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT creator_id, datetime('now'), datetime('now') FROM templates WHERE creator_id != '';
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT customer_id, datetime('now'), datetime('now') FROM deployments WHERE customer_id != '';
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT creator_id, datetime('now'), datetime('now') FROM nodes WHERE creator_id != '';
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT creator_id, datetime('now'), datetime('now') FROM ssh_keys WHERE creator_id != '';
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT creator_id, datetime('now'), datetime('now') FROM cloud_credentials WHERE creator_id != '';
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT creator_id, datetime('now'), datetime('now') FROM cloud_provisions WHERE creator_id != '';
INSERT OR IGNORE INTO users (reference_id, created_at, updated_at)
    SELECT DISTINCT user_id, datetime('now'), datetime('now') FROM usage_events WHERE user_id != '';

CREATE INDEX idx_users_reference_id ON users(reference_id);

-- =============================================================================
-- 2. Recreate ssh_keys with integer PK (must come before nodes due to FK)
-- =============================================================================

CREATE TABLE ssh_keys_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    private_key_encrypted BLOB NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at TEXT NOT NULL
);

INSERT INTO ssh_keys_new (reference_id, creator_id, name, private_key_encrypted, fingerprint, created_at)
    SELECT sk.id, u.id, sk.name, sk.private_key_encrypted, sk.fingerprint, sk.created_at
    FROM ssh_keys sk
    JOIN users u ON u.reference_id = sk.creator_id;

DROP TABLE ssh_keys;
ALTER TABLE ssh_keys_new RENAME TO ssh_keys;

CREATE UNIQUE INDEX idx_ssh_keys_creator_name ON ssh_keys(creator_id, name);

-- =============================================================================
-- 3. Recreate templates with integer PK
-- =============================================================================

CREATE TABLE templates_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
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
    creator_id INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO templates_new (reference_id, name, slug, description, version, compose_spec,
    variables, config_files, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    price_monthly_cents, category, tags, required_capabilities, published, creator_id,
    created_at, updated_at)
    SELECT t.id, t.name, t.slug, t.description, t.version, t.compose_spec,
        t.variables, t.config_files, t.resources_cpu_cores, t.resources_memory_mb, t.resources_disk_mb,
        t.price_monthly_cents, t.category, t.tags, t.required_capabilities, t.published, u.id,
        t.created_at, t.updated_at
    FROM templates t
    JOIN users u ON u.reference_id = t.creator_id;

DROP TABLE templates;
ALTER TABLE templates_new RENAME TO templates;

CREATE INDEX idx_templates_slug ON templates(slug);
CREATE INDEX idx_templates_creator ON templates(creator_id);
CREATE INDEX idx_templates_published ON templates(published);
CREATE INDEX idx_templates_reference_id ON templates(reference_id);

-- =============================================================================
-- 4. Recreate deployments with integer PK
-- =============================================================================

CREATE TABLE deployments_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    template_id INTEGER NOT NULL REFERENCES templates(id),
    template_version TEXT NOT NULL DEFAULT '',
    customer_id INTEGER NOT NULL REFERENCES users(id),
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
    stopped_at TEXT
);

INSERT INTO deployments_new (reference_id, name, template_id, template_version, customer_id,
    node_id, status, variables, domains, containers,
    resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    proxy_port, error_message, created_at, updated_at, started_at, stopped_at)
    SELECT d.id, d.name, t.id, d.template_version, u.id,
        d.node_id, d.status, d.variables, d.domains, d.containers,
        d.resources_cpu_cores, d.resources_memory_mb, d.resources_disk_mb,
        d.proxy_port, d.error_message, d.created_at, d.updated_at, d.started_at, d.stopped_at
    FROM deployments d
    JOIN templates t ON t.reference_id = d.template_id
    JOIN users u ON u.reference_id = d.customer_id;

DROP TABLE deployments;
ALTER TABLE deployments_new RENAME TO deployments;

CREATE INDEX idx_deployments_template ON deployments(template_id);
CREATE INDEX idx_deployments_customer ON deployments(customer_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_node ON deployments(node_id);
CREATE INDEX idx_deployments_reference_id ON deployments(reference_id);

-- =============================================================================
-- 5. Recreate container_events with integer PK
-- =============================================================================

CREATE TABLE container_events_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    deployment_id INTEGER NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    container TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO container_events_new (reference_id, deployment_id, type, container, message, timestamp, created_at)
    SELECT ce.id, d.id, ce.type, ce.container, ce.message, ce.timestamp, ce.created_at
    FROM container_events ce
    JOIN deployments d ON d.reference_id = ce.deployment_id;

DROP TABLE container_events;
ALTER TABLE container_events_new RENAME TO container_events;

CREATE INDEX idx_container_events_deployment_time ON container_events(deployment_id, timestamp DESC);
CREATE INDEX idx_container_events_type ON container_events(deployment_id, type);

-- =============================================================================
-- 6. Recreate usage_events with integer PK
-- =============================================================================

CREATE TABLE usage_events_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id),
    event_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    quantity INTEGER DEFAULT 1,
    metadata TEXT,
    timestamp TEXT NOT NULL,
    reported_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO usage_events_new (reference_id, user_id, event_type, resource_id, resource_type,
    quantity, metadata, timestamp, reported_at, created_at)
    SELECT ue.id, u.id, ue.event_type, ue.resource_id, ue.resource_type,
        ue.quantity, ue.metadata, ue.timestamp, ue.reported_at, ue.created_at
    FROM usage_events ue
    JOIN users u ON u.reference_id = ue.user_id;

DROP TABLE usage_events;
ALTER TABLE usage_events_new RENAME TO usage_events;

CREATE INDEX idx_usage_events_unreported ON usage_events(reported_at) WHERE reported_at IS NULL;
CREATE INDEX idx_usage_events_user_time ON usage_events(user_id, timestamp);

-- =============================================================================
-- 7. Recreate nodes with integer PK
-- =============================================================================

CREATE TABLE nodes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    ssh_host TEXT NOT NULL,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    ssh_user TEXT NOT NULL,
    ssh_key_id INTEGER REFERENCES ssh_keys(id) ON DELETE SET NULL,
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
    updated_at TEXT NOT NULL
);

INSERT INTO nodes_new (reference_id, name, creator_id, ssh_host, ssh_port, ssh_user,
    ssh_key_id, docker_socket, status, capabilities,
    capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
    capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
    location, last_health_check, error_message,
    provider_type, provision_id, base_domain,
    created_at, updated_at)
    SELECT n.id, n.name, u.id, n.ssh_host, n.ssh_port, n.ssh_user,
        sk.id, n.docker_socket, n.status, n.capabilities,
        n.capacity_cpu_cores, n.capacity_memory_mb, n.capacity_disk_mb,
        n.capacity_cpu_used, n.capacity_memory_used_mb, n.capacity_disk_used_mb,
        n.location, n.last_health_check, n.error_message,
        n.provider_type, n.provision_id, n.base_domain,
        n.created_at, n.updated_at
    FROM nodes n
    JOIN users u ON u.reference_id = n.creator_id
    LEFT JOIN ssh_keys sk ON sk.reference_id = n.ssh_key_id;

DROP TABLE nodes;
ALTER TABLE nodes_new RENAME TO nodes;

CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_creator ON nodes(creator_id);
CREATE UNIQUE INDEX idx_nodes_creator_name ON nodes(creator_id, name);
CREATE INDEX idx_nodes_reference_id ON nodes(reference_id);

-- =============================================================================
-- 8. Recreate cloud_credentials with integer PK
-- =============================================================================

CREATE TABLE cloud_credentials_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    credentials_encrypted BLOB NOT NULL,
    default_region TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO cloud_credentials_new (reference_id, creator_id, name, provider, credentials_encrypted,
    default_region, created_at, updated_at)
    SELECT cc.id, u.id, cc.name, cc.provider, cc.credentials_encrypted,
        cc.default_region, cc.created_at, cc.updated_at
    FROM cloud_credentials cc
    JOIN users u ON u.reference_id = cc.creator_id;

DROP TABLE cloud_credentials;
ALTER TABLE cloud_credentials_new RENAME TO cloud_credentials;

CREATE INDEX idx_cloud_credentials_creator ON cloud_credentials(creator_id);
CREATE UNIQUE INDEX idx_cloud_credentials_creator_name ON cloud_credentials(creator_id, name);

-- =============================================================================
-- 9. Recreate cloud_provisions with integer PK
-- =============================================================================

CREATE TABLE cloud_provisions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    credential_id INTEGER NOT NULL REFERENCES cloud_credentials(id),
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

INSERT INTO cloud_provisions_new (reference_id, creator_id, credential_id, provider, status,
    instance_name, region, size, provider_instance_id, public_ip,
    node_id, ssh_key_id, current_step, error_message,
    created_at, updated_at, completed_at)
    SELECT cp.id, u.id, cc.id, cp.provider, cp.status,
        cp.instance_name, cp.region, cp.size, cp.provider_instance_id, cp.public_ip,
        cp.node_id, cp.ssh_key_id, cp.current_step, cp.error_message,
        cp.created_at, cp.updated_at, cp.completed_at
    FROM cloud_provisions cp
    JOIN users u ON u.reference_id = cp.creator_id
    JOIN cloud_credentials cc ON cc.reference_id = cp.credential_id;

DROP TABLE cloud_provisions;
ALTER TABLE cloud_provisions_new RENAME TO cloud_provisions;

CREATE INDEX idx_cloud_provisions_creator ON cloud_provisions(creator_id);
CREATE INDEX idx_cloud_provisions_status ON cloud_provisions(status);
CREATE INDEX idx_cloud_provisions_credential ON cloud_provisions(credential_id);

PRAGMA foreign_keys = ON;
