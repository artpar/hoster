-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- This is a destructive migration - config_files data will be lost

-- Create new table without config_files
CREATE TABLE templates_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    version TEXT NOT NULL,
    compose_spec TEXT NOT NULL,
    variables TEXT,
    resources_cpu_cores REAL NOT NULL DEFAULT 0,
    resources_memory_mb INTEGER NOT NULL DEFAULT 0,
    resources_disk_mb INTEGER NOT NULL DEFAULT 0,
    price_monthly_cents INTEGER DEFAULT 0,
    category TEXT DEFAULT '',
    tags TEXT,
    published INTEGER DEFAULT 0,
    creator_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Copy data
INSERT INTO templates_new SELECT
    id, name, slug, description, version, compose_spec, variables,
    resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    price_monthly_cents, category, tags, published, creator_id,
    created_at, updated_at
FROM templates;

-- Drop old table
DROP TABLE templates;

-- Rename new table
ALTER TABLE templates_new RENAME TO templates;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_templates_slug ON templates(slug);
CREATE INDEX IF NOT EXISTS idx_templates_creator ON templates(creator_id);
CREATE INDEX IF NOT EXISTS idx_templates_published ON templates(published);
