-- Add config_files column to templates table
ALTER TABLE templates ADD COLUMN config_files TEXT;  -- JSON array of ConfigFile objects
