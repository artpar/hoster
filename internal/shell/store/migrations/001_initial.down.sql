-- Drop indexes first
DROP INDEX IF EXISTS idx_deployments_status;
DROP INDEX IF EXISTS idx_deployments_customer;
DROP INDEX IF EXISTS idx_deployments_template;

DROP INDEX IF EXISTS idx_templates_published;
DROP INDEX IF EXISTS idx_templates_creator;
DROP INDEX IF EXISTS idx_templates_slug;

-- Drop tables
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS templates;
