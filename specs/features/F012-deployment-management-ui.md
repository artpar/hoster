# F012: Deployment Management UI

## Overview

Authenticated user interface for managing personal deployments: viewing status, starting/stopping, viewing logs, and monitoring health.

## User Stories

### US-1: As a customer, I want to see all my deployments

**Acceptance Criteria:**
- Deployments page lists all my deployments
- Shows deployment name, template name, status, created date
- Status is visually distinct (color-coded badges)
- Empty state when no deployments exist

### US-2: As a customer, I want to view deployment details

**Acceptance Criteria:**
- Clicking deployment opens detail view
- Shows deployment configuration and variables
- Shows all containers with their status
- Shows auto-generated domain/URL

### US-3: As a customer, I want to control my deployments

**Acceptance Criteria:**
- Can start a stopped deployment
- Can stop a running deployment
- Can restart a running deployment
- Can delete a deployment (with confirmation)
- Controls disabled during state transitions

### US-4: As a customer, I want to monitor my deployment

**Acceptance Criteria:**
- See real-time health status
- View container logs (polling, not streaming)
- See resource usage stats (CPU, memory)
- See recent events (starts, stops, errors)

## Pages

### MyDeploymentsPage (`/deployments`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Header (Logo, Nav, User Menu)                                  │
├─────────────────────────────────────────────────────────────────┤
│  Sidebar    │                                                    │
│             │  My Deployments                                    │
│  Dashboard  │                                                    │
│  ● Deploy.  │  ┌─────────────────────────────────────────────┐  │
│  Templates  │  │ ● my-wordpress    WordPress    Running      │  │
│             │  │   dep_abc123      Jan 15       [View]       │  │
│             │  └─────────────────────────────────────────────┘  │
│             │                                                    │
│             │  ┌─────────────────────────────────────────────┐  │
│             │  │ ○ my-postgres     Postgres     Stopped      │  │
│             │  │   dep_def456      Jan 10       [View]       │  │
│             │  └─────────────────────────────────────────────┘  │
│             │                                                    │
│             │  ┌─────────────────────────────────────────────┐  │
│             │  │ ◐ staging-app     NextJS       Starting     │  │
│             │  │   dep_ghi789      Jan 18       [View]       │  │
│             │  └─────────────────────────────────────────────┘  │
│             │                                                    │
│             │  No more deployments.                              │
│             │  [Browse Marketplace]                              │
│             │                                                    │
└─────────────┴────────────────────────────────────────────────────┘
```

### DeploymentDetailPage (`/deployments/:id`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Header                                                         │
├─────────────────────────────────────────────────────────────────┤
│  Sidebar    │  ← Back to Deployments                            │
│             │                                                    │
│             │  my-wordpress                   ● Running          │
│             │  WordPress v1.0.0                                  │
│             │  https://my-wordpress.hoster.local                 │
│             │                                                    │
│             │  [Stop]  [Restart]  [Delete]                       │
│             │                                                    │
│             │  ┌──────────────────────────────────────────────┐ │
│             │  │ [Health] [Logs] [Stats] [Events]            │ │
│             │  ├──────────────────────────────────────────────┤ │
│             │  │                                               │ │
│             │  │  Health Status: Healthy ✓                     │ │
│             │  │                                               │ │
│             │  │  Containers:                                  │ │
│             │  │  ┌────────────────────────────────────────┐  │ │
│             │  │  │ wordpress   Running  ● Healthy         │  │ │
│             │  │  │ Started: 2h ago   Restarts: 0          │  │ │
│             │  │  └────────────────────────────────────────┘  │ │
│             │  │  ┌────────────────────────────────────────┐  │ │
│             │  │  │ mysql       Running  ● Healthy         │  │ │
│             │  │  │ Started: 2h ago   Restarts: 0          │  │ │
│             │  │  └────────────────────────────────────────┘  │ │
│             │  │                                               │ │
│             │  └──────────────────────────────────────────────┘ │
│             │                                                    │
└─────────────┴────────────────────────────────────────────────────┘
```

### Logs Tab

