# Next Session: Production Deployment

> **Date Created:** January 22, 2026
> **Status:** Monitoring features complete, ready for production deployment

---

## What Was Completed This Session

### 1. Deployment Monitoring ✅

**Event Recording:**
- Added container lifecycle event tracking to orchestrator
- Events: created, started, stopped, restarted, died, OOM, health checks
- Stored in `container_events` table with timestamps
- Events tab in UI shows deployment history timeline

**Stats Monitoring:**
- Real-time CPU, memory, network, disk I/O metrics
- Endpoint: `GET /api/v1/deployments/{id}/monitoring/stats`
- Stats tab shows current resource usage

**Logs Streaming:**
- Container logs with timestamps and filtering
- Endpoint: `GET /api/v1/deployments/{id}/monitoring/logs`
- Logs tab shows scrollable log output

**Files Changed:**
- `internal/shell/docker/orchestrator.go` - Added StoreInterface and recordEvent()
- `internal/shell/api/handler.go` - Updated NewOrchestrator calls (5 locations)
- `internal/shell/api/resources/deployment.go` - Updated NewOrchestrator calls (4 locations)

### 2. Default Marketplace Templates ✅

**Migration Created:**
- `internal/shell/store/migrations/007_default_templates.up.sql` - 6 templates with pricing
- `internal/shell/store/migrations/007_default_templates.down.sql` - Rollback

**Templates Added:**
1. PostgreSQL Database - $5/month (512MB RAM, 0.5 CPU, 5GB disk)
2. MySQL Database - $5/month (512MB RAM, 0.5 CPU, 5GB disk)
3. Redis Cache - $3/month (256MB RAM, 0.25 CPU, 2GB disk)
4. MongoDB Database - $5/month (512MB RAM, 0.5 CPU, 10GB disk)
5. Nginx Web Server - $2/month (64MB RAM, 0.1 CPU, 512MB disk)
6. Node.js Application - $4/month (256MB RAM, 0.5 CPU, 2GB disk)

### 3. Local E2E Environment ✅

**Setup:**
- APIGate running on localhost:8082
- Hoster running on localhost:8080
- App Proxy on localhost:9091
- All traffic flows through APIGate (single entry point)

**Routes Configuration:**
- Frontend: `/*` (priority 10, auth_required=0)
- API: `/api/*` (priority 50, auth_required=0)
- App Proxy: `*.apps.localhost/*` (priority 100, auth_required=0)

**Testing:**
- Browser-based E2E verified via Chrome DevTools MCP
- Frontend accessible at http://localhost:8082/
- Marketplace showing 7 templates
- All monitoring features working

---

## What Needs to Happen Next

### Priority 1: Fix CI Workflow

**Issue:** npm/rollup build errors in GitHub Actions

**Check Status:**
```bash
gh run list --repo artpar/hoster --limit 3
```

**If Still Failing:**

Option A - Pin rollup version in `web/package.json`:
```json
{
  "overrides": {
    "rollup": "4.9.6"
  }
}
```

Option B - Clean install in CI workflow:
```yaml
- run: rm -rf node_modules package-lock.json && npm install
```

Option C - Try different Node version:
```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '20'
```

### Priority 2: Fix APIGate Auto-Registration

**Current Issue:**
- Auto-registration fails with 401 when accessing `/admin/upstreams`
- Frontend route (`/*`, priority 10) catches all requests including `/admin/*`
- Admin endpoints get proxied to Hoster instead of APIGate

**Solution Options:**

**Option A: Higher Priority Admin Route (RECOMMENDED)**
```sql
-- Add admin route with priority 5 (higher than frontend priority 10)
INSERT INTO routes (
  id, name, path_pattern, match_type, upstream_id, priority, enabled, auth_required
) VALUES (
  'route_apigate_admin',
  'apigate-admin',
  '/admin/*',
  'prefix',
  'upstream_apigate_internal', -- Points to APIGate itself
  5,
  1,
  0
);
```

**Option B: Keep Manual Configuration (CURRENT)**
- Continue using manually configured routes
- Keep `HOSTER_APIGATE_AUTO_REGISTER=false`
- Simple and working for now

### Priority 3: Create Release

**Once CI Passes:**

1. Commit remaining changes:
```bash
git add internal/shell/docker/orchestrator.go
git add internal/shell/api/handler.go
git add internal/shell/api/resources/deployment.go
git add internal/shell/store/migrations/007_default_templates.*.sql
git add CLAUDE.md specs/SESSION-HANDOFF.md
git add .gitignore
git commit -m "feat: Add deployment monitoring and default templates

- Container event recording in orchestrator
- Stats, Logs, Events monitoring endpoints and UI
- 6 default marketplace templates with pricing
- Local E2E environment documentation
- Updated session handoff for monitoring phase"
```

2. Create release tag:
```bash
git tag v0.2.2
git push origin main --tags
```

3. Wait for GitHub Actions to build and create release

### Priority 4: Deploy to Production

**Using Makefile:**
```bash
cd deploy/local
make deploy-release VERSION=v0.2.2
```

