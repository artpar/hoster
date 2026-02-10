-- Remove standalone infrastructure templates that don't make sense as deployable apps.
-- Keep only full application stacks (WordPress, Uptime Kuma, Gitea, n8n, IT Tools, Metabase).
-- Use slug to identify (stable across ID normalization from migration 010).

DELETE FROM templates WHERE slug = 'postgresql-database';
DELETE FROM templates WHERE slug = 'mysql-database';
DELETE FROM templates WHERE slug = 'redis-cache';
DELETE FROM templates WHERE slug = 'mongodb-database';
DELETE FROM templates WHERE slug = 'nginx-web-server';
DELETE FROM templates WHERE slug = 'nodejs-application';
