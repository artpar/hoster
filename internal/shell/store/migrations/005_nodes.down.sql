-- Migration 005: Remove nodes and ssh_keys tables

-- Remove indexes first
DROP INDEX IF EXISTS idx_deployments_node;
DROP INDEX IF EXISTS idx_nodes_creator;
DROP INDEX IF EXISTS idx_nodes_status;

-- SQLite doesn't support DROP COLUMN, so we need to recreate tables
-- For simplicity in development, we'll leave the columns (they're nullable)
-- In production migration, you would recreate the tables

-- Drop nodes table
DROP TABLE IF EXISTS nodes;

-- Drop ssh_keys table
DROP TABLE IF EXISTS ssh_keys;
