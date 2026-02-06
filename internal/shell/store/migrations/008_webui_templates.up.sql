-- Add marketplace templates for apps with web UIs

-- WordPress - Blog/CMS with full admin UI
INSERT OR IGNORE INTO templates (
    id, name, slug, description, version, category, compose_spec, variables,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_wordpress',
    'WordPress',
    'wordpress',
    'WordPress - The world''s most popular CMS. Full admin dashboard, themes, plugins. Includes MySQL database.',
    '6.7',
    'web',
    'services:
  wordpress:
    image: wordpress:6-apache
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: ${DB_PASSWORD:-changeme}
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - wp_data:/var/www/html
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 512M
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "80:80"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 30s
  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD:-changeme}
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: ${DB_PASSWORD:-changeme}
    volumes:
      - db_data:/var/lib/mysql
    deploy:
      resources:
        limits:
          cpus: ''0.25''
          memory: 256M
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  wp_data:
  db_data:',
    '[{"name":"DB_PASSWORD","label":"Database Password","description":"Password for the WordPress database","type":"password","required":true}]',
    800, -- $8.00/month
    0.75,
    768,
    10240,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- Uptime Kuma - Self-hosted monitoring dashboard
INSERT OR IGNORE INTO templates (
    id, name, slug, description, version, category, compose_spec, variables,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_uptime_kuma',
    'Uptime Kuma',
    'uptime-kuma',
    'Uptime Kuma - Self-hosted monitoring tool. Beautiful status pages, notifications via Slack/Discord/Email, TCP/HTTP/DNS checks.',
    '1.23',
    'monitoring',
    'services:
  uptime-kuma:
    image: louislam/uptime-kuma:1
    volumes:
      - uptime_data:/app/data
    deploy:
      resources:
        limits:
          cpus: ''0.25''
          memory: 256M
    ports:
      - "3001:3001"
    healthcheck:
      test: ["CMD-SHELL", "node -e \"require(''http'').get(''http://localhost:3001'', (r) => r.statusCode === 200 ? process.exit(0) : process.exit(1))\""]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 15s

volumes:
  uptime_data:',
    NULL,
    400, -- $4.00/month
    0.25,
    256,
    2048,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- Gitea - Self-hosted Git service
INSERT OR IGNORE INTO templates (
    id, name, slug, description, version, category, compose_spec, variables,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_gitea',
    'Gitea',
    'gitea',
    'Gitea - Lightweight self-hosted Git service. Repository management, issue tracking, pull requests, CI/CD via Gitea Actions.',
    '1.22',
    'development',
    'services:
  gitea:
    image: gitea/gitea:1.22
    environment:
      GITEA__database__DB_TYPE: sqlite3
      GITEA__server__ROOT_URL: http://localhost:3000
      GITEA__server__HTTP_PORT: "3000"
    volumes:
      - gitea_data:/data
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 512M
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/api/healthz"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 20s

volumes:
  gitea_data:',
    NULL,
    500, -- $5.00/month
    0.5,
    512,
    5120,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- n8n - Workflow automation platform
INSERT OR IGNORE INTO templates (
    id, name, slug, description, version, category, compose_spec, variables,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_n8n',
    'n8n Workflow Automation',
    'n8n-workflow-automation',
    'n8n - Open-source workflow automation. Connect APIs, build automations with a visual editor. 400+ integrations.',
    '1.70',
    'automation',
    'services:
  n8n:
    image: n8nio/n8n:latest
    environment:
      N8N_SECURE_COOKIE: "false"
      WEBHOOK_URL: http://localhost:5678/
    volumes:
      - n8n_data:/home/node/.n8n
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 512M
    ports:
      - "5678:5678"
    healthcheck:
      test: ["CMD-SHELL", "wget --spider -q http://localhost:5678/healthz || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 20s

volumes:
  n8n_data:',
    NULL,
    600, -- $6.00/month
    0.5,
    512,
    5120,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- IT Tools - Collection of developer utilities
INSERT OR IGNORE INTO templates (
    id, name, slug, description, version, category, compose_spec, variables,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_it_tools',
    'IT Tools',
    'it-tools',
    'IT Tools - 80+ useful tools for developers. Hash generators, encoders, converters, network tools, formatters, and more.',
    '2024.5',
    'development',
    'services:
  it-tools:
    image: corentinth/it-tools:latest
    deploy:
      resources:
        limits:
          cpus: ''0.1''
          memory: 64M
    ports:
      - "80:80"
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost/"]
      interval: 10s
      timeout: 5s
      retries: 5',
    NULL,
    200, -- $2.00/month
    0.1,
    64,
    256,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- Metabase - Business intelligence and analytics
INSERT OR IGNORE INTO templates (
    id, name, slug, description, version, category, compose_spec, variables,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_metabase',
    'Metabase',
    'metabase',
    'Metabase - Open-source business intelligence. Connect databases, build dashboards, ask questions in plain English.',
    '0.51',
    'analytics',
    'services:
  metabase:
    image: metabase/metabase:latest
    environment:
      MB_DB_TYPE: h2
      JAVA_TOOL_OPTIONS: -Xmx512m
    volumes:
      - metabase_data:/metabase-data
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 768M
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/api/health"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 60s

volumes:
  metabase_data:',
    NULL,
    700, -- $7.00/month
    0.5,
    768,
    5120,
    'system',
    1,
    datetime('now'),
    datetime('now')
);
