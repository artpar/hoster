-- Users table — needed by all resources for FK references.
-- The engine's schema-driven resources reference users(id) via creator_id/customer_id.
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reference_id TEXT UNIQUE NOT NULL,
    email TEXT DEFAULT '',
    name TEXT DEFAULT '',
    plan_id TEXT DEFAULT 'free',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_users_reference_id ON users(reference_id);
