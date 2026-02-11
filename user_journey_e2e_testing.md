# User Journey E2E Testing - Engine Rewrite

**Date:** 2026-02-11
**Environment:** Production-like local setup (built binaries, no dev mode)
**Hoster:** Built from source, running on :8080 (`HOSTER_DATA_DIR=/tmp/hoster-e2e-prod`)
**APIGate:** v0.3.4 released binary, running on :8082 (DB: `/tmp/hoster-e2e-prod/apigate.db`)
**Test user:** testuser1@example.com / test123456
**Admin:** admin@localhost.test / admin123456
**Browser:** Chrome via Chrome DevTools MCP

---

## Setup

1. Built Hoster from source: `go build -o /tmp/hoster-e2e-prod/hoster ./cmd/hoster`
2. Downloaded APIGate v0.3.4 release binary
3. Fresh databases for both services
4. Configured APIGate routes:
   - `hoster-billing`: `/api/v1/deployments*`, priority 55, auth_required=1 (billable)
   - `hoster-api`: `/api/*`, priority 50, auth_required=0 (pass-through, header injection when authed)
   - `hoster-front`: `/*`, priority 10, auth_required=0 (public)
5. Frontend embedded in Go binary via `//go:embed`
6. APIGate setup wizard completed: admin account, upstream to localhost:8080

---

## Journey Results

### Journey 1: Landing Page (Anonymous)
**Status: PASS**

- Navigated to `http://localhost:8082/`
- Hero section: "Deploy apps to your own servers" with description
- CTAs: "Browse Apps" and "Create Account"
- 3-step guide: Deploy from Marketplace, Bring Your Own Servers, Create Templates
- Feature highlights: Real-time Monitoring, SSH Key Management, Self-Hosted & Private
- Footer with copyright 2026
- Navigation shows "Marketplace", "Sign In", "Get Started" for anonymous users

### Journey 2: User Registration (Sign Up)
**Status: PASS**

- Navigated to `/signup`
- Form: Full Name, Email, Password fields + "Create Account" button
- Submitted with: testuser1 / testuser1@example.com / test123456
- Redirected to `/marketplace` after successful registration
- Header shows username "testuser1" and "Sign Out" button
- User created in APIGate, resolved in Hoster via `ResolveUser()`

### Journey 3: User Login (Sign In)
**Status: PASS**

- Navigated to `/login`
- Form: Email, Password fields + "Sign in" button + "Sign up" link
- Submitted with: testuser1@example.com / test123456
- Redirected to `/marketplace`
- JWT token stored in localStorage under `hoster-auth` key (Zustand persist)
- Header shows username and Sign Out button

### Journey 4: Sign Out
**Status: PASS**

- Clicked "Sign Out" button in header
- Redirected to landing page
- Token cleared from localStorage
- Header shows "Sign In" button instead of username

### Journey 5: Anonymous Marketplace Browsing
**Status: PASS** (after fix)

- **Bug found:** Initially, `/api/*` route had `auth_required=1`, blocking anonymous template listing
- **Fix:** Changed `hoster-api` route to `auth_required=0` so anonymous reads pass through
- After fix: Templates visible without login, category pills shown
- "Deploy Now" shows "Authentication Required" dialog with "Sign In" button
- Template detail page accessible without login

### Journey 6: Creator Template Publishing
**Status: PASS** (after fixes)

- Navigated to `/templates` (App Templates page)
- Clicked "Create Template" button, dialog opened
- Filled: Name="Redis Cache", Description, Version="1.0.0", Compose spec, Category="database"
- **Bug found:** `NOT NULL constraint failed: templates.resources_cpu_cores` - resource fields missing defaults
- **Fix:** Added `.WithDefault(0)` to resources_cpu_cores, resources_memory_mb, resources_disk_mb, price_monthly_cents
- Template created successfully, shown on page with "Draft" badge
- Clicked "Publish" button
- **Bug found:** POST `/api/v1/templates/{id}/publish` returned 404 - missing Actions declaration
- **Fix:** Added `Actions: []CustomAction{{Name: "publish", Method: "POST"}}` to template resource
- After fix: Publish succeeded, "Draft" badge removed
- Template appeared in Marketplace under "Database" category

### Journey 7: Marketplace Template Detail
**Status: PASS**

- Clicked template card in marketplace
- Detail page shows: name, "Published" badge, version, description
- "Included Services" section lists services from compose spec
- Full compose spec shown in code block
- Price info: "Free / month"
- "Deploy Now" button prominent
- "Published" and "Last Updated" dates shown
- "Back to Marketplace" button

### Journey 8: Deploy from Marketplace
**Status: PASS** (after fixes)

- Clicked "Deploy Now" on template detail
- Dialog opened with: auto-generated deployment name, custom domain field, env overrides textarea, monthly cost
- **Bug found:** `NOT NULL constraint failed: deployments.template_version` - frontend doesn't send template_version
- **Fix:** Made `template_version` nullable + added BeforeCreate hook to copy version from template
- **Bug found:** RefField `template_id` received reference_id string ("tmpl_12d24030") but DB expected integer FK
- **Fix:** Added `resolveRefFields()` helper in create/update handlers to convert reference_ids to integer PKs
- After fixes: Deployment created, auto-started, navigated to deployment detail page
- Status correctly shows "Failed" with error "no online nodes available" (expected - no nodes)

