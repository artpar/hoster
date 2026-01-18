# ADR-006: Frontend Architecture

## Status
Accepted

## Context

Hoster needs a web frontend for:
- **Marketplace**: Browse and search published templates
- **Deployment Management**: View, start, stop, restart, delete deployments
- **Creator Dashboard**: Manage templates, view deployment stats
- **Monitoring Views**: Health status, logs, resource stats

Requirements:
- Type-safe API integration (leveraging OpenAPI spec from ADR-004)
- Modern, responsive UI
- Fast development iteration
- Good developer experience
- Minimal bundle size

## Decision

We will build the frontend using **React + Vite** with the following stack:

| Layer | Technology | Purpose |
|-------|------------|---------|
| Framework | React 18 | UI components |
| Build Tool | Vite | Fast HMR, ES modules |
| Language | TypeScript | Type safety |
| Server State | TanStack Query v5 | API data fetching/caching |
| Client State | Zustand | Local UI state |
| Styling | TailwindCSS | Utility-first CSS |
| Components | shadcn/ui | Accessible, customizable components |
| Routing | React Router v6 | Client-side routing |
| Type Generation | openapi-typescript | TypeScript from OpenAPI |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       React Application                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │    Pages     │  │  Components  │  │    Hooks     │          │
│  │              │  │              │  │              │          │
│  │ Marketplace  │  │ TemplateCard │  │ useTemplates │          │
│  │ Deployments  │  │ DeployCard   │  │ useDeploym.. │          │
│  │ Creator      │  │ LogViewer    │  │ useAuth      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│         │                 │                 │                   │
│         └─────────────────┼─────────────────┘                   │
│                           │                                      │
│  ┌────────────────────────┴────────────────────────────────┐   │
│  │                    TanStack Query                        │   │
│  │         (Server state, caching, refetching)             │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                      │
│  ┌────────────────────────┴────────────────────────────────┐   │
│  │                     API Client                           │   │
│  │    (JSON:API format, typed from OpenAPI schema)         │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                      │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
              ┌──────────────────────────┐
              │    Hoster Backend API    │
              │   (JSON:API + OpenAPI)   │
              └──────────────────────────┘
```

### Project Structure

```
web/
├── package.json
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── index.html
├── public/
│   └── favicon.ico
└── src/
    ├── main.tsx                    # Entry point
    ├── App.tsx                     # Root component + routing
    ├── api/
    │   ├── schema.d.ts             # Generated from OpenAPI
    │   ├── client.ts               # Base fetch client
    │   ├── types.ts                # JSON:API wrapper types
    │   ├── templates.ts            # Template API functions
    │   ├── deployments.ts          # Deployment API functions
    │   └── monitoring.ts           # Monitoring API functions
    ├── hooks/
    │   ├── useTemplates.ts         # TanStack Query hooks
    │   ├── useDeployments.ts
    │   ├── useMonitoring.ts
    │   └── useAuth.ts
    ├── stores/
    │   └── authStore.ts            # Zustand auth state
    ├── pages/
    │   ├── marketplace/
    │   │   ├── MarketplacePage.tsx
    │   │   └── TemplateDetailPage.tsx
    │   ├── deployments/
    │   │   ├── MyDeploymentsPage.tsx
    │   │   └── DeploymentDetailPage.tsx
    │   ├── creator/
    │   │   ├── CreatorDashboardPage.tsx
    │   │   └── TemplateEditorPage.tsx
    │   └── NotFoundPage.tsx
    ├── components/
    │   ├── ui/                     # shadcn/ui components
    │   ├── layout/
    │   │   ├── Header.tsx
    │   │   ├── Sidebar.tsx
    │   │   └── PageContainer.tsx
    │   ├── templates/
    │   │   ├── TemplateCard.tsx
    │   │   ├── TemplateForm.tsx
    │   │   └── ComposeEditor.tsx
    │   ├── deployments/
    │   │   ├── DeploymentCard.tsx
    │   │   ├── DeploymentStatus.tsx
    │   │   ├── DeploymentControls.tsx
    │   │   └── ContainerList.tsx
    │   ├── monitoring/
    │   │   ├── HealthStatus.tsx
    │   │   ├── ResourceStats.tsx
    │   │   └── LogViewer.tsx
    │   └── common/
    │       ├── LoadingSpinner.tsx
    │       ├── EmptyState.tsx
    │       └── StatusBadge.tsx
    └── lib/
        └── cn.ts                   # Tailwind class utilities
