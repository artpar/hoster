# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 19, 2026)

### Status: Post-MVP, All Phases COMPLETE, Creator Worker Nodes ALL PHASES COMPLETE

**What's Done:**
- MVP complete (core deployment loop works)
- All backend tests passing (500+ tests)
- Phase -1 (ADR & Spec Updates) COMPLETE
- Phase 0 (API Layer Migration) COMPLETE
- Phase 1 (APIGate Auth Integration) COMPLETE
- Phase 2 (Billing Integration) COMPLETE
- Phase 3 (Monitoring Backend) COMPLETE
- Phase 4 (Frontend Foundation) COMPLETE
- Phase 5 (Frontend Views) COMPLETE
- Phase 5 Manual Testing COMPLETE (via Chrome DevTools MCP)
- Phase 6 Integration bug fixes COMPLETE
- **Creator Worker Nodes Feature - ALL 7 PHASES COMPLETE**
- **Generic API/Hook factories implemented for code reuse**

**Frontend Build Status:**
```
dist/index.html                   0.54 kB
dist/assets/index-*.css          23.47 kB (gzip: 5.02 kB)
dist/assets/index-*.js          383.96 kB (gzip: 114.21 kB)
```

**All UI Components Tested & Working:**
- Marketplace page with search/sort/filter
- Template detail page with pricing
- Deploy dialog for creating deployments
- Deployment detail page with monitoring tabs
- Creator dashboard for template management
- **Nodes tab for worker node management**

**Creator Worker Nodes Feature Progress:**
- Phase 1 (Domain Model & Scheduler): COMPLETE
- Phase 2 (Database Layer): COMPLETE
- Phase 3 (SSH Docker Client via Minion): COMPLETE
- Phase 4 (Scheduler Integration): COMPLETE
- Phase 5 (Node API Resource): COMPLETE
- Phase 6 (Frontend Nodes Tab): COMPLETE
- Phase 7 (Health Checker Worker): COMPLETE

**Creator Worker Nodes Feature: FULLY COMPLETE**

All phases of the Creator Worker Nodes feature are now implemented. The feature includes:
- Node and SSH Key domain models with full validation
- Database layer with encrypted SSH key storage (AES-256-GCM)
- SSH-based Docker client via the minion protocol
- Intelligent scheduler for node selection based on capabilities and capacity
- JSON:API resources for nodes and SSH keys
- Frontend UI for node management in Creator Dashboard
- Background health checker worker for periodic node monitoring

---

## LAST SESSION SUMMARY (January 19, 2026)

### What Was Accomplished: Phase 7 (Health Checker Worker) COMPLETE

This session implemented the Health Checker Worker, completing all 7 phases of the Creator Worker Nodes feature.

**1. Store Interface Extension:**

Added `ListCheckableNodes(ctx context.Context) ([]domain.Node, error)` to Store interface:
- Returns all nodes NOT in maintenance mode (online and offline)
- Used by health checker to determine which nodes to check
- Implemented in SQLite store with proper filtering

**2. Health Checker Worker:**

Created `internal/shell/workers/health_checker.go`:
- Configurable health check interval (default: 60 seconds)
- Configurable timeout per node (default: 10 seconds)
- Concurrent node checking with configurable max concurrency (default: 5)
- Connects to nodes via SSH and pings Docker daemon
- Updates node status (online/offline) and last_health_check timestamp
- Records error messages for offline nodes
- On-demand checking via CheckNodeNow() and CheckAllNow()
- Graceful start/stop lifecycle management

**3. Configuration:**

Added `NodesConfig` to application configuration:
- `enabled` - Enable/disable remote nodes (default: false)
- `encryption_key` - 32-byte key for AES-256-GCM SSH key encryption
- `health_check_interval` - Check interval (default: 60s)
- `health_check_timeout` - Per-node timeout (default: 10s)
- `health_check_max_concurrent` - Max concurrent checks (default: 5)

Environment variables:
- `HOSTER_NODES_ENABLED=true`
- `HOSTER_NODES_ENCRYPTION_KEY=<32-byte-key>`
- `HOSTER_NODES_HEALTH_CHECK_INTERVAL=60s`

**4. Server Integration:**

Updated `cmd/hoster/server.go`:
- Creates NodePool when nodes are enabled
- Starts health checker on server start
- Stops health checker and closes NodePool on shutdown
- Validates encryption key is exactly 32 bytes

