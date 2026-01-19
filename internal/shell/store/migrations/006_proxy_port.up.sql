-- Add proxy_port column to deployments for App Proxy routing
ALTER TABLE deployments ADD COLUMN proxy_port INTEGER;

-- Create index on domains JSON for faster lookups
-- Note: SQLite doesn't support functional indexes, but we can query with json_extract