```

### Type Generation Flow

```
┌───────────────────────────────────────────────────────────┐
│     Go structs (internal/core/domain/)                    │
└───────────────────────────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│     api2go resources (internal/shell/api/resources/)      │
│     Implement GetID, GetName, GetReferences               │
└───────────────────────────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│     OpenAPI Generator (internal/shell/api/openapi/)       │
│     Reflects on resources → generates OpenAPI 3.0         │
└───────────────────────────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│     /openapi.json endpoint (served at runtime)            │
└───────────────────────────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│     openapi-typescript (build time)                       │
│     npm run generate:types                                │
└───────────────────────────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│     src/api/schema.d.ts                                   │
│     TypeScript types matching backend exactly             │
└───────────────────────────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────┐
│     React components (type-safe API calls)                │
│     import type { components } from './schema'            │
└───────────────────────────────────────────────────────────┘
```

### API Client Pattern

```typescript
// src/api/client.ts
const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

export async function jsonApiClient<T>(
  endpoint: string,
  options?: RequestInit
): Promise<JsonApiResponse<T>> {
  const response = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/vnd.api+json',
      'Accept': 'application/vnd.api+json',
      ...options?.headers,
    },
    credentials: 'include', // For cookies from APIGate
  });

  if (!response.ok) {
    const error = await response.json();
    throw new JsonApiError(error);
  }

  return response.json();
}
```

### TanStack Query Hooks

```typescript
// src/hooks/useTemplates.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { templatesApi } from '../api/templates';
import type { components } from '../api/schema';

type Template = components['schemas']['Template'];

export function useTemplates() {
  return useQuery({
    queryKey: ['templates'],
    queryFn: templatesApi.list,
  });
}

export function useTemplate(id: string) {
  return useQuery({
    queryKey: ['templates', id],
    queryFn: () => templatesApi.get(id),
    enabled: !!id,
  });
}

export function useCreateTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: templatesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
    },
  });
}
```

### Zustand Store Pattern

```typescript
// src/stores/authStore.ts
import { create } from 'zustand';

