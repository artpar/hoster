# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 20, 2026)

### Status: LOCAL E2E WORKING - READY FOR PRODUCTION DEPLOYMENT

**Local Development - FULLY WORKING:**
- Dev auth mode enabled (`HOSTER_AUTH_MODE=dev`)
- Login/signup via `/auth/*` endpoints
- Full E2E flow tested: signup → browse → deploy → access app
- App proxy subdomain routing fixed (port stripping bug fixed)

**Production Deployment:**
- **URL**: https://emptychair.dev
- **Server**: AWS EC2 (ubuntu@emptychair.dev)
- **APIGate**: Handling TLS via ACME (auto-cert from Let's Encrypt)
- **Hoster**: Running as systemd service (v0.1.0 - backend only)

**What's Working Locally:**
- Dev auth mode with session-based login
- Marketplace browsing
- Template deployment
- Container orchestration
- App proxy routing via subdomain
- All E2E flows verified

**What Needs Production Work:**
- CI workflows need verification (npm/rollup issues were being fixed)
- Need release with embedded frontend
- Production APIGate integration for real auth

---

## LOCAL DEVELOPMENT SETUP

### Quick Start

**Terminal 1: Start Hoster**
```bash
cd /Users/artpar/workspace/code/hoster
HOSTER_AUTH_MODE=dev go run ./cmd/hoster
```

**Terminal 2: Start Frontend**
```bash
cd /Users/artpar/workspace/code/hoster/web
npm run dev
```

**Access:**
- Frontend: http://localhost:3000
- API: http://localhost:8080
- App Proxy: http://localhost:9091
- Deployed apps: http://{name}.apps.localhost:9091

**Note:** Add entries to `/etc/hosts` for app subdomains:
```
127.0.0.1 {deployment-name}.apps.localhost
```

### E2E Test Flow (Verified Working)

1. Open http://localhost:3000
2. Navigate to http://localhost:3000/login
3. Login with any email/password (dev auth accepts anything)
4. Browse marketplace at /marketplace
5. Click on a template, click "Deploy Now"
6. After deployment created, start it via API or UI
7. Access deployed app at http://{name}.apps.localhost:9091

---

## IMMEDIATE NEXT STEPS (Priority Order)

### 1. Verify CI Workflow is Fixed

Check if the npm/rollup issue is resolved:
```bash
gh run list --repo artpar/hoster --limit 3
```

If still failing, see troubleshooting section below.

### 2. Create New Release with Embedded Frontend

Once CI passes:
```bash
# Delete old failed tag if exists
git tag -d v0.2.0 2>/dev/null
git push origin :refs/tags/v0.2.0 2>/dev/null

# Create fresh release
git tag v0.2.0
git push origin v0.2.0
```

### 3. Deploy to Production

```bash
cd deploy/local
make deploy-release VERSION=v0.2.0
```

### 4. Test Production E2E

1. Navigate to https://emptychair.dev
2. Should see Hoster marketplace (not APIGate portal)
3. Sign up / Log in via APIGate
4. Browse templates
5. Deploy a template
6. Access deployed app at https://{name}.apps.emptychair.dev

---

## Files Changed This Session (Session 3)

**Added dev auth mode:**
- `internal/shell/api/dev_auth.go` - NEW - In-memory session auth endpoints
- `internal/shell/api/setup.go` - Register dev auth routes when auth.mode=dev

**Fixed app proxy:**
- `internal/shell/proxy/server.go` - Strip port from Host header for domain matching

**Updated frontend proxy:**
- `web/vite.config.ts` - Proxy /auth and /api to Hoster for local dev

**Commit:** `feat: Add dev auth mode for local E2E testing`

---

## Architecture

### Local Development Mode

```
┌─────────────────────────────────────────────────────────────┐
│  Local Development Stack (Dev Auth Mode)                    │
│                                                             │
│  Frontend (localhost:3000)                                  │
│    │                                                        │
│    ├─ /api/* ──► Hoster API (localhost:8080)               │
│    │                                                        │
│    └─ /auth/* ──► Hoster Dev Auth (localhost:8080)         │
│                                                             │
│  Hoster (localhost:8080)                                    │
│    ├─ /api/v1/* - Deployment API                           │
│    └─ /auth/* - Dev auth endpoints (session-based)         │
│                                                             │
│  App Proxy (localhost:9091)                                 │
│    └─ *.apps.localhost → Deployed containers               │
└─────────────────────────────────────────────────────────────┘
```

### Production Mode (APIGate Integration)

```
┌─────────────────────────────────────────────────────────────┐
│  Production Stack (APIGate Auth)                            │
│                                                             │
│  Internet ───► APIGate (TLS termination)                   │
│                    │                                        │
│                    ├─ /portal/* → APIGate portal           │
│                    ├─ /api/* → Hoster API (with X-User-ID) │
│                    └─ /* → Hoster static files             │
│                                                             │
│  Hoster (auth.mode=header)                                 │
│    └─ Trusts X-User-ID headers from APIGate                │
│                                                             │
│  App Proxy                                                  │
│    └─ *.apps.emptychair.dev → Deployed containers          │
└─────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting

### CI npm/rollup Issue

If CI fails with:
```
Error: Cannot find module @rollup/rollup-linux-x64-gnu
```

**Option A: Clean install (already attempted)**
```yaml
- run: rm -rf node_modules package-lock.json && npm install
```

**Option B: Pin rollup version**
```json
// In web/package.json, add:
"overrides": {
  "rollup": "4.9.6"
}
```

**Option C: Use different npm version**
```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '20'
```

### App Proxy Not Finding Apps

1. Check deployment has a domain assigned:
   ```bash
   curl http://localhost:8080/api/v1/deployments/{id} | jq '.data.attributes.domains'
   ```

2. Ensure /etc/hosts has the subdomain entry

3. Verify app proxy is running on port 9091

### Dev Auth Session Issues

Dev auth stores sessions in memory. Restarting Hoster clears all sessions.
Just log in again after restart.

---

## Production Management

**Deployment via Makefile:**
```bash
cd deploy/local
make deploy-release                    # Deploy latest release
make deploy-release VERSION=v0.2.0     # Deploy specific version
```

**Server Management:**
```bash
cd deploy/local
make status           # Show service status
make logs             # Tail all logs
make logs-hoster      # Tail Hoster logs only
make restart          # Restart both services
make shell            # SSH into server
```

---

## Session History

### Session 3 (January 20, 2026) - CURRENT SESSION

**Goal:** Make Hoster E2E usable locally

**Accomplished:**
1. Created dev auth mode (`internal/shell/api/dev_auth.go`)
   - Session-based auth with in-memory storage
   - Endpoints: /auth/login, /auth/register, /auth/me, /auth/logout
2. Fixed app proxy port stripping bug
   - Browsers send `Host: name.apps.localhost:9091` with port
   - DB stores hostname without port
   - Fixed by stripping port before domain lookup
3. Verified full E2E flow locally:
   - Login ✅
   - Browse marketplace ✅
   - Deploy template ✅
   - Start deployment ✅
   - Access via subdomain ✅

**Not Completed:**
- Production deployment (CI needs verification)
- Production E2E testing

### Session 2 (January 20, 2026)

**Accomplished:**
- Fixed APIGate admin API bug (/api/ → /admin/)
- Set up GitHub Actions CI/CD workflows
- Created v0.1.0 release (backend only)
- Deployed v0.1.0 to production
- Created embedded frontend handler

**Issues:**
- CI workflows failing (npm/rollup issues)

### Session 1 (Earlier)

- Initial Hoster development
- Backend implementation complete

---

## Quick Reference

**GitHub:**
- Repo: https://github.com/artpar/hoster
- Releases: https://github.com/artpar/hoster/releases
- Actions: https://github.com/artpar/hoster/actions

**Production:**
- URL: https://emptychair.dev
- Server: ubuntu@emptychair.dev

**Local Dev Ports:**
- Hoster API: 8080
- App Proxy: 9091
- Frontend dev: 3000
- APIGate (optional): 8082

---

## Onboarding Checklist for New Session

1. [ ] Read CLAUDE.md completely
2. [ ] Read this SESSION-HANDOFF.md
3. [ ] Check CI status: `gh run list --repo artpar/hoster --limit 3`
4. [ ] If CI failing, fix the issue (see troubleshooting)
5. [ ] If CI passing, create release and deploy
6. [ ] Test end-to-end on production

**For local development:**
1. [ ] Start Hoster: `HOSTER_AUTH_MODE=dev go run ./cmd/hoster`
2. [ ] Start frontend: `cd web && npm run dev`
3. [ ] Open http://localhost:3000
4. [ ] Test E2E flow (login → browse → deploy → access)