```
┌──────────────────────────────────────────────────────────────────┐
│  Container: [All ▼]    Lines: [100 ▼]    [Refresh]              │
├──────────────────────────────────────────────────────────────────┤
│  12:00:01 [wordpress] Apache/2.4.54 configured -- resuming...   │
│  12:00:01 [wordpress] [core:notice] AH00094: Command line: ...  │
│  12:00:00 [mysql] ready for connections. Version: 8.0.32        │
│  11:59:59 [mysql] Starting MySQL 8.0.32-0ubuntu0.22.04.2        │
│  ...                                                             │
│  [Load More]                                                     │
└──────────────────────────────────────────────────────────────────┘
```

### Stats Tab

```
┌──────────────────────────────────────────────────────────────────┐
│  Last updated: 12:00:00    [Refresh]                            │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  wordpress                                                       │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ CPU: 2.5%   Memory: 128MB / 512MB (25%)                     ││
│  │ Network: ↓ 1.2MB  ↑ 0.5MB   PIDs: 12                        ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  mysql                                                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ CPU: 5.1%   Memory: 256MB / 1GB (25%)                       ││
│  │ Network: ↓ 2.1MB  ↑ 1.0MB   PIDs: 28                        ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Events Tab

```
┌──────────────────────────────────────────────────────────────────┐
│  Filter: [All Events ▼]    [Refresh]                            │
├──────────────────────────────────────────────────────────────────┤
│  ● 12:00:00  Container started    wordpress    Container started │
│  ● 11:59:55  Container started    mysql        Container started │
│  ○ 11:59:50  Container created    wordpress    Container created │
│  ○ 11:59:50  Container created    mysql        Container created │
│  ...                                                             │
└──────────────────────────────────────────────────────────────────┘
```

## Components

### DeploymentCard

```typescript
interface DeploymentCardProps {
  deployment: Deployment;
  onClick: () => void;
}
```

Displays:
- Deployment name
- Template name
- Status with color-coded badge
- Created date (relative: "2 hours ago")
- Click to navigate to detail

### DeploymentStatus

```typescript
interface DeploymentStatusProps {
  status: DeploymentStatus;
  size?: 'sm' | 'md' | 'lg';
}
```

Status badges:
- `pending` - Gray
- `scheduled` - Blue
- `starting` - Yellow (animated)
- `running` - Green
- `stopping` - Yellow (animated)
- `stopped` - Gray
- `failed` - Red
- `deleting` - Red (animated)

### DeploymentControls

```typescript
interface DeploymentControlsProps {
  deployment: Deployment;
  onStart: () => void;
  onStop: () => void;
  onRestart: () => void;
  onDelete: () => void;
  isLoading: boolean;
}
```

Button states:
- Start: enabled when stopped
- Stop: enabled when running
- Restart: enabled when running
- Delete: always enabled (with confirmation)
- All disabled during transitions

### ContainerList

```typescript
interface ContainerListProps {
  containers: ContainerHealth[];
}
```

### LogViewer

```typescript
interface LogViewerProps {
  deploymentId: string;
  container?: string;
  tail?: number;
}
```

Features:
- Auto-scroll to bottom
- Container filter dropdown
- Tail size selector
- Manual refresh button
- Polling interval (5s when tab active)

### ResourceStats

```typescript
interface ResourceStatsProps {
  stats: ContainerStats[];
  lastUpdated: Date;
}
```

Displays:
- CPU percentage with progress bar
- Memory usage with progress bar
- Network I/O
- Process count

### EventsList

```typescript
interface EventsListProps {
  events: ContainerEvent[];
  onRefresh: () => void;
}
```

## API Integration

### Hooks

```typescript
// src/hooks/useDeployments.ts

export function useMyDeployments() {
  return useQuery({
    queryKey: ['deployments'],
    queryFn: deploymentsApi.listMine,
  });
}

export function useDeployment(id: string) {
  return useQuery({
    queryKey: ['deployments', id],
    queryFn: () => deploymentsApi.get(id),
    enabled: !!id,
  });
}

export function useStartDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deploymentsApi.start,
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['deployments', id] });
    },
  });
}

// Similar for stop, restart, delete

// src/hooks/useMonitoring.ts