### Journey 9: Deployment Detail Page
**Status: PASS**

- URL: `/deployments/{uuid}`
- Header: deployment name, status badge ("Failed"), action buttons (Start, Delete)
- Error message displayed: "no online nodes available"
- Tabs: Overview, Logs, Stats, Events, Domains
- Deployment info: Created date, Last Updated date
- "Back to Deployments" button

### Journey 10: My Deployments List
**Status: PASS**

- Navigated to `/deployments`
- Page header: "My Deployments" with description
- "New Deployment" link to marketplace
- Deployment card shows: name, status badge ("Failed"), created date
- Clicking card navigates to detail page

### Journey 11: Dashboard
**Status: PASS**

- Navigated to `/dashboard`
- Stats cards: Deployments (0 running / 1 total), App Templates (1 published / 1 total), Nodes (0 online / 0 total), Monthly Revenue ($0.00)
- Recent Deployments: shows deployment with template name and status
- Node Health: "No nodes configured" with "Add a node" link
- Template Performance: "Redis Cache - 1 deployment, $0.00/mo"

### Journey 12: Nodes Page
**Status: PASS**

- Navigated to `/nodes`
- Tab navigation: Nodes, Cloud Servers, Credentials
- Empty state: "No worker nodes" with guide
- Action links: "Add Existing Server", "Create Cloud Server"
- "SSH Keys" link
- Node Setup Guide section

### Journey 13: SSH Keys Page
**Status: PASS**

- Navigated to `/ssh-keys`
- Header with description about AES-256 encryption
- Empty state: "No SSH keys" with explanation
- "Add SSH Key" button (top and empty state)

---

## Bugs Found & Fixed

| # | Bug | Root Cause | Fix | File(s) |
|---|-----|-----------|-----|---------|
| 1 | No embedded frontend | `spaHandler()` was placeholder redirect to /health | Added `//go:embed`, proper file server with SPA fallback | `setup.go` |
| 2 | Auth failing on API routes | `hoster-api` route had `auth_required=0`, no header injection | Updated route to auth_required=1 with header injection | APIGate DB |
| 3 | NOT NULL on resource fields | Default values not set for `resources_cpu_cores` etc. | Added `.WithDefault(0)` to resource field definitions | `resources.go` |
| 4 | FK integer IDs in API responses | `creator_id` returned `1` instead of reference_id | Added ref resolution in `stripFields()`, `GetRefIDByIntID()` | `api.go`, `store.go` |
| 5 | Missing action endpoints (404) | Template/Deployment resources didn't declare `Actions` | Added `Actions: []CustomAction{...}` to resource defs | `resources.go` |
| 6 | Bool fields as 0/1 not true/false | SQLite stores bools as integers, `decodeRow` didn't coerce | Added bool coercion in `decodeRow()` for TypeBool fields | `store.go` |
| 7 | template_version NOT NULL | Frontend doesn't send template_version on deploy | Made nullable + BeforeCreate hook copies from template | `resources.go`, `setup.go` |
| 8 | RefField IDs not resolved on create | Frontend sends reference_ids, DB expects integer FKs | Added `resolveRefFields()` in create/update handlers | `api.go` |
| 9 | Anonymous marketplace blocked | `/api/*` route had `auth_required=1` | Changed to `auth_required=0` for pass-through reads | APIGate DB |

---

## Architecture Observations

1. **Generic engine works well** - All CRUD, state machines, and validation handled by the engine. No per-entity handler code needed except custom actions (publish, start, stop).

2. **APIGate route configuration is critical** - The split between billing (auth_required=1 for writes) and pass-through (auth_required=0 for reads) is essential for anonymous marketplace browsing.

3. **Type coercion needed** - SQLite doesn't have native bool/JSON types. The engine's `decodeRow()` must handle: `[]byte` to `string`, integer to `bool`, JSON string to parsed object.

4. **Ref resolution is bidirectional** - On create/update: resolve reference_id to integer PK. On read: resolve integer PK to reference_id. Both handled generically now.

5. **Frontend dialog limitation** - Custom Dialog component doesn't set `role="dialog"`, making it invisible to accessibility snapshots. Must use screenshots or JS evaluation for dialog interaction.

---

## Summary

**13 journeys tested, all passing.** 9 bugs found and fixed during testing. The engine rewrite is functionally complete for all core user flows. The deployment correctly fails with "no online nodes available" since no compute nodes are configured in the test environment.

### Not Testable in This Environment
- Node registration and health check flow (needs SSH-accessible server)
- Cloud provisioning flow (needs cloud credentials)
- Actual container deployment on remote node
- Deployment lifecycle: start -> running -> stop -> stopped -> restart
- Real-time monitoring (CPU, memory, logs) on running containers
