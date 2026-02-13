-- System user for default template ownership
INSERT OR IGNORE INTO users (reference_id, email, name, plan_id, created_at, updated_at)
VALUES ('system', 'system@hoster.local', 'System', 'admin', datetime('now'), datetime('now'));

-- 20 default application templates (owned by system user)
-- web (6), development (4), monitoring (3), automation (4), analytics (3)

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_wordpress', 'WordPress', 'wordpress',
  'The world''s most popular CMS. Build blogs, business sites, and online stores with thousands of themes and plugins.',
  '1.0.0',
  'version: "3.8"

services:
  wordpress:
    image: wordpress:6-apache
    ports:
      - "80:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - wp_data:/var/www/html
    depends_on:
      - db

  db:
    image: mariadb:11
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: wordpress
    volumes:
      - db_data:/var/lib/mysql

volumes:
  wp_data:
  db_data:
',
  'web', 0.5, 512, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_ghost', 'Ghost', 'ghost',
  'Professional publishing platform for creators. Beautiful editor, memberships, newsletters, and built-in SEO.',
  '1.0.0',
  'version: "3.8"

services:
  ghost:
    image: ghost:5-alpine
    ports:
      - "80:2368"
    environment:
      url: http://localhost
      database__client: mysql
      database__connection__host: db
      database__connection__user: ghost
      database__connection__password: ghost
      database__connection__database: ghost
    volumes:
      - ghost_data:/var/lib/ghost/content
    depends_on:
      - db

  db:
    image: mariadb:11
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: ghost
      MYSQL_USER: ghost
      MYSQL_PASSWORD: ghost
    volumes:
      - db_data:/var/lib/mysql

volumes:
  ghost_data:
  db_data:
',
  'web', 0.5, 512, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_nextcloud', 'Nextcloud', 'nextcloud',
  'Self-hosted productivity platform. File sync, calendar, contacts, mail, video calls, and office document editing.',
  '1.0.0',
  'version: "3.8"

services:
  nextcloud:
    image: nextcloud:28-apache
    ports:
      - "80:80"
    environment:
      MYSQL_HOST: db
      MYSQL_DATABASE: nextcloud
      MYSQL_USER: nextcloud
      MYSQL_PASSWORD: nextcloud
    volumes:
      - nextcloud_data:/var/www/html
    depends_on:
      - db

  db:
    image: mariadb:11
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: nextcloud
      MYSQL_USER: nextcloud
      MYSQL_PASSWORD: nextcloud
    volumes:
      - db_data:/var/lib/mysql

volumes:
  nextcloud_data:
  db_data:
',
  'web', 1.0, 1024, 5120, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_bookstack', 'BookStack', 'bookstack',
  'Simple and free wiki software. Organize content into books, chapters, and pages with a WYSIWYG editor.',
  '1.0.0',
  'version: "3.8"

services:
  bookstack:
    image: lscr.io/linuxserver/bookstack:latest
    ports:
      - "80:80"
    environment:
      APP_URL: http://localhost
      DB_HOST: db
      DB_DATABASE: bookstack
      DB_USERNAME: bookstack
      DB_PASSWORD: bookstack
    volumes:
      - bookstack_data:/config
    depends_on:
      - db

  db:
    image: mariadb:11
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: bookstack
      MYSQL_USER: bookstack
      MYSQL_PASSWORD: bookstack
    volumes:
      - db_data:/var/lib/mysql

volumes:
  bookstack_data:
  db_data:
',
  'web', 0.5, 512, 1024, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_chatwoot', 'Chatwoot', 'chatwoot',
  'Open-source customer engagement suite. Live chat, email, social media, and WhatsApp in one unified inbox.',
  '1.0.0',
  'version: "3.8"

