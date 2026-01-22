-- Add default marketplace templates for common stable Docker apps

-- PostgreSQL Database
INSERT INTO templates (
    id, name, slug, description, version, category, compose_spec,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_postgres',
    'PostgreSQL Database',
    'postgresql-database',
    'PostgreSQL 16 - Production-ready relational database with 512MB RAM and 0.5 CPU cores',
    '16.0',
    'database',
    'services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-myapp}
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 512M
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $$POSTGRES_USER"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "5432:5432"

volumes:
  postgres_data:',
    500, -- $5.00/month
    0.5,
    512,
    5120,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- MySQL Database
INSERT INTO templates (
    id, name, slug, description, version, category, compose_spec,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_mysql',
    'MySQL Database',
    'mysql-database',
    'MySQL 8.0 - Reliable relational database with 512MB RAM and 0.5 CPU cores',
    '8.0',
    'database',
    'services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: ${MYSQL_DATABASE:-myapp}
      MYSQL_USER: ${MYSQL_USER:-appuser}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 512M
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "3306:3306"

volumes:
  mysql_data:',
    500,
    0.5,
    512,
    5120,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- Redis Cache
INSERT INTO templates (
    id, name, slug, description, version, category, compose_spec,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_redis',
    'Redis Cache',
    'redis-cache',
    'Redis 7 - High-performance in-memory data store with 256MB RAM and 0.25 CPU cores',
    '7.0',
    'cache',
    'services:
  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    deploy:
      resources:
        limits:
          cpus: ''0.25''
          memory: 256M
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "6379:6379"

volumes:
  redis_data:',
    300,
    0.25,
    256,
    2048,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- MongoDB Database
INSERT INTO templates (
    id, name, slug, description, version, category, compose_spec,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_mongodb',
    'MongoDB Database',
    'mongodb-database',
    'MongoDB 7.0 - Flexible NoSQL document database with 512MB RAM and 0.5 CPU cores',
    '7.0',
    'database',
    'services:
  mongodb:
    image: mongo:7.0
    environment:
      MONGO_INITDB_ROOT_USERNAME: ${MONGO_USERNAME:-admin}
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_PASSWORD}
      MONGO_INITDB_DATABASE: ${MONGO_DATABASE:-myapp}
    volumes:
      - mongodb_data:/data/db
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 512M
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand(''ping'')"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "27017:27017"

volumes:
  mongodb_data:',
    500,
    0.5,
    512,
    10240,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- Nginx Web Server
INSERT INTO templates (
    id, name, slug, description, version, category, compose_spec,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_nginx',
    'Nginx Web Server',
    'nginx-web-server',
    'Nginx - High-performance web server and reverse proxy with 64MB RAM and 0.1 CPU cores',
    '1.27',
    'web',
    'services:
  nginx:
    image: nginx:alpine
    volumes:
      - ./html:/usr/share/nginx/html:ro
    deploy:
      resources:
        limits:
          cpus: ''0.1''
          memory: 64M
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost/"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "80:80"',
    200,
    0.1,
    64,
    512,
    'system',
    1,
    datetime('now'),
    datetime('now')
);

-- Node.js Runtime
INSERT INTO templates (
    id, name, slug, description, version, category, compose_spec,
    price_monthly_cents, resources_cpu_cores, resources_memory_mb, resources_disk_mb,
    creator_id, published, created_at, updated_at
) VALUES (
    'tmpl_nodejs',
    'Node.js Application',
    'nodejs-application',
    'Node.js 20 - JavaScript runtime for building scalable applications with 256MB RAM and 0.5 CPU cores',
    '20.0',
    'runtime',
    'services:
  nodejs:
    image: node:20-alpine
    working_dir: /app
    volumes:
      - ./app:/app
    command: npm start
    environment:
      NODE_ENV: ${NODE_ENV:-production}
      PORT: ${PORT:-3000}
    deploy:
      resources:
        limits:
          cpus: ''0.5''
          memory: 256M
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:3000/"]
      interval: 10s
      timeout: 5s
      retries: 5
    ports:
      - "3000:3000"',
    400,
    0.5,
    256,
    2048,
    'system',
    1,
    datetime('now'),
    datetime('now')
);