**5. Tests:**

Created `internal/shell/workers/health_checker_test.go`:
- Configuration tests
- Lifecycle (start/stop) tests
- Run cycle tests
- Concurrency limit tests
- Node not found handling
- Maintenance mode skipping

Added SQLite store test for ListCheckableNodes.

**Files Created/Modified This Session:**
```
# New files
internal/shell/workers/health_checker.go       # Health checker worker
internal/shell/workers/health_checker_test.go  # Health checker tests

# Modified files
internal/shell/store/store.go        # Added ListCheckableNodes interface
internal/shell/store/sqlite.go       # Implemented ListCheckableNodes
internal/shell/store/sqlite_test.go  # Added ListCheckableNodes test
internal/shell/api/handler_test.go   # Added stub for ListCheckableNodes
cmd/hoster/config.go                 # Added NodesConfig
cmd/hoster/server.go                 # Integrated health checker
```

**All Tests Pass:** 500+ tests across all packages

---

## PREVIOUS SESSION SUMMARY (January 19, 2026)

### What Was Accomplished: Phase 6 (Frontend Nodes Tab) COMPLETE + Generic Factories

This session implemented the frontend Nodes tab AND created reusable factories to reduce code duplication.

**1. Generic API Client Factory:**

Created `web/src/api/createResourceApi.ts`:
- Type-safe JSON:API client factory
- Standard CRUD operations (list, get, create, update, delete)
- Support for custom actions (e.g., maintenance mode, publish)
- Configurable update/delete support

**2. Generic TanStack Query Hooks Factory:**

Created `web/src/hooks/createResourceHooks.ts`:
- Query keys management with standard pattern
- Automatic cache invalidation on mutations
- `createIdActionHook` for custom id-based actions
- `createActionHook` for generic action hooks

**3. Frontend Nodes Implementation:**

Created node API and hooks using the factories:
- `web/src/api/nodes.ts` - Node API client with maintenance mode actions
- `web/src/api/ssh-keys.ts` - SSH Key API client (immutable, no update)
- `web/src/hooks/useNodes.ts` - Node query hooks + maintenance actions
- `web/src/hooks/useSSHKeys.ts` - SSH Key query hooks

Created node UI components:
- `web/src/components/nodes/NodeCard.tsx` - Node display with capacity bars
- `web/src/components/nodes/AddNodeDialog.tsx` - Create node form
- `web/src/components/nodes/AddSSHKeyDialog.tsx` - Upload SSH key form
- `web/src/components/nodes/index.ts` - Component exports

Updated:
- `web/src/components/common/StatusBadge.tsx` - Added node statuses (online, offline, maintenance)
- `web/src/pages/creator/CreatorDashboardPage.tsx` - Added Nodes tab with full node management UI

**4. Refactored Existing Code to Use Factories:**

Refactored templates to use generic factories:
- `web/src/api/templates.ts` - Now uses createResourceApi with publish action
- `web/src/hooks/useTemplates.ts` - Now uses createResourceHooks + createIdActionHook

**Code Reduction from Factories:**
| File | Before | After | Savings |
|------|--------|-------|---------|
| nodes.ts | ~45 lines | ~27 lines | 40% |
| ssh-keys.ts | ~27 lines | ~21 lines | 22% |
| useNodes.ts | ~73 lines | ~34 lines | 53% |
| useSSHKeys.ts | ~45 lines | ~27 lines | 40% |
| templates.ts | ~44 lines | ~25 lines | 43% |
| useTemplates.ts | ~73 lines | ~29 lines | 60% |

**Files Created/Modified This Session:**
```
# New files
web/src/api/createResourceApi.ts       # Generic API client factory
web/src/api/nodes.ts                   # Node API client
web/src/api/ssh-keys.ts                # SSH Key API client
web/src/hooks/createResourceHooks.ts   # Generic hooks factory
web/src/hooks/useNodes.ts              # Node query hooks
web/src/hooks/useSSHKeys.ts            # SSH Key query hooks
web/src/components/nodes/NodeCard.tsx         # Node card component
web/src/components/nodes/AddNodeDialog.tsx    # Add node dialog
web/src/components/nodes/AddSSHKeyDialog.tsx  # Add SSH key dialog
web/src/components/nodes/index.ts             # Component exports

# Modified files
web/src/api/templates.ts               # Refactored to use factory
web/src/hooks/useTemplates.ts          # Refactored to use factory
web/src/components/common/StatusBadge.tsx     # Added node statuses
web/src/pages/creator/CreatorDashboardPage.tsx # Added Nodes tab
```