services:
  chatwoot:
    image: chatwoot/chatwoot:latest
    ports:
      - "80:3000"
    environment:
      RAILS_ENV: production
      SECRET_KEY_BASE: replace_with_secret_key
      FRONTEND_URL: http://localhost
      POSTGRES_HOST: db
      POSTGRES_USERNAME: chatwoot
      POSTGRES_PASSWORD: chatwoot
      POSTGRES_DATABASE: chatwoot
      REDIS_URL: redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: chatwoot
      POSTGRES_PASSWORD: chatwoot
      POSTGRES_DB: chatwoot
    volumes:
      - pg_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  pg_data:
  redis_data:
',
  'web', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_jellyfin', 'Jellyfin', 'jellyfin',
  'Free media server. Stream your movies, TV shows, music, and photos to any device from your own server.',
  '1.0.0',
  'version: "3.8"

services:
  jellyfin:
    image: jellyfin/jellyfin:latest
    ports:
      - "80:8096"
    environment:
      JELLYFIN_PublishedServerUrl: http://localhost
    volumes:
      - config:/config
      - cache:/cache
      - media:/media

volumes:
  config:
  cache:
  media:
',
  'web', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_gitea', 'Gitea', 'gitea',
  'Lightweight self-hosted Git service. Repositories, issues, pull requests, and CI/CD — like GitHub on your own server.',
  '1.0.0',
  'version: "3.8"

services:
  gitea:
    image: gitea/gitea:latest
    ports:
      - "80:3000"
      - "2222:22"
    environment:
      GITEA__database__DB_TYPE: postgres
      GITEA__database__HOST: db:5432
      GITEA__database__NAME: gitea
      GITEA__database__USER: gitea
      GITEA__database__PASSWD: gitea
    volumes:
      - gitea_data:/data
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: gitea
      POSTGRES_PASSWORD: gitea
      POSTGRES_DB: gitea
    volumes:
      - pg_data:/var/lib/postgresql/data

volumes:
  gitea_data:
  pg_data:
',
  'development', 0.5, 512, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_code-server', 'Code Server', 'code-server',
  'VS Code in the browser. Full development environment accessible from any device with a web browser.',
  '1.0.0',
  'version: "3.8"

services:
  code-server:
    image: lscr.io/linuxserver/code-server:latest
    ports:
      - "80:8443"
    environment:
      PASSWORD: changeme
      SUDO_PASSWORD: changeme
      DEFAULT_WORKSPACE: /workspace
    volumes:
      - config:/config
      - workspace:/workspace

volumes:
  config:
  workspace:
',
  'development', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_wikijs', 'Wiki.js', 'wiki-js',
  'Modern and powerful wiki engine with visual editor, Git sync, and full-text search. Beautiful and intuitive.',
  '1.0.0',
  'version: "3.8"

services:
  wikijs:
    image: ghcr.io/requarks/wiki:2
    ports:
      - "80:3000"
    environment:
      DB_TYPE: postgres
      DB_HOST: db
      DB_PORT: 5432
      DB_USER: wikijs
      DB_PASS: wikijs
      DB_NAME: wikijs
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: wikijs
      POSTGRES_PASSWORD: wikijs
      POSTGRES_DB: wikijs
    volumes:
      - pg_data:/var/lib/postgresql/data

volumes:
  pg_data:
',
  'development', 0.5, 512, 1024, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_outline', 'Outline', 'outline',
  'Beautiful team knowledge base and wiki. Real-time collaboration, Markdown support, and Slack integration.',
  '1.0.0',
  'version: "3.8"

services:
  outline:
    image: outlinewiki/outline:latest
    ports:
      - "80:3000"
    environment:
      DATABASE_URL: postgres://outline:outline@db:5432/outline
      REDIS_URL: redis://redis:6379
      SECRET_KEY: replace_with_64_char_hex_secret
      UTILS_SECRET: replace_with_64_char_hex_secret
      URL: http://localhost
      FILE_STORAGE: local
      FILE_STORAGE_LOCAL_ROOT_DIR: /var/lib/outline/data
    volumes:
      - outline_data:/var/lib/outline/data
    depends_on:
      - db
      - redis

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: outline
      POSTGRES_PASSWORD: outline
      POSTGRES_DB: outline
    volumes:
      - pg_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  outline_data:
  pg_data:
  redis_data:
