# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 18, 2026)

### Status: Post-MVP, Phases 0-5 COMPLETE, Phase 6 IN PROGRESS

**What's Done:**
- MVP complete (core deployment loop works)
- 460+ tests passing (backend)
- Phase -1 (ADR & Spec Updates) COMPLETE
- Phase 0 (API Layer Migration) COMPLETE
- Phase 1 (APIGate Auth Integration) COMPLETE
- Phase 2 (Billing Integration) COMPLETE
- Phase 3 (Monitoring Backend) COMPLETE
- Phase 4 (Frontend Foundation) COMPLETE
- Phase 5 (Frontend Views) COMPLETE
- Phase 5 Manual Testing COMPLETE (via Chrome DevTools MCP)
- Phase 6 Integration bug fixes COMPLETE

**Frontend Build Status:**
```
dist/index.html                   0.54 kB
dist/assets/index-*.css          21.94 kB (gzip: 4.83 kB)
dist/assets/index-*.js          364.07 kB (gzip: 109.41 kB)
```

**All UI Components Tested & Working:**
- Marketplace page with search/sort/filter
- Template detail page with pricing
- Deploy dialog for creating deployments
- Deployment detail page with monitoring tabs
- Creator dashboard for template management

**Next Step: Phase 6 Remaining - Polish & Real Testing**
- Polish UI/UX (mobile, accessibility, error boundaries)
- Test with real Docker containers running
- Performance optimization

---

## LAST SESSION SUMMARY (January 18, 2026)

### What Was Accomplished: Phase 5 Manual Testing & Bug Fixes

**Testing Completed via Chrome DevTools MCP:**
- Marketplace page - 29 templates displaying with search/sort/filter
- Template detail page - Pricing, description, deploy button working
- Deploy dialog - Template selection and deployment creation (201 response)
- Deployment detail page - All tabs working (Overview, Logs, Stats, Events)
- Creator Dashboard - Template creation, listing, and management

**Bug Fixes Applied During Testing:**

| Issue | Root Cause | Fix |
|-------|------------|-----|
| NaN price display | Used `price_cents` instead of `price_monthly_cents` | Updated field name in TemplateDetailPage.tsx and DeployDialog.tsx |
| api2go 406 error | Deployment resource missing `UnmarshalToOneRelations` interface | Added `SetToOneReferenceID` method to deployment.go |
| 401 Authentication error | Auth middleware "none" mode didn't set authenticated context | Fixed middleware to set dev user context in "none" mode |
| React null errors | API returned `logs: null`, `events: null`, `containers: null` | Added null safety checks (`logs?.logs && logs.logs.length > 0`) |
| Creator Dashboard empty | localStorage userId didn't match backend dev mode | Fixed localStorage to use "dev-user" |

**Backend Fixes:**
- `internal/shell/api/resources/deployment.go` - Added `SetToOneReferenceID` method for api2go relationship handling
- `internal/shell/api/middleware/auth.go` - Dev mode now sets authenticated context with `dev-user` userId

**Frontend Fixes:**
- `web/src/pages/marketplace/TemplateDetailPage.tsx` - Fixed price field and status badge
- `web/src/components/templates/DeployDialog.tsx` - Fixed price field display
- `web/src/pages/deployments/DeploymentDetailPage.tsx` - Added null safety for logs/stats/events arrays
- `web/src/api/client.ts` - Added auth headers from localStorage for API requests

### Files Modified This Session:
```
web/src/pages/marketplace/TemplateDetailPage.tsx (fixed price field)
web/src/components/templates/DeployDialog.tsx (fixed price field)
web/src/pages/deployments/DeploymentDetailPage.tsx (null safety)
web/src/api/client.ts (auth headers from localStorage)
internal/shell/api/resources/deployment.go (SetToOneReferenceID)
internal/shell/api/middleware/auth.go (dev user context)
specs/SESSION-HANDOFF.md (this file)
```

### Testing Environment Notes:
- Backend runs on port 9090: `HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=none ./bin/hoster`
- Frontend dev server proxies to backend via vite.config.ts
- Dev mode uses `dev-user` as the authenticated user ID

---

## Phase 6 Tasks (NEXT SESSION)

