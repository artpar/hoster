-- F009: Billing Integration - Rollback Usage Events Table

DROP INDEX IF EXISTS idx_usage_events_user_time;
DROP INDEX IF EXISTS idx_usage_events_unreported;
DROP TABLE IF EXISTS usage_events;