',
  'development', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_uptime-kuma', 'Uptime Kuma', 'uptime-kuma',
  'Self-hosted monitoring tool. Track uptime for HTTP, TCP, DNS, and more with beautiful status pages and notifications.',
  '1.0.0',
  'version: "3.8"

services:
  uptime-kuma:
    image: louislam/uptime-kuma:latest
    ports:
      - "80:3001"
    volumes:
      - data:/app/data

volumes:
  data:
',
  'monitoring', 0.5, 256, 512, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_grafana', 'Grafana', 'grafana',
  'Observability dashboards for your data. Visualize metrics, logs, and traces from any source with stunning dashboards.',
  '1.0.0',
  'version: "3.8"

services:
  grafana:
    image: grafana/grafana-oss:latest
    ports:
      - "80:3000"
    environment:
      GF_SECURITY_ADMIN_USER: admin
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana_data:/var/lib/grafana

volumes:
  grafana_data:
',
  'monitoring', 0.5, 256, 512, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_healthchecks', 'Healthchecks', 'healthchecks',
  'Cron job and background task monitoring. Get alerted when your scheduled tasks don''t run on time.',
  '1.0.0',
  'version: "3.8"

services:
  healthchecks:
    image: lscr.io/linuxserver/healthchecks:latest
    ports:
      - "80:8000"
    environment:
      SITE_ROOT: http://localhost
      SITE_NAME: Healthchecks
      SECRET_KEY: replace_with_secret_key
      ALLOWED_HOSTS: "*"
    volumes:
      - data:/config

volumes:
  data:
',
  'monitoring', 0.5, 256, 512, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_n8n', 'n8n', 'n8n',
  'Workflow automation platform. Connect 350+ apps and services with a visual editor. Alternative to Zapier and Make.',
  '1.0.0',
  'version: "3.8"

services:
  n8n:
    image: n8nio/n8n:latest
    ports:
      - "80:5678"
    environment:
      N8N_BASIC_AUTH_ACTIVE: "true"
      N8N_BASIC_AUTH_USER: admin
      N8N_BASIC_AUTH_PASSWORD: changeme
      WEBHOOK_URL: http://localhost/
    volumes:
      - n8n_data:/home/node/.n8n

volumes:
  n8n_data:
',
  'automation', 0.5, 512, 1024, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_huginn', 'Huginn', 'huginn',
  'Build agents that monitor the web and act on your behalf. Automated data collection, alerts, and workflows.',
  '1.0.0',
  'version: "3.8"

services:
  huginn:
    image: ghcr.io/huginn/huginn:latest
    ports:
      - "80:3000"
    environment:
      DOMAIN: localhost
      DATABASE_ADAPTER: postgresql
      DATABASE_HOST: db
      DATABASE_NAME: huginn
      DATABASE_USERNAME: huginn
      DATABASE_PASSWORD: huginn
      INVITATION_CODE: changeme
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: huginn
      POSTGRES_PASSWORD: huginn
      POSTGRES_DB: huginn
    volumes:
      - pg_data:/var/lib/postgresql/data

volumes:
  pg_data:
',
  'automation', 0.5, 512, 1024, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_directus', 'Directus', 'directus',
  'Headless CMS and data platform. Instant REST and GraphQL APIs for any SQL database with a beautiful admin UI.',
  '1.0.0',
  'version: "3.8"

