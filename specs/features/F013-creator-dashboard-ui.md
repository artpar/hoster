# F013: Creator Dashboard UI

## Overview

Authenticated interface for template creators to manage their templates: create, edit, publish, and view deployment statistics.

## User Stories

### US-1: As a creator, I want to see all my templates

**Acceptance Criteria:**
- Dashboard shows list of my templates
- Shows template name, version, status (draft/published), deployment count
- Can filter by published/draft status
- Shows creation and last updated dates

### US-2: As a creator, I want to create a new template

**Acceptance Criteria:**
- Click "New Template" opens editor
- Can enter name, description, version
- Can write/paste Docker Compose YAML
- Can define variables with types, defaults, descriptions
- Can set pricing (monthly cost in cents)
- Validation before saving

### US-3: As a creator, I want to edit my templates

**Acceptance Criteria:**
- Click template opens editor with existing data
- Can modify all template fields
- Version must be incremented on publish
- Can save as draft without publishing
- Published templates can still be edited (creates new version)

### US-4: As a creator, I want to publish my templates

**Acceptance Criteria:**
- Publish button makes template visible in marketplace
- Validation required before publish (name, compose, version)
- Confirmation dialog before publishing
- Can unpublish (removes from marketplace)

### US-5: As a creator, I want to see template statistics

**Acceptance Criteria:**
- See total active deployments per template
- See deployment history (created/deleted counts)
- Displayed on dashboard and detail view

## Pages

### CreatorDashboardPage (`/creator`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Header                                                         │
├─────────────────────────────────────────────────────────────────┤
│  Sidebar    │                                                    │
│             │  Creator Dashboard                                 │
│  Dashboard  │                                           [+ New]  │
│  Deployments│                                                    │
│  ● Templates│  My Templates                                      │
│             │  Filter: [All ▼]                                   │
│             │                                                    │
│             │  ┌─────────────────────────────────────────────┐  │
│             │  │ WordPress          v1.0.0   Published       │  │
│             │  │ $5/month           12 active deployments    │  │
│             │  │ Updated: 2 days ago                [Edit]   │  │
│             │  └─────────────────────────────────────────────┘  │
│             │                                                    │
│             │  ┌─────────────────────────────────────────────┐  │
│             │  │ Ghost Blog         v0.9.0   Draft           │  │
│             │  │ $8/month           0 deployments            │  │
│             │  │ Updated: 1 hour ago                [Edit]   │  │
│             │  └─────────────────────────────────────────────┘  │
│             │                                                    │
│             │  ┌─────────────────────────────────────────────┐  │
│             │  │ NextCloud          v2.1.0   Published       │  │
│             │  │ $10/month          5 active deployments     │  │
│             │  │ Updated: 1 week ago               [Edit]    │  │
│             │  └─────────────────────────────────────────────┘  │
│             │                                                    │
└─────────────┴────────────────────────────────────────────────────┘
```

### TemplateEditorPage (`/creator/templates/new` or `/creator/templates/:id/edit`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Header                                                         │
├─────────────────────────────────────────────────────────────────┤
│  ← Back to Dashboard                                            │
│                                                                  │
│  New Template                    [Save Draft]  [Publish]        │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ [Basic Info] [Compose] [Variables] [Pricing]              │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │                                                             │ │
│  │  Name *                                                     │ │
│  │  ┌────────────────────────────────────────────────────┐    │ │
│  │  │ WordPress                                          │    │ │
│  │  └────────────────────────────────────────────────────┘    │ │
│  │                                                             │ │
│  │  Description                                                │ │
│  │  ┌────────────────────────────────────────────────────┐    │ │
│  │  │ WordPress with MySQL database. Perfect for blogs,  │    │ │
│  │  │ business sites, and small e-commerce stores.       │    │ │
│  │  └────────────────────────────────────────────────────┘    │ │
│  │                                                             │ │
│  │  Version *                                                  │ │
│  │  ┌────────────────────────────────────────────────────┐    │ │
│  │  │ 1.0.0                                              │    │ │
│  │  └────────────────────────────────────────────────────┘    │ │
│  │                                                             │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Compose Tab

```
┌────────────────────────────────────────────────────────────────┐
│  Docker Compose *                                               │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ version: "3.8"                                          │   │
│  │ services:                                               │   │
│  │   wordpress:                                            │   │
│  │     image: wordpress:latest                             │   │
│  │     ports:                                              │   │
│  │       - "${PORT}:80"                                    │   │
│  │     environment:                                        │   │
│  │       WORDPRESS_DB_HOST: mysql                          │   │
│  │       WORDPRESS_DB_USER: ${DB_USER}                     │   │
│  │       WORDPRESS_DB_PASSWORD: ${DB_PASSWORD}             │   │
│  │   mysql:                                                │   │
│  │     image: mysql:8.0                                    │   │
│  │     environment:                                        │   │
│  │       MYSQL_ROOT_PASSWORD: ${DB_PASSWORD}               │   │
│  │       MYSQL_DATABASE: wordpress                         │   │
│  │       MYSQL_USER: ${DB_USER}                            │   │
│  │       MYSQL_PASSWORD: ${DB_PASSWORD}                    │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ⓘ Use ${VARIABLE_NAME} syntax for configurable values        │
│                                                                 │
│  Detected Services: wordpress, mysql                           │
│  Detected Variables: PORT, DB_USER, DB_PASSWORD                │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