interface AuthState {
  userId: string | null;
  planId: string | null;
  planLimits: PlanLimits | null;
  isAuthenticated: boolean;
  setAuth: (userId: string, planId: string, limits: PlanLimits) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  userId: null,
  planId: null,
  planLimits: null,
  isAuthenticated: false,
  setAuth: (userId, planId, planLimits) => set({
    userId,
    planId,
    planLimits,
    isAuthenticated: true,
  }),
  clearAuth: () => set({
    userId: null,
    planId: null,
    planLimits: null,
    isAuthenticated: false,
  }),
}));
```

### Route Structure

| Path | Component | Auth Required | Description |
|------|-----------|---------------|-------------|
| `/marketplace` | MarketplacePage | No | Browse published templates |
| `/marketplace/:id` | TemplateDetailPage | No | Template details + deploy |
| `/deployments` | MyDeploymentsPage | Yes | User's deployments |
| `/deployments/:id` | DeploymentDetailPage | Yes | Deployment details + controls |
| `/creator` | CreatorDashboardPage | Yes | Creator's templates |
| `/creator/templates/new` | TemplateEditorPage | Yes | Create new template |
| `/creator/templates/:id/edit` | TemplateEditorPage | Yes | Edit template |

## Implementation

### Build Scripts

```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "generate:types": "openapi-typescript http://localhost:9090/openapi.json -o src/api/schema.d.ts",
    "prebuild": "npm run generate:types",
    "predev": "npm run generate:types",
    "test": "vitest",
    "test:e2e": "playwright test"
  }
}
```

### Vite Configuration

```typescript
// vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:9090',
        changeOrigin: true,
      },
    },
  },
});
```

### Development Workflow

1. Start backend: `make run` (port 9090)
2. Generate types: `npm run generate:types`
3. Start frontend: `npm run dev` (port 3000)
4. Vite proxies `/api` requests to backend

## Consequences

### Positive
- **Type safety**: OpenAPI → TypeScript ensures API changes are caught at compile time
- **Fast development**: Vite HMR provides instant feedback
- **Good UX**: TanStack Query handles loading, caching, refetching
- **Maintainable**: Clear separation of concerns (pages, components, hooks, stores)
- **Accessible**: shadcn/ui provides WCAG-compliant components
- **Modern tooling**: Well-supported ecosystem with good documentation

### Negative
- **Separate build**: Frontend has its own build process, separate from Go
- **Initial setup**: More configuration than a simple SPA
- **Type generation step**: Must run `generate:types` when API changes
- **Additional dependencies**: Many npm packages to manage

### Neutral
- Can serve from Go backend (embed static files) or separate hosting
- Development requires both backend and frontend running
- Types need regeneration when backend OpenAPI spec changes

## Alternatives Considered

### Vue.js
- **Rejected because**: Team more familiar with React ecosystem
- **Would reconsider if**: Performance becomes critical (Vue has smaller bundle)

### Next.js
- **Rejected because**: SSR not needed for admin dashboard, adds complexity
- **Would reconsider if**: SEO becomes important for marketplace

### Remix
- **Rejected because**: Overkill for our use case, learning curve
- **Would reconsider if**: Need server components or streaming

### htmx + Go templates
- **Rejected because**: Less interactive, harder to maintain complex UI
- **Would reconsider if**: Bundle size becomes critical issue

### Redux Toolkit
- **Rejected because**: Overkill for our state needs, Zustand is simpler
- **Would reconsider if**: State becomes very complex

## Dependencies

```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.21.0",
    "@tanstack/react-query": "^5.17.0",
    "zustand": "^4.4.7",
    "tailwind-merge": "^2.2.0",
    "clsx": "^2.1.0",
    "lucide-react": "^0.303.0"
  },
  "devDependencies": {
    "vite": "^5.0.10",
    "@vitejs/plugin-react": "^4.2.1",
    "typescript": "^5.3.3",
    "tailwindcss": "^3.4.0",
    "postcss": "^8.4.33",
    "autoprefixer": "^10.4.16",
    "vitest": "^1.1.0",
    "@playwright/test": "^1.40.1",
    "openapi-typescript": "^6.7.0",
    "@types/react": "^18.2.47",
    "@types/react-dom": "^18.2.18"
  }
}
```

## Files to Create

- `web/package.json` - npm configuration
- `web/vite.config.ts` - Vite configuration
- `web/tailwind.config.ts` - Tailwind configuration
- `web/tsconfig.json` - TypeScript configuration
- `web/index.html` - HTML entry point
- `web/src/main.tsx` - React entry point
- `web/src/App.tsx` - Root component
- `web/src/api/*` - API client and types
- `web/src/hooks/*` - TanStack Query hooks
- `web/src/stores/*` - Zustand stores
- `web/src/pages/*` - Page components
- `web/src/components/*` - Reusable components

## References

- React: https://react.dev
- Vite: https://vitejs.dev
- TanStack Query: https://tanstack.com/query
- Zustand: https://github.com/pmndrs/zustand
- TailwindCSS: https://tailwindcss.com
- shadcn/ui: https://ui.shadcn.com
- openapi-typescript: https://github.com/drwpow/openapi-typescript
