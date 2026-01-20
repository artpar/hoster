# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 20, 2026)

### Status: CI FIXING IN PROGRESS - FRONTEND NOT YET DEPLOYED

**Production Deployment:**
- **URL**: https://emptychair.dev
- **Server**: AWS EC2 (ubuntu@emptychair.dev)
- **APIGate**: Handling TLS via ACME (auto-cert from Let's Encrypt)
- **Hoster**: Running as systemd service (v0.1.0 - backend only)

**CRITICAL ISSUE:**
- Users visiting emptychair.dev see APIGate's default portal, NOT Hoster marketplace
- Frontend is NOT embedded in the binary yet
- CI workflows are failing due to npm/rollup native module issues
- v0.2.0 tag exists but release failed - need to fix CI first, then delete and recreate tag

**What's Working:**
- Hoster backend v0.1.0 deployed and running
- APIGate integration fixed (was using /api/ instead of /admin/ for admin endpoints)
- API accessible at https://emptychair.dev/api/v1/...
- Billing reporter running
- App Proxy running on port 9091

**What's NOT Working:**
- No frontend UI visible to users
- CI workflows failing (npm/rollup issues)
- No release with embedded frontend yet

---

## IMMEDIATE NEXT STEPS (Priority Order)

### 1. Fix CI Workflow (npm/rollup issue)

The CI is failing because of npm optional dependency issues with rollup native modules.

**Current error:**
```
Error: Cannot find module @rollup/rollup-linux-x64-gnu
```

**Latest attempt (in progress):**
- Added `rm -rf node_modules package-lock.json` before `npm install`
- Pushed in latest commit on main

**If still failing, try these options:**

**Option A: Pin rollup version**
```json
// In web/package.json, add:
"overrides": {
  "rollup": "4.9.6"
}
```

**Option B: Use npm ci with regenerated lockfile**
```bash
cd web
rm -rf node_modules package-lock.json
npm install
# Commit the new package-lock.json
```

**Option C: Remove unused vitest**
```bash
cd web
npm uninstall vitest
```

### 2. Delete Failed v0.2.0 Tag and Create New Release

The v0.2.0 tag was created before CI was fixed. Once CI passes:

```bash
# Delete the failed tag
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0

# Create fresh release
git tag v0.2.0
git push origin v0.2.0
```

### 3. Deploy and Test End-to-End

Once release succeeds:
```bash
cd deploy/local
make deploy-release VERSION=v0.2.0
```

Test as a real user:
1. Navigate to https://emptychair.dev
2. Should see Hoster marketplace (not APIGate portal)
3. Sign up / Log in
4. Browse templates
5. Deploy a template
6. Access deployed app

---

## Files Changed This Session

**Fixed APIGate admin API:**
- `internal/shell/apigate/client.go` - Changed /api/ to /admin/ prefix
- `internal/shell/apigate/client_test.go` - Updated test expectations
- `internal/shell/apigate/registrar_test.go` - Updated test mocks

**Added embedded frontend (following APIGate pattern):**
- `internal/shell/api/webui.go` - NEW - Serves embedded static files with SPA fallback
- `internal/shell/api/webui/.gitignore` - Ignores dist/ (generated during build)
- `internal/shell/api/setup.go` - Added WebUIHandler() to serve UI at root path

**CI/CD Workflows:**
- `.github/workflows/ci.yml` - NEW - Test, Build, Vet jobs with frontend build
- `.github/workflows/release.yml` - NEW - Release on version tags

**Frontend config:**
- `web/vite.config.ts` - Changed base from /app/ to / (served at root)

**Makefile:**
- `deploy/local/Makefile` - Added deploy-release target

---

## Production Management

**Deployment via Makefile (RECOMMENDED):**
```bash
cd deploy/local
make deploy-release                    # Deploy latest release from GitHub
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

**Service Locations:**
- Hoster binary: `/opt/hoster/bin/hoster`
- Hoster env: `/etc/hoster/.env`
- Hoster DB: `/var/lib/hoster/hoster.db`
- APIGate DB: `/var/lib/apigate/apigate.db`

---

## Architecture: Embedded Frontend

Following APIGate's pattern for embedded UI:

```
internal/shell/api/
├── setup.go           # Mounts WebUIHandler() at PathPrefix("/")
├── webui.go           # Embedded UI handler (SPA pattern)
└── webui/
    ├── .gitignore     # Ignores dist/
    └── dist/          # Copied from web/dist during build (NOT committed)
```

**Build Process (in CI):**
1. `cd web && npm install && npm run build`
2. `cp -r dist ../internal/shell/api/webui/`
3. `go build ./cmd/hoster` (embeds webui/dist via //go:embed)

**Local Development:**
- Frontend: `cd web && npm run dev` (Vite dev server on :3000)
- Backend: `make run` (Hoster on :8080)
- Vite proxies /api to backend

**Local Testing (with embedded frontend):**
```bash
# Build frontend
cd web && npm install && npm run build
cp -r dist ../internal/shell/api/webui/

# Build and run Hoster
cd .. && go build -o /tmp/hoster ./cmd/hoster
/tmp/hoster

# Visit http://localhost:8080
```

---

## Key Technical Decisions Made

1. **Embed frontend into binary** - Like APIGate, no separate nginx/static file server needed
2. **Use npm install (not npm ci)** - Avoids lockfile sync issues across npm versions
3. **Clean node_modules before install** - Fixes rollup native module resolution
4. **Base path = /** - Frontend served at root, not under /app/

---

## What NOT to Do

1. **DON'T use ssh commands directly** - Use Makefile targets
2. **DON'T push without testing locally** - Build and verify before pushing
3. **DON'T skip STC methodology** - Spec first, then test, then code
4. **DON'T create multiple broken commits** - Test CI locally if possible
5. **DON'T keep checking GitHub Actions repeatedly** - Check once, fix if needed
6. **DON'T create tags before CI is green** - Wait for CI to pass first

---

## Verification Commands

**Check CI status:**
```bash
gh run list --repo artpar/hoster --limit 5
gh run view <run-id> --repo artpar/hoster --log-failed
```

**Check production:**
```bash
cd deploy/local
make status
curl -s https://emptychair.dev/ | head -20
curl -s https://emptychair.dev/api/v1/templates | jq .
```

---

## Session History

### Session 2 (January 20, 2026) - Current Session

**Goal:** Deploy Hoster with working frontend to emptychair.dev

**Accomplished:**
1. Fixed APIGate admin API bug (/api/ → /admin/)
2. Set up GitHub Actions CI/CD workflows
3. Created v0.1.0 release (backend only)
4. Deployed v0.1.0 to production
5. Created embedded frontend handler (webui.go following APIGate pattern)
6. Updated CI workflows to build frontend

**Not Completed:**
1. CI workflows still failing (npm/rollup issues) - fix in progress
2. No release with embedded frontend deployed
3. End-to-end user testing not done

**Lessons Learned:**
- Test CI locally before pushing
- Follow STC - don't cowboy code
- Use Makefile for deployments, not raw SSH
- Don't create tags before CI passes

### Session 1 (Earlier)

- Initial Hoster development
- Backend implementation complete
- APIGate integration set up (with bug in admin API path)

---

## Quick Reference

**GitHub:**
- Repo: https://github.com/artpar/hoster
- Releases: https://github.com/artpar/hoster/releases
- Actions: https://github.com/artpar/hoster/actions

**Production:**
- URL: https://emptychair.dev
- Server: ubuntu@emptychair.dev (SSH key: ~/Downloads/emptychair-key.pem)

**Local Dev Ports:**
- Hoster API: 8080
- App Proxy: 9091
- Frontend dev: 3000 or 5173/5174
- APIGate (if running): 8082

---

## Onboarding Checklist for New Session

1. [ ] Read CLAUDE.md completely
2. [ ] Read this SESSION-HANDOFF.md
3. [ ] Check CI status: `gh run list --repo artpar/hoster --limit 3`
4. [ ] If CI failing, fix the npm/rollup issue (see options above)
5. [ ] Once CI green, delete old v0.2.0 tag and create new release
6. [ ] Deploy: `cd deploy/local && make deploy-release VERSION=v0.2.0`
7. [ ] Test end-to-end as a user at https://emptychair.dev