**Frontend Build Status:** SUCCESS (383.96 kB gzip: 114.21 kB)

---

### Architecture Overview:
```
┌──────────────────────────────────────────────────────────────────────────┐
│ PRODUCTION PATH (setup.go → resources/deployment.go)                      │
│                                                                           │
│ DeploymentResource                                                        │
│  ├─ Scheduler (scheduler.Service)                                         │
│  │   ├── store.ListOnlineNodes()                                         │
│  │   ├── corescheduler.Schedule() (pure algorithm)                       │
│  │   └── nodePool.GetClient() or local fallback                          │
│  └─ docker.Orchestrator (created per-request with scheduled client)      │
└──────────────────────────────────────────────────────────────────────────┘
             │
             │ For nodeID="local" → uses local Docker client
             │ For nodeID="node-X" → uses SSHDockerClient from NodePool
             ▼
┌────────────────┐                      ┌────────────────┐
│ Local Docker   │  OR                  │ Remote Node    │
│ daemon         │                      │ via SSH+Minion │
└────────────────┘                      └────────────────┘
```

**Key Insight:** Production uses `setup.go` → `DeploymentResource` NOT `handler.go`.
The handler.go path is primarily for tests and non-api2go endpoints.

### Generic Factory Pattern:
```
┌─────────────────────────────────────────────────────────────────┐
│ createResourceApi<Resource, CreateReq, UpdateReq, CustomActions>│
│   → ResourceApi { list, get, create, update, delete }           │
│   → Custom actions (e.g., publish, enterMaintenance)            │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ createResourceHooks({ resourceName, api })                      │
│   → keys, useList, useGet, useCreate, useUpdate, useDelete      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ createIdActionHook(keys, actionFn)                              │
│   → Custom mutation hook for id-based actions                   │
└─────────────────────────────────────────────────────────────────┘
```

### Testing Environment Notes:
- Backend runs on port 9090: `HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=dev ./bin/hoster`
- Frontend dev server proxies to backend via vite.config.ts
- Dev mode (`HOSTER_AUTH_MODE=dev`) auto-authenticates as `dev-user`
- Auth modes: `header` (production), `dev` (local development), `none` (unauthenticated)

---

## Creator Worker Nodes - Task Status (ALL PHASES COMPLETE)

### Completed (All 7 Phases):
- [x] `specs/domain/node.md` - Node entity specification
- [x] `internal/core/domain/node.go` - Node domain model (51 tests)
- [x] `internal/core/scheduler/scheduler.go` - Pure scheduling algorithm (26 tests)
- [x] `internal/core/crypto/encryption.go` - AES-256-GCM encryption (26 tests)
- [x] `internal/core/auth/context.go` - Added AllowedCapabilities to PlanLimits
- [x] `internal/core/domain/template.go` - Added RequiredCapabilities field
- [x] Database migration 005_nodes (up/down)
- [x] Store interface + SQLite implementation for nodes/SSH keys (20 tests)
- [x] Handler test stubs for new Store interface
- [x] `internal/core/minion/protocol.go` - Minion protocol types (20 tests)
- [x] `cmd/hoster-minion/` - Minion binary with 18 commands
- [x] `internal/shell/docker/ssh_client.go` - SSHDockerClient implementing Client
- [x] `internal/shell/docker/node_pool.go` - Connection pool with lazy init
- [x] `internal/shell/docker/minion_embed.go` - Embedded minion binaries
- [x] Makefile updates for minion build
- [x] `internal/shell/scheduler/service.go` - Scheduling service with I/O
- [x] `internal/shell/scheduler/service_test.go` - 9 tests for scheduling service
- [x] `internal/shell/api/handler.go` - Scheduler integration
- [x] `internal/shell/api/resources/node.go` - Node JSON:API resource
- [x] `internal/shell/api/resources/ssh_key.go` - SSH Key resource
- [x] Authorization checks (CanManageNode, CanViewNode, CanCreateNode)
- [x] SSH key encryption with AES-256-GCM
- [x] Maintenance mode endpoints
- [x] OpenAPI documentation for new resources
- [x] `web/src/api/createResourceApi.ts` - Generic API factory
- [x] `web/src/hooks/createResourceHooks.ts` - Generic hooks factory
- [x] `web/src/api/nodes.ts` - Node API client
- [x] `web/src/api/ssh-keys.ts` - SSH Key API client
- [x] `web/src/hooks/useNodes.ts` - Node query hooks
- [x] `web/src/hooks/useSSHKeys.ts` - SSH Key query hooks
- [x] `web/src/components/nodes/` - NodeCard, AddNodeDialog, AddSSHKeyDialog
- [x] Nodes tab in Creator Dashboard

