# Local E2E Development Guide

This guide covers running APIGate + Hoster together locally for full production-like end-to-end testing.

## Architecture Overview

```
Browser/API Clients
        |
        v
+------------------------------+
|   APIGate (localhost:8082)   |
|   - User Portal (/portal)    |
|   - Auth (login/signup)      |
|   - API Key Management       |
+-------------+----------------+
              | Injects: X-User-ID, X-Plan-ID, X-Plan-Limits
              v
+------------------------------+
|   Hoster API (localhost:8080)|
|   - Templates API            |
|   - Deployments API          |
|   - Nodes API                |
|   - Frontend UI              |
+-------------+----------------+
              |
              v
+------------------------------+
| App Proxy (localhost:9091)   |
| *.apps.localhost -> containers|
+------------------------------+
```

## Prerequisites

- Docker and docker-compose
- Go 1.21+
- curl and jq (for testing)

## Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# Start everything
make local-e2e-setup

# Or manually:
docker-compose -f deploy/docker-compose.local.yml up -d

# View logs
make local-e2e-logs

# Stop
make local-e2e-down
```

### Option 2: Run from Source

**Terminal 1: Start APIGate**
```bash
cd /path/to/apigate
go run ./cmd/apigate serve
```

**Terminal 2: Start Hoster**
```bash
cd /path/to/hoster
HOSTER_AUTH_MODE=dev go run ./cmd/hoster
```

## Service URLs

| Service | URL | Purpose |
|---------|-----|---------|
| APIGate Portal | http://localhost:8082/portal | User signup, login, API key management |
| Hoster API | http://localhost:8080/api/v1 | Templates, deployments, nodes |
| App Proxy | http://localhost:9091 | Deployed app access |
| Deployed Apps | http://{name}.apps.localhost:9091 | Individual app access |

## Auth Modes

### Dev Mode (Standalone)

Run Hoster without APIGate for quick development:

```bash
HOSTER_AUTH_MODE=dev go run ./cmd/hoster
```

- Session-based auth at `/auth/*` endpoints
- Auto-accepts any email/password
- Generous default limits (100 deployments, 16 CPU cores)

### Header Mode (Production-like)

Run with APIGate for full auth flow:

```bash
HOSTER_AUTH_MODE=header go run ./cmd/hoster
```

- Trusts `X-User-ID`, `X-Plan-ID`, `X-Plan-Limits` headers from APIGate
- Requires APIGate authentication

## E2E Test Flow

### 1. User Registration (APIGate)

```bash
curl -X POST http://localhost:8082/portal/api/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123", "name": "Test User"}'
```

### 2. Login and Get Session

```bash
curl -X POST http://localhost:8082/portal/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}' \
  -c cookies.txt
```

### 3. Create API Key

```bash
curl -X POST http://localhost:8082/portal/api/keys \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name": "my-key"}'
```

### 4. Create Template (via APIGate)

```bash
curl -X POST http://localhost:8082/api/v1/templates \
  -H "Authorization: Bearer ak_xxx" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{
    "data": {
      "type": "templates",
      "attributes": {
        "name": "My App",
        "version": "1.0.0",
        "compose_spec": "version: \"3.8\"\nservices:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"80:80\""
      }
    }
  }'
```

### 5. Deploy and Access

```bash
# Create deployment
curl -X POST http://localhost:8082/api/v1/deployments \
  -H "Authorization: Bearer ak_xxx" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{"data": {"type": "deployments", "attributes": {"template_id": "tmpl_xxx"}}}'

# Start deployment
curl -X POST http://localhost:8082/api/v1/deployments/{id}/start \
  -H "Authorization: Bearer ak_xxx"

# Access deployed app
curl --resolve "my-app-xxx.apps.localhost:9091:127.0.0.1" \
  http://my-app-xxx.apps.localhost:9091/
```

## DNS Configuration

For subdomain routing, add entries to `/etc/hosts`:

```
127.0.0.1 apps.localhost
127.0.0.1 my-app.apps.localhost
```

Or use `nip.io` for automatic DNS:
```bash
HOSTER_APP_PROXY_BASE_DOMAIN=apps.127.0.0.1.nip.io
```

## Header Contract

When running with APIGate, these headers are injected:

| Header | Source | Description |
|--------|--------|-------------|
| `X-User-ID` | `userID` | Authenticated user's UUID |
| `X-Plan-ID` | `planID` | User's subscription plan |
| `X-Plan-Limits` | `planLimits` | JSON with resource limits |
| `X-Key-ID` | `keyID` | API key identifier |

## Troubleshooting

### Port Already in Use

```bash
# Find process using port
lsof -i :8080

# Kill it
kill -9 <PID>
```

### App Proxy Not Routing

1. Check deployment has a domain assigned
2. Ensure `/etc/hosts` has the subdomain entry
3. Verify app proxy is running: `curl localhost:9091/health`

### Plan Limit Errors

In dev mode, limits come from the session. In header mode, limits come from `X-Plan-Limits` header. Ensure the auth mode matches your setup.

## Running Tests

```bash
# Full E2E test suite
make local-e2e-test

# Or manually
./scripts/test-local-e2e.sh
```
