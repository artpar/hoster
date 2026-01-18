# Hoster

A self-hosted deployment platform - your own Railway/Render/Heroku with a template marketplace.

## What is Hoster?

Hoster is a multi-tenant deployment platform where:
- **Package Creators** define deployment templates (docker-compose + config files + pricing)
- **Customers** deploy instances from templates via REST API
- **You** run the platform on your infrastructure

## Features

- Template-based deployments with Docker Compose support
- Multi-service deployments with dependency ordering
- Config file injection into containers
- Environment variable support
- Volume management
- Health checks and restart policies
- Auto-generated domains
- Full REST API

## Quick Start

```bash
# Clone and build
git clone https://github.com/artpar/hoster.git
cd hoster
make deps
make build

# Run (requires Docker)
./hoster serve

# Or with custom config
HOSTER_SERVER_PORT=9090 ./hoster serve
```

Server starts at `http://localhost:9090` (default).

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HOSTER_SERVER_PORT` | 9090 | HTTP server port |
| `HOSTER_DATABASE_PATH` | ./hoster.db | SQLite database path |
| `HOSTER_DOMAIN_BASE_DOMAIN` | apps.localhost | Base domain for deployments |
| `HOSTER_DOCKER_HOST` | unix:///var/run/docker.sock | Docker socket |
| `HOSTER_CONFIG_DIR` | /var/lib/hoster/configs | Config files directory |

---

## API Reference

Base URL: `http://localhost:9090/api/v1`

### Health Check

```bash
# Health status
curl http://localhost:9090/health

# Readiness (checks Docker connection)
curl http://localhost:9090/ready
```

---

## Templates

Templates define what can be deployed. They contain a Docker Compose spec, optional config files, and metadata.

### Create a Template

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-web",
    "version": "1.0.0",
    "creator_id": "user-1",
    "description": "Simple nginx web server",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\""
  }'
```

Response:
```json
{
  "id": "tmpl_abc123",
  "name": "nginx-web",
  "slug": "nginx-web",
  "version": "1.0.0",
  "published": false,
  ...
}
```

### Publish a Template

Templates must be published before they can be deployed:

```bash
curl -X POST http://localhost:9090/api/v1/templates/tmpl_abc123/publish
```

### List Templates

```bash
curl http://localhost:9090/api/v1/templates
curl http://localhost:9090/api/v1/templates?limit=10&offset=0
```

### Get Template

```bash
curl http://localhost:9090/api/v1/templates/tmpl_abc123
```

### Update Template

Only unpublished templates can be updated:

```bash
curl -X PUT http://localhost:9090/api/v1/templates/tmpl_abc123 \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "price_monthly": 500
  }'
```

### Delete Template

Templates with active deployments cannot be deleted:

```bash
curl -X DELETE http://localhost:9090/api/v1/templates/tmpl_abc123
```

---

## Deployments

Deployments are instances created from templates.

### Create Deployment

```bash
curl -X POST http://localhost:9090/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "tmpl_abc123",
    "customer_id": "customer-1",
    "name": "my-website"
  }'
```

Response:
```json
{
  "id": "depl_xyz789",
  "name": "my-website",
  "template_id": "tmpl_abc123",
  "status": "pending",
  ...
}
```

### Start Deployment

```bash
curl -X POST http://localhost:9090/api/v1/deployments/depl_xyz789/start
```

Response includes container info:
```json
{
  "id": "depl_xyz789",
  "status": "running",
  "containers": [
    {
      "service_name": "web",
      "container_id": "abc123def456",
      "status": "running"
    }
  ],
  ...
}
```

### Stop Deployment

```bash
curl -X POST http://localhost:9090/api/v1/deployments/depl_xyz789/stop
```

### List Deployments

```bash
# All deployments
curl http://localhost:9090/api/v1/deployments

# Filter by customer
curl http://localhost:9090/api/v1/deployments?customer_id=customer-1

# Filter by template
curl http://localhost:9090/api/v1/deployments?template_id=tmpl_abc123
```

### Get Deployment

```bash
curl http://localhost:9090/api/v1/deployments/depl_xyz789
```

### Delete Deployment

Stops containers and removes all resources:

```bash
curl -X DELETE http://localhost:9090/api/v1/deployments/depl_xyz789
```

---

## Example Templates

### 1. Simple Web Server (Nginx)

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-simple",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\""
  }'
```

### 2. Database (PostgreSQL)

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "postgres-db",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  db:\n    image: postgres:15-alpine\n    ports:\n      - \"5432:5432\"\n    environment:\n      POSTGRES_PASSWORD: secret\n      POSTGRES_USER: app\n      POSTGRES_DB: appdb"
  }'
```

### 3. Redis Cache

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "redis-cache",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  redis:\n    image: redis:alpine\n    ports:\n      - \"6379:6379\""
  }'
```

### 4. Multi-Service (Web + Database)

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-with-db",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: adminer\n    ports:\n      - \"8080:8080\"\n    depends_on:\n      - db\n  db:\n    image: postgres:15-alpine\n    environment:\n      POSTGRES_PASSWORD: secret"
  }'
