# F011: Marketplace UI

## Overview

Public-facing marketplace page where visitors can browse and search published templates, view details, and initiate deployment.

## User Stories

### US-1: As a visitor, I want to browse available templates

**Acceptance Criteria:**
- Marketplace page shows grid of published templates
- Each template card shows: name, description snippet, price, creator name
- Templates are sorted by most recently published (default)
- Page loads without authentication

### US-2: As a visitor, I want to search and filter templates

**Acceptance Criteria:**
- Search bar filters templates by name and description
- Can filter by category (if categories exist)
- Can sort by: newest, price (low-high), price (high-low)
- Clear filters button resets to default view

### US-3: As a visitor, I want to view template details

**Acceptance Criteria:**
- Clicking template card opens detail page
- Detail page shows: full description, compose services, variables, price
- Shows "Deploy" button (redirects to login if not authenticated)
- Shows creator name and template version

### US-4: As a customer, I want to deploy a template from marketplace

**Acceptance Criteria:**
- Click "Deploy" on template detail page
- If not authenticated, redirect to APIGate login
- After login, return to deployment flow
- Fill in required variables
- Confirm deployment and see status

## Pages

### MarketplacePage (`/marketplace`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Header (Logo, Nav, Login/User Menu)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  🔍 Search templates...                    [Sort ▼]       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │             │  │             │  │             │             │
│  │  WordPress  │  │   Ghost     │  │  Postgres   │             │
│  │             │  │             │  │             │             │
│  │  $5/month   │  │  $8/month   │  │  $3/month   │             │
│  │  by admin   │  │  by john    │  │  by admin   │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │             │  │             │  │             │             │
│  │  NextCloud  │  │  Gitea      │  │  Mattermost │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                  │
│  [Load More] or pagination                                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### TemplateDetailPage (`/marketplace/:id`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Header                                                         │
├─────────────────────────────────────────────────────────────────┤
│  ← Back to Marketplace                                          │
│                                                                  │
│  ┌──────────────────────────────────────┬────────────────────┐  │
│  │                                      │                    │  │
│  │  WordPress                           │  $5/month          │  │
│  │  Version 1.0.0                       │                    │  │
│  │  by admin                            │  [Deploy Now]      │  │
│  │                                      │                    │  │
│  ├──────────────────────────────────────┴────────────────────┤  │
│  │                                                            │  │
│  │  Description                                               │  │
│  │  WordPress with MySQL database. Perfect for blogs,         │  │
│  │  business sites, and small e-commerce stores.              │  │
│  │                                                            │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │  Services                                                  │  │
│  │  • wordpress (image: wordpress:latest)                     │  │
│  │  • mysql (image: mysql:8.0)                                │  │
│  │                                                            │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │  Configuration Variables                                   │  │
│  │  • MYSQL_ROOT_PASSWORD (required)                          │  │
│  │  • WORDPRESS_DB_USER (default: wordpress)                  │  │
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### DeploymentDialog (Modal on TemplateDetailPage)

```
┌─────────────────────────────────────────────────────────────────┐
│  Deploy WordPress                                          [X]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Deployment Name                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  my-wordpress-blog                                        │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Configuration                                                   │
│                                                                  │
│  MYSQL_ROOT_PASSWORD *                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  ••••••••••••                                             │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  WORDPRESS_DB_USER                                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  wordpress                                                │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ⓘ This will cost $5/month. You can stop or delete anytime.    │
│                                                                  │
│  [Cancel]                                    [Create Deployment] │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### TemplateCard

```typescript
interface TemplateCardProps {
  id: string;
  name: string;
  description: string;
  priceMonthly: number;
  creatorName: string;
  imageUrl?: string;
  onClick: () => void;
}
```

Displays:
- Template name (truncate at 30 chars)
- Description snippet (first 100 chars)
- Price formatted as "$X/month"
- Creator name
- Placeholder image or first letter avatar

### TemplateDetail

```typescript
interface TemplateDetailProps {
  template: Template;
  onDeploy: () => void;
  isAuthenticated: boolean;
}
```

Displays:
- Full template information
- List of services from compose spec
- List of variables with types and defaults
- Deploy button (state depends on auth)

### DeploymentForm

```typescript
interface DeploymentFormProps {
  template: Template;
  onSubmit: (data: DeploymentFormData) => void;
  onCancel: () => void;
  isSubmitting: boolean;
}