### Phase 7 - Health Checker Worker (COMPLETE):
- [x] `internal/shell/workers/health_checker.go` - Periodic health check worker (11 tests)
- [x] `internal/shell/store/store.go` - Added ListCheckableNodes interface method
- [x] `internal/shell/store/sqlite.go` - Implemented ListCheckableNodes
- [x] `cmd/hoster/config.go` - Added NodesConfig
- [x] `cmd/hoster/server.go` - Integrated health checker with server lifecycle
- [x] Background goroutine that pings nodes periodically
- [x] Updates node status (online/offline) and last_health_check timestamp
- [x] Records error messages for offline nodes
- [x] Configurable check interval (default: 60s), timeout, and concurrency

### Backend API Endpoints (Already Implemented):
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/nodes` | List creator's nodes |
| POST | `/api/v1/nodes` | Create new node |
| GET | `/api/v1/nodes/:id` | Get node details |
| PATCH | `/api/v1/nodes/:id` | Update node |
| DELETE | `/api/v1/nodes/:id` | Delete node |
| POST | `/api/v1/nodes/:id/maintenance` | Enter maintenance mode |
| DELETE | `/api/v1/nodes/:id/maintenance` | Exit maintenance mode |
| GET | `/api/v1/ssh_keys` | List creator's SSH keys |
| POST | `/api/v1/ssh_keys` | Create SSH key (upload private key) |
| GET | `/api/v1/ssh_keys/:id` | Get SSH key (fingerprint only) |
| DELETE | `/api/v1/ssh_keys/:id` | Delete SSH key |

### Verification Commands:
```bash
# Backend
make test      # All backend tests pass
make build     # Build with embedded minion binaries
HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=dev make run   # Start backend on :9090

# Backend with remote nodes enabled
HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=dev \
HOSTER_NODES_ENABLED=true \
HOSTER_NODES_ENCRYPTION_KEY=<32-byte-secret-key-here> \
make run

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

