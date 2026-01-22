-- Remove default marketplace templates

DELETE FROM templates WHERE id IN (
    'tmpl_postgres',
    'tmpl_mysql',
    'tmpl_redis',
    'tmpl_mongodb',
    'tmpl_nginx',
    'tmpl_nodejs'
);