interface DeploymentFormData {
  name: string;
  variables: Record<string, string>;
}
```

Form fields:
- Deployment name (auto-generated suggestion)
- Dynamic fields for each template variable
- Required fields marked with asterisk
- Validation before submission

## API Integration

### Hooks

```typescript
// src/hooks/useMarketplace.ts

// List published templates
export function usePublishedTemplates(options?: {
  search?: string;
  sort?: 'newest' | 'price_asc' | 'price_desc';
  page?: number;
  limit?: number;
}) {
  return useQuery({
    queryKey: ['templates', 'published', options],
    queryFn: () => templatesApi.listPublished(options),
  });
}

// Get template by ID
export function useTemplate(id: string) {
  return useQuery({
    queryKey: ['templates', id],
    queryFn: () => templatesApi.get(id),
    enabled: !!id,
  });
}

// Create deployment
export function useCreateDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deploymentsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deployments'] });
    },
  });
}
```

### API Calls

```typescript
// src/api/templates.ts

export const templatesApi = {
  listPublished: async (options?: PublishedOptions) => {
    const params = new URLSearchParams();
    if (options?.search) params.set('filter[search]', options.search);
    if (options?.sort) params.set('sort', options.sort);
    if (options?.page) params.set('page[number]', String(options.page));
    if (options?.limit) params.set('page[size]', String(options.limit));

    return jsonApiClient<Template[]>(`/templates?published=true&${params}`);
  },

  get: async (id: string) => {
    return jsonApiClient<Template>(`/templates/${id}`);
  },
};
```

## State Management

### URL State (React Router)
- Search query: `?search=wordpress`
- Sort order: `?sort=price_asc`
- Page: `?page=2`

### Local State (Component)
- Modal open/close
- Form values
- Form validation errors

### Server State (TanStack Query)
- Template list with caching
- Template detail with caching
- Deployment mutation

## Routing

| Path | Component | Auth |
|------|-----------|------|
| `/marketplace` | MarketplacePage | No |
| `/marketplace/:id` | TemplateDetailPage | No |

## Files to Create

```
web/src/pages/marketplace/
├── MarketplacePage.tsx
├── TemplateDetailPage.tsx
└── index.ts

web/src/components/templates/
├── TemplateCard.tsx
├── TemplateDetail.tsx
├── TemplateGrid.tsx
├── TemplateSearch.tsx
├── DeploymentForm.tsx
└── index.ts

web/src/hooks/
└── useMarketplace.ts
```

## Test Cases

### Unit Tests (Vitest)

```typescript
// TemplateCard.test.tsx
describe('TemplateCard', () => {
  it('displays template name and price', () => {});
  it('truncates long descriptions', () => {});
  it('calls onClick when clicked', () => {});
  it('formats price correctly', () => {});
});

// DeploymentForm.test.tsx
describe('DeploymentForm', () => {
  it('renders required variables as required fields', () => {});
  it('pre-fills default values', () => {});
  it('validates required fields before submit', () => {});
  it('disables submit button while submitting', () => {});
});
```

### E2E Tests (Playwright)

```typescript
// marketplace.spec.ts
test('browse marketplace without login', async ({ page }) => {
  await page.goto('/marketplace');
  await expect(page.getByRole('heading', { name: 'Marketplace' })).toBeVisible();
  await expect(page.getByTestId('template-card')).toHaveCount(greaterThan(0));
});

test('search templates by name', async ({ page }) => {
  await page.goto('/marketplace');
  await page.getByPlaceholder('Search templates').fill('wordpress');
  await expect(page.getByTestId('template-card')).toContainText('WordPress');
});

test('view template details', async ({ page }) => {
  await page.goto('/marketplace');
  await page.getByTestId('template-card').first().click();
  await expect(page.getByRole('heading')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Deploy' })).toBeVisible();
});

test('deploy requires authentication', async ({ page }) => {
  await page.goto('/marketplace/tmpl_123');
  await page.getByRole('button', { name: 'Deploy' }).click();
  // Should redirect to login
  await expect(page).toHaveURL(/login/);
});
```

## NOT Supported

- Template ratings/reviews
- Template categories/tags
- Template recommendations
- Template preview/demo
- Template comparison
- Social sharing
- Wishlists/favorites
- Template comments
- Creator profiles

## Dependencies

- ADR-006: Frontend Architecture (React + Vite stack)
- ADR-003: JSON:API with api2go (API format)
- F008: Authentication Integration (deploy requires auth)