### Variables Tab

```
┌────────────────────────────────────────────────────────────────┐
│  Template Variables                              [+ Add Variable]│
│                                                                 │
│  Detected from compose: PORT, DB_USER, DB_PASSWORD             │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ PORT                                                    │   │
│  │ Type: [number ▼]  Default: [80    ]  Required: [✓]     │   │
│  │ Description: [HTTP port for WordPress              ]    │   │
│  │                                              [Remove]   │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ DB_USER                                                 │   │
│  │ Type: [string ▼]  Default: [wordpress]  Required: [ ]  │   │
│  │ Description: [MySQL database username              ]    │   │
│  │                                              [Remove]   │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ DB_PASSWORD                                             │   │
│  │ Type: [secret ▼]  Default: [        ]  Required: [✓]   │   │
│  │ Description: [MySQL root and user password         ]    │   │
│  │                                              [Remove]   │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

### Pricing Tab

```
┌────────────────────────────────────────────────────────────────┐
│  Pricing                                                        │
│                                                                 │
│  Monthly Price (USD) *                                          │
│  ┌────────────────────────────────────────────────────────┐   │
│  │ $  5.00                                                 │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ⓘ Price is charged monthly per active deployment.             │
│    You'll receive revenue share based on your creator tier.    │
│                                                                 │
│  Preview:                                                       │
│  • Customer pays: $5.00/month                                   │
│  • Platform fee (20%): $1.00                                    │
│  • Your revenue: $4.00/month per deployment                     │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

## Components

### CreatorTemplateCard

```typescript
interface CreatorTemplateCardProps {
  template: Template;
  deploymentCount: number;
  onEdit: () => void;
}
```

Displays:
- Template name and version
- Published/Draft badge
- Monthly price
- Active deployment count
- Last updated date
- Edit button

### TemplateForm

```typescript
interface TemplateFormProps {
  template?: Template; // undefined for new
  onSave: (data: TemplateFormData, publish: boolean) => void;
  isSaving: boolean;
}

interface TemplateFormData {
  name: string;
  description: string;
  version: string;
  composeSpec: string;
  variables: Variable[];
  priceMonthly: number;
}
```

### ComposeEditor

```typescript
interface ComposeEditorProps {
  value: string;
  onChange: (value: string) => void;
  onVariablesDetected: (variables: string[]) => void;
}
```

Features:
- Monaco editor with YAML syntax highlighting
- Auto-detect variables (${VAR_NAME} pattern)
- YAML validation
- Line numbers
- Error highlighting

### VariableEditor

```typescript
interface VariableEditorProps {
  variables: Variable[];
  detectedVariables: string[];
  onChange: (variables: Variable[]) => void;
}
```

Features:
- Add/remove variables
- Edit type, default, required, description
- Highlight detected but undefined variables
- Highlight defined but unused variables

### PublishDialog

```typescript
interface PublishDialogProps {
  template: Template;
  onPublish: () => void;
  onCancel: () => void;
  isPublishing: boolean;
  validationErrors: string[];
}
```

Validation checks:
- Name is not empty
- Version follows semver
- Compose spec is valid YAML
- All required variables have definitions
- Price is >= 0

## API Integration

### Hooks