services:
  directus:
    image: directus/directus:latest
    ports:
      - "80:8055"
    environment:
      SECRET: replace_with_secret_key
      ADMIN_EMAIL: admin@example.com
      ADMIN_PASSWORD: changeme
      DB_CLIENT: pg
      DB_HOST: db
      DB_PORT: 5432
      DB_DATABASE: directus
      DB_USER: directus
      DB_PASSWORD: directus
    volumes:
      - uploads:/directus/uploads
      - extensions:/directus/extensions
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: directus
      POSTGRES_PASSWORD: directus
      POSTGRES_DB: directus
    volumes:
      - pg_data:/var/lib/postgresql/data

volumes:
  uploads:
  extensions:
  pg_data:
',
  'automation', 1.0, 512, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_appsmith', 'Appsmith', 'appsmith',
  'Low-code platform to build internal tools fast. Drag-and-drop UI builder with database and API integrations.',
  '1.0.0',
  'version: "3.8"

services:
  appsmith:
    image: appsmith/appsmith-ee:latest
    ports:
      - "80:80"
    volumes:
      - stacks:/appsmith-stacks

volumes:
  stacks:
',
  'automation', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_plausible', 'Plausible Analytics', 'plausible-analytics',
  'Privacy-friendly Google Analytics alternative. Lightweight script, no cookies, fully compliant with GDPR and CCPA.',
  '1.0.0',
  'version: "3.8"

services:
  plausible:
    image: ghcr.io/plausible/community-edition:latest
    ports:
      - "80:8000"
    environment:
      BASE_URL: http://localhost
      SECRET_KEY_BASE: replace_with_64_char_secret
      DATABASE_URL: postgres://plausible:plausible@db:5432/plausible
      CLICKHOUSE_DATABASE_URL: http://clickhouse:8123/plausible
    depends_on:
      - db
      - clickhouse

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: plausible
      POSTGRES_PASSWORD: plausible
      POSTGRES_DB: plausible
    volumes:
      - pg_data:/var/lib/postgresql/data

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    volumes:
      - ch_data:/var/lib/clickhouse
    ulimits:
      nofile:
        soft: 262144
        hard: 262144

volumes:
  pg_data:
  ch_data:
',
  'analytics', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_metabase', 'Metabase', 'metabase',
  'Business intelligence and analytics. Connect your database and create beautiful dashboards and questions without SQL.',
  '1.0.0',
  'version: "3.8"

services:
  metabase:
    image: metabase/metabase:latest
    ports:
      - "80:3000"
    environment:
      MB_DB_TYPE: postgres
      MB_DB_DBNAME: metabase
      MB_DB_PORT: 5432
      MB_DB_USER: metabase
      MB_DB_PASS: metabase
      MB_DB_HOST: db
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: metabase
      POSTGRES_PASSWORD: metabase
      POSTGRES_DB: metabase
    volumes:
      - pg_data:/var/lib/postgresql/data

volumes:
  pg_data:
',
  'analytics', 1.0, 1024, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);

INSERT OR IGNORE INTO templates (reference_id, name, slug, description, version, compose_spec, category, resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents, published, creator_id, created_at, updated_at)
VALUES (
  'tmpl_matomo', 'Matomo', 'matomo',
  'Full-featured web analytics platform. 100% data ownership, heatmaps, session recordings, and conversion tracking.',
  '1.0.0',
  'version: "3.8"

services:
  matomo:
    image: matomo:latest
    ports:
      - "80:80"
    environment:
      MATOMO_DATABASE_HOST: db
      MATOMO_DATABASE_DBNAME: matomo
      MATOMO_DATABASE_USERNAME: matomo
      MATOMO_DATABASE_PASSWORD: matomo
    volumes:
      - matomo_data:/var/www/html
    depends_on:
      - db

  db:
    image: mariadb:11
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: matomo
      MYSQL_USER: matomo
      MYSQL_PASSWORD: matomo
    volumes:
      - db_data:/var/lib/mysql

volumes:
  matomo_data:
  db_data:
',
  'analytics', 0.5, 512, 2048, 0, 1,
  (SELECT id FROM users WHERE reference_id = 'system'),
  datetime('now'), datetime('now')
);