### Completed in This Session:
- [x] Basic API connection working (Vite proxy configured)
- [x] End-to-end testing via Chrome DevTools MCP
- [x] Fixed api2go relationship handling (SetToOneReferenceID)
- [x] Fixed auth middleware dev mode
- [x] Fixed frontend null safety issues
- [x] Fixed price field mapping

### Remaining Tasks:

1. **Polish UI/UX**
   - Mobile responsiveness
   - Accessibility (ARIA labels, keyboard navigation)
   - Loading skeletons where appropriate
   - Error boundaries
   - Toast notifications for actions

2. **Real Deployment Testing**
   - Test with actual Docker containers running
   - Verify logs streaming works with real output
   - Verify stats collection from running containers
   - Test start/stop/restart lifecycle

3. **Performance Optimization**
   - Code splitting for routes
   - Lazy loading for heavy components
   - Optimize bundle size

4. **Edge Cases & Error Handling**
   - Network failure handling
   - Concurrent operation handling
   - Long-running operation feedback

### Verification Commands:
```bash
# Backend
make test      # All backend tests pass
HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=none make run  # Start backend on :9090

# Frontend
cd web
npm install    # Install dependencies
npm run build  # Build for production (should succeed)
npm run dev    # Start dev server on :3000 (proxies to :9090)
```

---

## Phase 1: Context Loading (MANDATORY)

### Step 1: Read Core Documents (in order)

```bash
# Run this to verify project exists
ls -la
```

Read these files in this exact order:

1. **`CLAUDE.md`** - Project memory, decisions, current state
2. **`specs/README.md`** - How specs work
3. **`specs/decisions/ADR-000-stc-methodology.md`** - THE methodology
4. **`specs/decisions/ADR-001-docker-direct.md`** - Architecture decision
5. **`specs/decisions/ADR-002-values-as-boundaries.md`** - Code organization

### Step 2: Read Post-MVP ADRs (for UI/API work)

These are critical for Phase 6:

6. **`specs/decisions/ADR-003-jsonapi-api2go.md`** - JSON:API with api2go
7. **`specs/decisions/ADR-004-reflective-openapi.md`** - OpenAPI generation
8. **`specs/decisions/ADR-005-apigate-integration.md`** - Auth/billing via APIGate
9. **`specs/decisions/ADR-006-frontend-architecture.md`** - React + Vite frontend
10. **`specs/decisions/ADR-007-uiux-guidelines.md`** - UI/UX consistency

### Step 3: Read Feature Specs (for implementation)

```
specs/features/F008-authentication.md     - Header-based auth
specs/features/F009-billing-integration.md - Usage tracking
specs/features/F010-monitoring-dashboard.md - Health/logs/stats
specs/features/F011-marketplace-ui.md      - Template browsing (IMPLEMENTED)
specs/features/F012-deployment-management-ui.md - Deployment controls (IMPLEMENTED)
specs/features/F013-creator-dashboard-ui.md - Template management (IMPLEMENTED)
```

### Step 4: Read Domain Specs

```
specs/domain/template.md    - Template entity + JSON:API definition
specs/domain/deployment.md  - Deployment entity + JSON:API definition
specs/domain/monitoring.md  - Health, stats, logs, events types
specs/domain/user-context.md - AuthContext from APIGate headers
```

### Step 5: Verify Understanding

After reading, you should know:
- [ ] What is Hoster? (Deployment marketplace platform)
- [ ] What is STC? (Spec -> Test -> Code)
- [ ] What is "Values as Boundaries"? (Pure core, thin shell)
- [ ] What is JSON:API? (Standardized API format via api2go)
- [ ] What is APIGate? (External auth/billing, injects X-User-ID headers)
- [ ] Where do specs go? (`specs/` directory)
- [ ] Where does core code go? (`internal/core/`)
- [ ] Where does I/O code go? (`internal/shell/`)
- [ ] What libraries to use? (Listed in CLAUDE.md)

### Step 6: Verify Tests Pass

```bash
make test
```

If tests fail, something is broken. Fix before proceeding.

---

## Phase 2: Task Understanding

### Step 7: Check Implementation Plan

Read the plan file for detailed implementation phases:
```
/Users/artpar/.claude/plans/merry-baking-rain.md
```

