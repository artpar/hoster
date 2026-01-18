-- Container events for monitoring (F010: Monitoring Dashboard)
CREATE TABLE IF NOT EXISTS container_events (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    type TEXT NOT NULL,
    container TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

-- Index for listing events by deployment, ordered by time
CREATE INDEX idx_container_events_deployment_time
    ON container_events(deployment_id, timestamp DESC);

-- Index for filtering by event type
CREATE INDEX idx_container_events_type
    ON container_events(deployment_id, type);