**Manually:**
```bash
ssh ubuntu@emptychair.dev
cd /opt/hoster
sudo systemctl stop hoster
sudo wget https://github.com/artpar/hoster/releases/download/v0.2.2/hoster-linux-amd64 -O hoster
sudo chmod +x hoster
sudo systemctl start hoster
sudo systemctl status hoster
```

### Priority 5: Production E2E Testing

**Test Checklist:**

1. **Access Hoster:**
   - [ ] Navigate to https://emptychair.dev
   - [ ] Frontend loads (not 404)
   - [ ] See Hoster marketplace (not APIGate portal)

2. **Browse Marketplace:**
   - [ ] See 7 templates with pricing
   - [ ] PostgreSQL ($5), MySQL ($5), Redis ($3), MongoDB ($5), Nginx ($2), Node.js ($4)
   - [ ] Template details show resource limits

3. **Authentication:**
   - [ ] Sign up via APIGate
   - [ ] Log in successfully
   - [ ] Can access protected endpoints

4. **Deploy Application:**
   - [ ] Select a template (suggest Nginx - simplest)
   - [ ] Click "Deploy Now"
   - [ ] Deployment created successfully
   - [ ] Deployment appears in "My Deployments"

5. **Monitor Deployment:**
   - [ ] Overview tab shows status and details
   - [ ] Events tab shows lifecycle events (created, started)
   - [ ] Stats tab shows CPU, memory, network, disk I/O
   - [ ] Logs tab shows container logs

6. **Access Deployed App:**
   - [ ] Get deployment URL (e.g., https://my-nginx.apps.emptychair.dev)
   - [ ] Navigate to URL
   - [ ] App loads successfully
   - [ ] Routing works correctly

7. **Stop/Start/Restart:**
   - [ ] Stop deployment
   - [ ] Events tab shows "container_stopped" event
   - [ ] Stats show no activity
   - [ ] Start deployment
   - [ ] Events tab shows "container_started" event
   - [ ] App accessible again

8. **Delete Deployment:**
   - [ ] Delete deployment
   - [ ] Deployment removed from list
   - [ ] URL no longer accessible
   - [ ] Container stopped on server

---

## Known Issues and Limitations

### APIGate Auto-Registration

**Issue:** Hoster frontend route catches `/admin/*` requests before they reach APIGate

**Impact:** Auto-registration fails with 401 errors

**Workaround:** Use manual route configuration with `HOSTER_APIGATE_AUTO_REGISTER=false`

**Long-term Fix:** Add higher-priority admin route to APIGate or exclude `/admin/*` in frontend route pattern

### Local Testing Port Requirements

**Issue:** App proxy requires port in URL for local testing

**Example:** `http://myapp.apps.localhost:8082/` (with `:8082`)

**Reason:** APIGate is on non-standard port 8082 locally

**Production:** No issue - production uses standard ports 80/443

---

## Files Ready for Commit

```
modified:   CLAUDE.md
modified:   internal/shell/api/handler.go
modified:   internal/shell/api/resources/deployment.go
modified:   internal/shell/docker/orchestrator.go
modified:   specs/SESSION-HANDOFF.md
modified:   .gitignore

new file:   internal/shell/store/migrations/007_default_templates.up.sql
new file:   internal/shell/store/migrations/007_default_templates.down.sql
```

**Not to commit:**
- `apigate.db*` files (test database files)

---

## Quick Start for Next Session

1. **Read SESSION-HANDOFF.md** - Comprehensive project state
2. **Read CLAUDE.md** - Project architecture and decisions
3. **Check CI status:** `gh run list --repo artpar/hoster --limit 3`
4. **If CI passing:** Create v0.2.2 release and deploy
5. **If CI failing:** Fix build issues first
6. **After deployment:** Run production E2E testing checklist above

---

## Local E2E Environment

**Location:** `/tmp/hoster-e2e-test/`

**Start Services:**
```bash
# Terminal 1: APIGate
cd /tmp/hoster-e2e-test
apigate serve --config apigate.yaml > apigate.log 2>&1 &

# Terminal 2: Hoster
cd /Users/artpar/workspace/code/hoster
HOSTER_DATABASE_DSN=/tmp/hoster-e2e-test/hoster.db \
HOSTER_APIGATE_AUTO_REGISTER=false \
./bin/hoster > /tmp/hoster-e2e-test/hoster.log 2>&1 &
```

**Access:**
- Frontend: http://localhost:8082/
- Marketplace: http://localhost:8082/marketplace
- API: http://localhost:8082/api/v1/*

**Important:** ALL access through APIGate (localhost:8082), never directly to Hoster (localhost:8080)

---

## Success Criteria

### For This Release (v0.2.2)

- ✅ Monitoring features working (Events, Stats, Logs)
- ✅ Default templates in marketplace (6 templates)
- ✅ Local E2E environment functional
- ⏳ CI workflow passing
- ⏳ Production deployment successful
- ⏳ Production E2E testing complete

### For Future Releases

- APIGate auto-registration fixed
- Real authentication enabled (not dev mode)
- Billing integration working
- Multi-node support
- Automatic SSL certificates
