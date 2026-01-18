-- Drop container events table
DROP INDEX IF EXISTS idx_container_events_type;
DROP INDEX IF EXISTS idx_container_events_deployment_time;
DROP TABLE IF EXISTS container_events;
