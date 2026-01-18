-- F009: Billing Integration - Usage Events Table

CREATE TABLE IF NOT EXISTS usage_events (
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

CREATE INDEX idx_usage_events_unreported
    ON usage_events(reported_at)
    WHERE reported_at IS NULL;

CREATE INDEX idx_usage_events_user_time
    ON usage_events(user_id, timestamp);