export function useDeploymentHealth(id: string) {
  return useQuery({
    queryKey: ['deployments', id, 'health'],
    queryFn: () => monitoringApi.getHealth(id),
    refetchInterval: 10000, // Poll every 10s
  });
}

export function useDeploymentLogs(id: string, options?: LogOptions) {
  return useQuery({
    queryKey: ['deployments', id, 'logs', options],
    queryFn: () => monitoringApi.getLogs(id, options),
    refetchInterval: 5000, // Poll every 5s
  });
}

export function useDeploymentStats(id: string) {
  return useQuery({
    queryKey: ['deployments', id, 'stats'],
    queryFn: () => monitoringApi.getStats(id),
    refetchInterval: 5000,
  });
}

export function useDeploymentEvents(id: string) {
  return useQuery({
    queryKey: ['deployments', id, 'events'],
    queryFn: () => monitoringApi.getEvents(id),
    refetchInterval: 10000,
  });
}
```

## State Management

### Server State (TanStack Query)
- Deployment list with 30s stale time
- Deployment detail with polling
- Health/logs/stats with short polling intervals

### Local State
- Active tab selection
- Container filter
- Delete confirmation modal

## Routing

| Path | Component | Auth |
|------|-----------|------|
| `/deployments` | MyDeploymentsPage | Yes |
| `/deployments/:id` | DeploymentDetailPage | Yes |

## Files to Create

```
web/src/pages/deployments/
├── MyDeploymentsPage.tsx
├── DeploymentDetailPage.tsx
└── index.ts

web/src/components/deployments/
├── DeploymentCard.tsx
├── DeploymentStatus.tsx
├── DeploymentControls.tsx
├── ContainerList.tsx
└── index.ts

web/src/components/monitoring/
├── HealthStatus.tsx
├── LogViewer.tsx
├── ResourceStats.tsx
├── EventsList.tsx
└── index.ts

web/src/hooks/
├── useDeployments.ts
└── useMonitoring.ts
```

## Test Cases

### Unit Tests (Vitest)

```typescript
// DeploymentStatus.test.tsx
describe('DeploymentStatus', () => {
  it('shows green badge for running', () => {});
  it('shows red badge for failed', () => {});
  it('shows animated badge for transitions', () => {});
});

// DeploymentControls.test.tsx
describe('DeploymentControls', () => {
  it('enables start when stopped', () => {});
  it('disables start when running', () => {});
  it('shows confirmation before delete', () => {});
  it('disables all during loading', () => {});
});

// LogViewer.test.tsx
describe('LogViewer', () => {
  it('displays log lines', () => {});
  it('filters by container', () => {});
  it('auto-scrolls on new logs', () => {});
});
```

### E2E Tests (Playwright)

```typescript
// deployments.spec.ts
test('list my deployments', async ({ page }) => {
  await loginAsUser(page);
  await page.goto('/deployments');
  await expect(page.getByTestId('deployment-card')).toHaveCount(greaterThan(0));
});

test('view deployment details', async ({ page }) => {
  await loginAsUser(page);
  await page.goto('/deployments');
  await page.getByTestId('deployment-card').first().click();
  await expect(page.getByRole('heading')).toBeVisible();
});

test('stop running deployment', async ({ page }) => {
  await loginAsUser(page);
  await page.goto('/deployments/dep_running');
  await page.getByRole('button', { name: 'Stop' }).click();
  await expect(page.getByText('Stopping')).toBeVisible();
});

test('view deployment logs', async ({ page }) => {
  await loginAsUser(page);
  await page.goto('/deployments/dep_123');
  await page.getByRole('tab', { name: 'Logs' }).click();
  await expect(page.getByTestId('log-line')).toHaveCount(greaterThan(0));
});
```

## NOT Supported

- Real-time log streaming (WebSocket)
- Historical metrics graphs
- Alerting/notifications
- Deployment scaling
- Rolling updates
- Bulk operations
- Deployment sharing
- Custom domains (auto-generated only)

## Dependencies

- ADR-006: Frontend Architecture
- F008: Authentication Integration
- F010: Monitoring Dashboard (backend endpoints)
