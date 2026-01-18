-- Templates table
CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    version TEXT NOT NULL,
    compose_spec TEXT NOT NULL,
    variables TEXT,           -- JSON array of Variable objects
    resources_cpu_cores REAL NOT NULL DEFAULT 0,
    resources_memory_mb INTEGER NOT NULL DEFAULT 0,
    resources_disk_mb INTEGER NOT NULL DEFAULT 0,
    price_monthly_cents INTEGER DEFAULT 0,
    category TEXT DEFAULT '',
    tags TEXT,                -- JSON array of strings
    published INTEGER DEFAULT 0,  -- 0 = false, 1 = true
    creator_id TEXT NOT NULL,
    created_at TEXT NOT NULL,     -- RFC3339 timestamp
    updated_at TEXT NOT NULL      -- RFC3339 timestamp
);

CREATE INDEX IF NOT EXISTS idx_templates_slug ON templates(slug);
CREATE INDEX IF NOT EXISTS idx_templates_creator ON templates(creator_id);
CREATE INDEX IF NOT EXISTS idx_templates_published ON templates(published);

-- Deployments table
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL,
    template_version TEXT NOT NULL DEFAULT '',
    customer_id TEXT NOT NULL,
    node_id TEXT DEFAULT '',
    status TEXT NOT NULL,             -- pending, scheduled, starting, running, etc.
    variables TEXT,                   -- JSON object of variable values
    domains TEXT,                     -- JSON array of Domain objects
    containers TEXT,                  -- JSON array of ContainerInfo objects
    resources_cpu_cores REAL NOT NULL DEFAULT 0,
    resources_memory_mb INTEGER NOT NULL DEFAULT 0,
    resources_disk_mb INTEGER NOT NULL DEFAULT 0,
    error_message TEXT DEFAULT '',
    created_at TEXT NOT NULL,         -- RFC3339 timestamp
    updated_at TEXT NOT NULL,         -- RFC3339 timestamp
    started_at TEXT,                  -- RFC3339 timestamp (nullable)
    stopped_at TEXT,                  -- RFC3339 timestamp (nullable)
    FOREIGN KEY (template_id) REFERENCES templates(id)
);

CREATE INDEX IF NOT EXISTS idx_deployments_template ON deployments(template_id);
CREATE INDEX IF NOT EXISTS idx_deployments_customer ON deployments(customer_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