```typescript
// src/hooks/useCreator.ts

export function useMyTemplates() {
  return useQuery({
    queryKey: ['templates', 'mine'],
    queryFn: templatesApi.listMine,
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
      queryClient.invalidateQueries({ queryKey: ['templates', 'mine'] });
    },
  });
}

export function useUpdateTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }) => templatesApi.update(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['templates', id] });
      queryClient.invalidateQueries({ queryKey: ['templates', 'mine'] });
    },
  });
}

export function usePublishTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: templatesApi.publish,
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['templates', id] });
      queryClient.invalidateQueries({ queryKey: ['templates', 'mine'] });
    },
  });
}

export function useUnpublishTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: templatesApi.unpublish,
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['templates', id] });
      queryClient.invalidateQueries({ queryKey: ['templates', 'mine'] });
    },
  });
}

export function useTemplateStats(id: string) {
  return useQuery({
    queryKey: ['templates', id, 'stats'],
    queryFn: () => templatesApi.getStats(id),
    enabled: !!id,
  });
}
```

## State Management

### Server State (TanStack Query)
- Template list
- Template detail
- Template stats
- Mutations for CRUD operations

### Local State
- Active tab in editor
- Form values (controlled)
- Unsaved changes warning
- Publish dialog open

### Form State
- Consider React Hook Form for complex form with validation
- Or use controlled components with local state

## Routing

| Path | Component | Auth |
|------|-----------|------|
| `/creator` | CreatorDashboardPage | Yes |
| `/creator/templates/new` | TemplateEditorPage | Yes |
| `/creator/templates/:id/edit` | TemplateEditorPage | Yes |

## Files to Create

```
web/src/pages/creator/
├── CreatorDashboardPage.tsx
├── TemplateEditorPage.tsx
└── index.ts

web/src/components/creator/
├── CreatorTemplateCard.tsx
├── TemplateForm.tsx
├── ComposeEditor.tsx
├── VariableEditor.tsx
├── PricingInput.tsx
├── PublishDialog.tsx
└── index.ts

web/src/hooks/
└── useCreator.ts
```

## Test Cases

### Unit Tests (Vitest)

```typescript
// ComposeEditor.test.tsx
describe('ComposeEditor', () => {
  it('detects variables in compose spec', () => {});
  it('validates YAML syntax', () => {});
  it('highlights errors', () => {});
});

// VariableEditor.test.tsx
describe('VariableEditor', () => {
  it('shows detected variables', () => {});
  it('allows adding new variables', () => {});
  it('warns about unused variables', () => {});
  it('warns about undefined detected variables', () => {});
});

// PublishDialog.test.tsx
describe('PublishDialog', () => {
  it('shows validation errors', () => {});
  it('enables publish when valid', () => {});
  it('disables publish when invalid', () => {});
});
```

### E2E Tests (Playwright)

```typescript
// creator.spec.ts
test('list my templates', async ({ page }) => {
  await loginAsCreator(page);
  await page.goto('/creator');
  await expect(page.getByTestId('creator-template-card')).toHaveCount(greaterThan(0));
});

test('create new template', async ({ page }) => {
  await loginAsCreator(page);
  await page.goto('/creator');
  await page.getByRole('button', { name: 'New' }).click();
  await expect(page).toHaveURL('/creator/templates/new');

  await page.getByLabel('Name').fill('Test Template');
  await page.getByLabel('Version').fill('1.0.0');
  // ... fill compose spec
  await page.getByRole('button', { name: 'Save Draft' }).click();

  await expect(page.getByText('Template saved')).toBeVisible();
});

test('publish template', async ({ page }) => {
  await loginAsCreator(page);
  await page.goto('/creator/templates/tmpl_draft/edit');
  await page.getByRole('button', { name: 'Publish' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();

  await expect(page.getByText('Template published')).toBeVisible();
});

test('edit existing template', async ({ page }) => {
  await loginAsCreator(page);
  await page.goto('/creator');
  await page.getByRole('button', { name: 'Edit' }).first().click();

  await page.getByLabel('Description').fill('Updated description');
  await page.getByRole('button', { name: 'Save Draft' }).click();

  await expect(page.getByText('Template saved')).toBeVisible();
});
```

## NOT Supported

- Template versioning (multiple versions)
- Template analytics (views, conversion)
- Template reviews/ratings
- Template collaboration
- Template import/export
- Template preview/test deploy
- Revenue dashboard
- Payout management
- Template categories/tags management

## Dependencies

- ADR-006: Frontend Architecture
- F008: Authentication Integration
- Monaco Editor package for YAML editing