These are critical for frontend work:

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
specs/domain/node.md        - Node entity for worker nodes
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
- [ ] How to add new CRUD resources? (Use createResourceApi + createResourceHooks factories)

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
/Users/artpar/.claude/plans/wondrous-forging-donut.md
```

**Implementation Phases:**
- Phase -1: ADR & Spec Updates (COMPLETE)
- Phase 0: API Layer Migration (JSON:API + OpenAPI) (COMPLETE)
- Phase 1: APIGate Integration (Backend Auth) (COMPLETE)
- Phase 2: Billing Integration (COMPLETE)
- Phase 3: Monitoring Backend (COMPLETE)
- Phase 4: Frontend Foundation (COMPLETE)
- Phase 5: Frontend Views (COMPLETE)
- Phase 6: Integration & Polish (COMPLETE)

**Creator Worker Nodes Phases:**
- Phase 1: Domain Model & Scheduler (COMPLETE)
- Phase 2: Database Layer (COMPLETE)
- Phase 3: SSH Docker Client via Minion (COMPLETE)
- Phase 4: Scheduler Integration (COMPLETE)
- Phase 5: Node API Resource (COMPLETE)
- Phase 6: Frontend Nodes Tab (COMPLETE)
- Phase 7: Health Checker Worker <- NEXT

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
- [ ] For new CRUD resources: use createResourceApi + createResourceHooks factories

---

## Phase 4: Making Changes (STC Flow)

### For New Features

```
1. SPEC   -> Create specs/features/F###-name.md
2. TEST   -> Create internal/core/xxx/feature_test.go (failing tests)
3. CODE   -> Create internal/core/xxx/feature.go (make tests pass)
4. VERIFY -> make test
```

### For New CRUD Resources (Frontend)

```
1. Add types to web/src/api/types.ts
2. Create API client using createResourceApi factory
3. Create hooks using createResourceHooks factory
4. Create UI components
5. VERIFY -> npm run build
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
- [ ] Frontend builds (`cd web && npm run build`)

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
- `golang.org/x/crypto/ssh` - SSH client for remote nodes

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
│   │   ├── client.ts              # JSON:API fetch wrapper
│   │   ├── types.ts               # TypeScript types
│   │   ├── createResourceApi.ts   # Generic API factory ← NEW
│   │   ├── templates.ts           # Template API (uses factory)
│   │   ├── deployments.ts         # Deployment API
│   │   ├── monitoring.ts          # Monitoring API
│   │   ├── nodes.ts               # Node API (uses factory) ← NEW
│   │   └── ssh-keys.ts            # SSH Key API (uses factory) ← NEW
│   ├── components/
│   │   ├── common/
│   │   │   ├── LoadingSpinner.tsx
│   │   │   ├── EmptyState.tsx
│   │   │   └── StatusBadge.tsx    # Includes node statuses
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
│   │   ├── nodes/                 # ← NEW
│   │   │   ├── NodeCard.tsx
│   │   │   ├── AddNodeDialog.tsx
│   │   │   ├── AddSSHKeyDialog.tsx
│   │   │   └── index.ts
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
│   │   ├── createResourceHooks.ts # Generic hooks factory ← NEW
│   │   ├── useTemplates.ts        # (uses factory)
│   │   ├── useDeployments.ts
│   │   ├── useMonitoring.ts
│   │   ├── useNodes.ts            # ← NEW
│   │   └── useSSHKeys.ts          # ← NEW
│   ├── pages/
│   │   ├── marketplace/
│   │   │   ├── MarketplacePage.tsx
│   │   │   └── TemplateDetailPage.tsx
│   │   ├── deployments/
│   │   │   ├── MyDeploymentsPage.tsx
│   │   │   └── DeploymentDetailPage.tsx
│   │   ├── creator/
│   │   │   └── CreatorDashboardPage.tsx  # Has Nodes tab
│   │   └── NotFoundPage.tsx
│   ├── stores/
│   │   └── authStore.ts           # Zustand store
│   ├── lib/
│   │   └── cn.ts                  # clsx + tailwind-merge
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

## Adding New CRUD Resources (Quick Reference)

When adding a new resource type, use the generic factories:

**1. Add types to `web/src/api/types.ts`:**
```typescript
export interface FooAttributes { name: string; /* ... */ }
export type Foo = JsonApiResource<'foos', FooAttributes>;
export interface CreateFooRequest { name: string; /* ... */ }
export interface UpdateFooRequest { name?: string; /* ... */ }
```

**2. Create API client `web/src/api/foos.ts`:**
```typescript
import { createResourceApi } from './createResourceApi';
import type { Foo, CreateFooRequest, UpdateFooRequest } from './types';

export const foosApi = createResourceApi<Foo, CreateFooRequest, UpdateFooRequest>({
  resourceName: 'foos',
  // Optional: customActions, supportsUpdate, supportsDelete
});
```

**3. Create hooks `web/src/hooks/useFoos.ts`:**
```typescript
import { foosApi } from '@/api/foos';
import type { Foo, CreateFooRequest, UpdateFooRequest } from '@/api/types';
import { createResourceHooks } from './createResourceHooks';

const fooHooks = createResourceHooks<Foo, CreateFooRequest, UpdateFooRequest>({
  resourceName: 'foos',
  api: foosApi,
});

export const fooKeys = fooHooks.keys;
export const useFoos = fooHooks.useList;
export const useFoo = fooHooks.useGet;
export const useCreateFoo = fooHooks.useCreate;
export const useUpdateFoo = fooHooks.useUpdate;
export const useDeleteFoo = fooHooks.useDelete;
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
| Not using factories for new CRUD | Code duplication | Use createResourceApi + createResourceHooks |

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
| API factories | `web/src/api/` | `createResourceApi.ts` |
| Hook factories | `web/src/hooks/` | `createResourceHooks.ts` |

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
