# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 18, 2026)

### Status: Post-MVP, Phases 0-5 COMPLETE

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

**Frontend Build Status:**
```
dist/index.html                   0.54 kB
dist/assets/index-*.css          21.94 kB (gzip: 4.83 kB)
dist/assets/index-*.js          364.07 kB (gzip: 109.41 kB)
```

**Next Step: Phase 6 - Integration & Polish**
- Connect frontend to backend API (ensure CORS, proxy config)
- End-to-end testing with real deployments
- Polish UI/UX based on real usage
- Performance optimization

---

## LAST SESSION SUMMARY (January 18, 2026)

### What Was Accomplished: Phase 5 Frontend Views

**UI Components Created (`web/src/components/ui/`):**
- `Button.tsx` - Variants: default, destructive, outline, secondary, ghost, link
- `Input.tsx` - Form input with focus states
- `Label.tsx` - Form labels
- `Textarea.tsx` - Multi-line text input
- `Select.tsx` - Dropdown with chevron icon
- `Tabs.tsx` - Tab navigation using React Context API
- `Dialog.tsx` - Modal dialog with header, footer, close button, escape key
- `Card.tsx` - Card layout (Header, Title, Description, Content, Footer)
- `Badge.tsx` - Status badges (default, secondary, destructive, outline, success, warning)
- `Skeleton.tsx` - Loading placeholder
- `index.ts` - Barrel export for all UI components

**Marketplace UI (F011) - `web/src/pages/marketplace/`:**
- `MarketplacePage.tsx` - Enhanced with:
  - Search by name and description
  - Sort options (newest, name, price asc/desc)
  - Price filtering (all, free, paid)
  - Results count with filter info
  - Clear filters button
- `TemplateDetailPage.tsx` - Enhanced with:
  - Services list parsed from compose spec
  - Sidebar with pricing card
  - Deploy dialog integration
  - Created/updated dates

**Deploy Dialog (`web/src/components/templates/`):**
- `DeployDialog.tsx` - Full deployment form with:
  - Deployment name input with validation
  - Custom domain (optional)
  - Environment variable overrides (KEY=value format)
  - Price display
  - Error handling

**Deployment Management UI (F012) - `web/src/pages/deployments/`:**
- `DeploymentDetailPage.tsx` - Complete rewrite with tabs:
  - **Overview tab**: Container health cards, resource usage summary, deployment info
  - **Logs tab**: Container filter, tail limit selector (50/100/200/500), refresh button
  - **Stats tab**: Full resource table (CPU, memory, network, block I/O, PIDs)
  - **Events tab**: Color-coded events (error=red, start=green)

**Creator Dashboard UI (F013) - `web/src/pages/creator/`:**
- `CreatorDashboardPage.tsx` - Enhanced with:
  - Stats cards (total templates, deployments, revenue, published)
  - **Templates tab**: Search and status filtering (all/draft/published/deprecated)
  - **Analytics tab**: Deployments by template, revenue by template, creator tips
- `CreateTemplateDialog.tsx` - Template creation form:
  - Name, description, version (semver validation)
  - Monthly price (USD, converted to cents)
  - Docker Compose specification textarea
  - Validation and error handling

**Other Updates:**
- `EmptyState.tsx` - Enhanced to support action objects `{label, onClick}`
- `types.ts` - Added `custom_domain` and `config_overrides` to CreateDeploymentRequest

### Files Modified This Session:
```
web/src/components/ui/Button.tsx (created)
web/src/components/ui/Input.tsx (created)
web/src/components/ui/Label.tsx (created)
web/src/components/ui/Textarea.tsx (created)
web/src/components/ui/Select.tsx (created)
web/src/components/ui/Tabs.tsx (created)
web/src/components/ui/Dialog.tsx (created)
web/src/components/ui/Card.tsx (created)
web/src/components/ui/Badge.tsx (created)
web/src/components/ui/Skeleton.tsx (created)
web/src/components/ui/index.ts (created)
web/src/components/templates/DeployDialog.tsx (created)
web/src/components/templates/CreateTemplateDialog.tsx (created)
web/src/components/common/EmptyState.tsx (modified)
web/src/pages/marketplace/MarketplacePage.tsx (modified)
web/src/pages/marketplace/TemplateDetailPage.tsx (modified)
web/src/pages/deployments/DeploymentDetailPage.tsx (modified)
web/src/pages/creator/CreatorDashboardPage.tsx (modified)
web/src/api/types.ts (modified)
specs/SESSION-HANDOFF.md (this file)
```

---

## Phase 6 Tasks (NEXT SESSION)

### Primary Goals:
1. **Configure API Connection**
   - Set up Vite proxy for development (`vite.config.ts`)
   - Configure `VITE_API_URL` environment variable
   - Handle CORS if backend and frontend are on different origins

2. **End-to-End Testing**
   - Start backend: `make run`
   - Start frontend: `cd web && npm run dev`
   - Test full flow: browse marketplace → deploy template → view deployment → monitor logs/stats

3. **Fix Integration Issues**
   - API response format mismatches
   - Missing error handling
   - Loading state edge cases

4. **Polish UI/UX**
   - Mobile responsiveness
   - Accessibility (ARIA labels, keyboard navigation)
   - Loading skeletons where appropriate
   - Error boundaries

5. **Performance Optimization**
   - Code splitting for routes
   - Lazy loading for heavy components
   - Optimize bundle size

### Verification Commands:
```bash
# Backend
make test      # All backend tests pass
make run       # Start backend on :9090

# Frontend
cd web
npm install    # Install dependencies
npm run build  # Build for production (should succeed)
npm run dev    # Start dev server on :5173
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