```

### 5. Custom Nginx with Config File

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-custom",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"",
    "config_files": [
      {
        "name": "default.conf",
        "path": "/etc/nginx/conf.d/default.conf",
        "content": "server {\n    listen 80;\n    location / {\n        return 200 \"Hello from Hoster!\";\n        add_header Content-Type text/plain;\n    }\n    location /health {\n        return 200 \"ok\";\n    }\n}"
      }
    ]
  }'
```

### 6. Three-Tier Application

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "three-tier-app",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"\n    depends_on:\n      - api\n  api:\n    image: traefik/whoami\n    depends_on:\n      - db\n  db:\n    image: redis:alpine"
  }'
```

### 7. With Volumes

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-persistent",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"\n    volumes:\n      - data:/usr/share/nginx/html\nvolumes:\n  data:"
  }'
```

### 8. With Health Check

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-healthcheck",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"\n    healthcheck:\n      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost\"]\n      interval: 30s\n      timeout: 10s\n      retries: 3"
  }'
```

### 9. Multiple Ports

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "multi-port",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\"\n      - \"8443:80\""
  }'
```

### 10. With Environment Variables

```bash
curl -X POST http://localhost:9090/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "app-with-env",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  app:\n    image: busybox\n    command: [\"env\"]\n    environment:\n      APP_NAME: myapp\n      DEBUG: \"true\"\n      MAX_CONNECTIONS: \"100\""
  }'
```

---

## Complete Workflow Example

```bash
BASE_URL="http://localhost:9090/api/v1"

# 1. Create a template
TEMPLATE=$(curl -s -X POST "$BASE_URL/templates" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-webapp",
    "version": "1.0.0",
    "creator_id": "admin",
    "compose_spec": "services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8888:80\"",
    "config_files": [
      {
        "name": "index.html",
        "path": "/usr/share/nginx/html/index.html",
        "content": "<h1>Welcome to My App!</h1>"
      }
    ]
  }')
TEMPLATE_ID=$(echo $TEMPLATE | jq -r '.id')
echo "Created template: $TEMPLATE_ID"

# 2. Publish the template
curl -s -X POST "$BASE_URL/templates/$TEMPLATE_ID/publish"
echo "Published template"

# 3. Create a deployment
DEPLOYMENT=$(curl -s -X POST "$BASE_URL/deployments" \
  -H "Content-Type: application/json" \
  -d "{
    \"template_id\": \"$TEMPLATE_ID\",
    \"customer_id\": \"customer-123\",
    \"name\": \"production-site\"
  }")
DEPLOYMENT_ID=$(echo $DEPLOYMENT | jq -r '.id')
echo "Created deployment: $DEPLOYMENT_ID"

# 4. Start the deployment
curl -s -X POST "$BASE_URL/deployments/$DEPLOYMENT_ID/start"
echo "Started deployment"

# 5. Test the service
sleep 2
curl http://localhost:8888
# Output: <h1>Welcome to My App!</h1>

# 6. Check deployment status
curl -s "$BASE_URL/deployments/$DEPLOYMENT_ID" | jq '.status'
# Output: "running"

# 7. Stop when done
curl -s -X POST "$BASE_URL/deployments/$DEPLOYMENT_ID/stop"
echo "Stopped deployment"

# 8. Clean up
curl -s -X DELETE "$BASE_URL/deployments/$DEPLOYMENT_ID"
curl -s -X DELETE "$BASE_URL/templates/$TEMPLATE_ID"
echo "Cleaned up"
```

---

## Supported Docker Compose Features

| Feature | Supported |
|---------|-----------|
| `services` | Yes |
| `image` | Yes |
| `ports` | Yes |
| `environment` | Yes |
| `depends_on` | Yes (ordering) |
| `volumes` (named) | Yes |
| `command` | Yes |
| `entrypoint` | Yes |
| `healthcheck` | Yes |
| `restart` | Yes |
| `networks` | Auto-created |

---

## Deployment States

```
pending -> scheduled -> starting -> running
                           |          |
                           v          v
                        failed <-- stopping -> stopped
                           ^                      |
                           |                      v
                           +---- deleting -> deleted
```

Valid transitions:
- `pending` -> `scheduled`
- `scheduled` -> `starting`
- `starting` -> `running` or `failed`
- `running` -> `stopping` or `failed`
- `stopping` -> `stopped`
- `stopped` -> `starting` (restart) or `deleting`
- `failed` -> `starting` (retry) or `deleting`

---

## Testing

```bash
make test           # All tests (~270)
make test-unit      # Core logic tests
make test-integration # Docker/DB tests
make test-e2e       # Full system tests
```

---

## Architecture

```
hoster/
├── cmd/hoster/           # Entry point
├── internal/
│   ├── core/             # Pure business logic (no I/O)
│   │   ├── domain/       # Domain types
│   │   ├── compose/      # Compose parsing
│   │   ├── deployment/   # Deployment planning
│   │   └── validation/   # Input validation
│   └── shell/            # I/O layer
│       ├── api/          # HTTP handlers
│       ├── docker/       # Docker SDK
│       └── store/        # SQLite storage
└── specs/                # Specifications
```

---

## License

MIT