**Implementation Phases:**
- Phase -1: ADR & Spec Updates (COMPLETE)
- Phase 0: API Layer Migration (JSON:API + OpenAPI) (COMPLETE)
- Phase 1: APIGate Integration (Backend Auth) (COMPLETE)
- Phase 2: Billing Integration (COMPLETE)
- Phase 3: Monitoring Backend (COMPLETE)
- Phase 4: Frontend Foundation (COMPLETE)
- Phase 5: Frontend Views (COMPLETE)
- Phase 6: Integration & Polish <- NEXT

### Step 8: Check Current Status

Read `CLAUDE.md` section "Current Implementation Status" to understand:
- What's DONE
- What's TODO
- What's BLOCKED

### Step 9: Check Agile Project

```
Use mcp__agile__workflow_execute with workflow: "backlog_status"
and project_id: "HOSTER" to see current tasks.
```

### Step 10: Understand User's Request

Now you can ask the user what they want to do. Compare against:
- What's already implemented (don't redo)
- What's in the TODO list
- What's explicitly NOT supported (don't implement)

---

## Phase 3: Before Making Changes

### Step 11: Identify Relevant Specs

For ANY change:
1. Find the relevant spec in `specs/`
2. If no spec exists -> WRITE SPEC FIRST
3. If spec exists -> READ IT before changing code

### Step 12: Pre-Flight Checklist

Before writing ANY code, verify:

- [ ] Spec exists for this feature/change
- [ ] I understand the spec's acceptance criteria
- [ ] I know what's "NOT Supported" (don't implement those)
- [ ] Tests exist (or I'll write them first)
- [ ] I know which directory: `internal/core/` or `internal/shell/`
- [ ] I'm using the approved libraries (check CLAUDE.md)

---

## Phase 4: Making Changes (STC Flow)

### For New Features

```
1. SPEC   -> Create specs/features/F###-name.md
2. TEST   -> Create internal/core/xxx/feature_test.go (failing tests)
3. CODE   -> Create internal/core/xxx/feature.go (make tests pass)
4. VERIFY -> make test
```

### For Bug Fixes

```
1. SPEC   -> Update spec if behavior was wrong
2. TEST   -> Add test that demonstrates the bug
3. CODE   -> Fix code to pass test
4. VERIFY -> make test
```

### For Refactoring

```
1. VERIFY -> make test (all pass before)
2. REFACTOR -> Make changes
3. VERIFY -> make test (all pass after)
```

---

## Phase 5: After Making Changes

### Step 13: Verify Sync

After any change:
- [ ] Spec still matches implementation
- [ ] Tests still match spec
- [ ] All tests pass (`make test`)

### Step 14: Update CLAUDE.md

If you:
- Completed a TODO item -> Move to DONE
- Made a new decision -> Document in CLAUDE.md
- Added new spec -> Reference in CLAUDE.md
- Changed architecture -> Update ADR or create new one

### Step 15: Update Agile Project

```
Use mcp__agile__task_transition to update task status.
```

---

## Key Library Changes (Post-MVP)

| Purpose | Old | New |
|---------|-----|-----|
| HTTP router | chi/v5 | gorilla/mux (api2go built-in) |
| API format | custom JSON | JSON:API via api2go |
| OpenAPI | manual | reflective generation |

**Backend Dependencies:**
- `github.com/manyminds/api2go` - JSON:API implementation
- `github.com/gorilla/mux` - Router (api2go support)
- `github.com/getkin/kin-openapi` - OpenAPI 3.0 types

**Frontend Dependencies (web/package.json):**
- React 19 + React DOM 19
- React Router DOM 7.1
- TanStack Query 5.62
- Zustand 5.0
- Vite 6.0
- TailwindCSS 3.4
- Lucide React 0.469 (icons)

---

## Frontend File Structure

```
web/
├── src/
│   ├── api/
│   │   ├── client.ts         # JSON:API fetch wrapper
│   │   ├── types.ts          # TypeScript types
│   │   ├── templates.ts      # Template API
│   │   ├── deployments.ts    # Deployment API
│   │   └── monitoring.ts     # Monitoring API
│   ├── components/
│   │   ├── common/
│   │   │   ├── LoadingSpinner.tsx
│   │   │   ├── EmptyState.tsx
│   │   │   └── StatusBadge.tsx
│   │   ├── layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── Layout.tsx
│   │   ├── templates/
│   │   │   ├── TemplateCard.tsx
│   │   │   ├── DeployDialog.tsx
│   │   │   └── CreateTemplateDialog.tsx
│   │   ├── deployments/
│   │   │   └── DeploymentCard.tsx
│   │   └── ui/
│   │       ├── Button.tsx
│   │       ├── Input.tsx
│   │       ├── Label.tsx
│   │       ├── Textarea.tsx
│   │       ├── Select.tsx
│   │       ├── Tabs.tsx
│   │       ├── Dialog.tsx
│   │       ├── Card.tsx
│   │       ├── Badge.tsx
│   │       ├── Skeleton.tsx
│   │       └── index.ts
│   ├── hooks/
│   │   ├── useTemplates.ts   # TanStack Query hooks
│   │   ├── useDeployments.ts
│   │   └── useMonitoring.ts
│   ├── pages/
│   │   ├── marketplace/
│   │   │   ├── MarketplacePage.tsx
│   │   │   └── TemplateDetailPage.tsx
│   │   ├── deployments/
│   │   │   ├── MyDeploymentsPage.tsx
│   │   │   └── DeploymentDetailPage.tsx
│   │   ├── creator/
│   │   │   └── CreatorDashboardPage.tsx
│   │   └── NotFoundPage.tsx
│   ├── stores/
│   │   └── authStore.ts      # Zustand store
│   ├── lib/
│   │   └── cn.ts             # clsx + tailwind-merge
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css
├── package.json
├── vite.config.ts
├── tailwind.config.ts
├── postcss.config.js
└── tsconfig.json
```

---

## Common Mistakes to Avoid

| Mistake | Why It's Bad | Prevention |
|---------|--------------|------------|
| Writing code without reading specs | You'll implement wrong behavior | Always read spec first |
| Writing code before tests | No safety net for refactoring | Write test first, see it fail |
| Putting I/O in `internal/core/` | Breaks architecture, needs mocks | Keep core pure |
| Using different library | Inconsistency, harder to maintain | Check CLAUDE.md library list |
| Implementing "NOT Supported" items | Scope creep, wasted effort | Read spec's NOT Supported section |
| Skipping `make test` | Broken code goes unnoticed | Run after every change |
| Not updating CLAUDE.md | Next session loses context | Update after significant changes |
| Ignoring ADR-007 UI guidelines | Inconsistent UI | Follow semantic colors, patterns |

---

## Quick Reference: Where Things Go

| What | Where | Example |
|------|-------|---------|
| Domain specs | `specs/domain/` | `template.md` |
| Feature specs | `specs/features/` | `F008-authentication.md` |
| ADRs | `specs/decisions/` | `ADR-003-jsonapi-api2go.md` |
| Pure logic | `internal/core/` | `domain/template.go` |
| I/O code | `internal/shell/` | `docker/client.go` |
| Unit tests | Same dir as code | `template_test.go` |
| E2E tests | `tests/e2e/` | `deploy_test.go` |
| Sample templates | `examples/` | `wordpress/compose.yml` |
| Frontend | `web/` | React + Vite app |
| UI components | `web/src/components/ui/` | `Button.tsx` |
| Page components | `web/src/pages/` | `MarketplacePage.tsx` |

---

## Emergency Recovery

### If Tests Are Failing

```bash
# See what's broken
make test 2>&1 | grep FAIL

# Check recent changes
git log --oneline -10

# Revert if needed
git checkout HEAD~1 -- <file>
```

### If Frontend Won't Build

```bash
cd web
rm -rf node_modules
npm install
npm run build
```

### If You're Lost

1. Re-read `CLAUDE.md` from the beginning
2. Run `make test` to verify baseline
3. Run `cd web && npm run build` to verify frontend
4. Read the specific spec for what you're working on
5. Read the plan file for implementation phases
6. Ask user for clarification

### If Spec Doesn't Exist

DO NOT write code. Instead:

1. Create spec file
2. Write requirements
3. Write "NOT Supported" section
4. Get user confirmation
5. Then proceed with tests and code

---

## Session End Checklist

Before ending a session:

- [ ] All tests pass (`make test`)
- [ ] Frontend builds (`cd web && npm run build`)
- [ ] CLAUDE.md is updated with:
  - [ ] New DONE items
  - [ ] New TODO items
  - [ ] Any new decisions
- [ ] Agile project updated
- [ ] User informed of current state
- [ ] This file updated if project state changed significantly
