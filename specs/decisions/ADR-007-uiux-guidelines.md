# ADR-007: UI/UX Implementation Guidelines

## Status
Accepted

## Context

With the frontend architecture decided (ADR-006), we need consistent guidelines for UI/UX implementation to ensure:
- **Consistency**: Same patterns used across all pages and components
- **Correctness**: Accessible, responsive, and performant UI
- **Completeness**: All states handled (loading, error, empty, success)

Without explicit guidelines, different components will have inconsistent styling, behavior, and user experience.

## Decision

We adopt the following UI/UX guidelines as the standard for all frontend development.

---

## 1. Design System Foundation

### Component Library: shadcn/ui

We use [shadcn/ui](https://ui.shadcn.com) as our component foundation because:
- Copy-paste components (not npm dependency) - full control
- Built on Radix UI primitives - accessible by default
- TailwindCSS styling - consistent with our stack
- Customizable - can modify to match brand

**Installation pattern**:
```bash
npx shadcn-ui@latest add button
npx shadcn-ui@latest add card
npx shadcn-ui@latest add dialog
# Components copied to src/components/ui/
```

### Required Components

| Component | Usage |
|-----------|-------|
| Button | All clickable actions |
| Card | Template cards, deployment cards |
| Dialog | Confirmations, forms |
| DropdownMenu | User menu, actions menu |
| Form | All forms (with react-hook-form) |
| Input | Text inputs |
| Label | Form labels |
| Select | Dropdowns |
| Tabs | Detail page sections |
| Toast | Notifications |
| Badge | Status indicators |
| Skeleton | Loading states |
| Alert | Error messages, warnings |
| Avatar | User avatars |

---

## 2. Color System

### Semantic Colors

Use semantic color names, not raw values:

| Token | Light Mode | Dark Mode | Usage |
|-------|------------|-----------|-------|
| `background` | white | slate-950 | Page background |
| `foreground` | slate-950 | slate-50 | Primary text |
| `muted` | slate-100 | slate-800 | Secondary backgrounds |
| `muted-foreground` | slate-500 | slate-400 | Secondary text |
| `primary` | blue-600 | blue-500 | Primary actions |
| `primary-foreground` | white | white | Text on primary |
| `destructive` | red-600 | red-500 | Delete, danger |
| `destructive-foreground` | white | white | Text on destructive |
| `border` | slate-200 | slate-800 | Borders |
| `ring` | blue-400 | blue-400 | Focus rings |

### Status Colors

| Status | Color | Tailwind Class |
|--------|-------|----------------|
| Running / Healthy | Green | `text-green-600 bg-green-100` |
| Stopped / Paused | Gray | `text-gray-600 bg-gray-100` |
| Starting / Pending | Yellow | `text-yellow-600 bg-yellow-100` |
| Failed / Error | Red | `text-red-600 bg-red-100` |
| Draft | Blue | `text-blue-600 bg-blue-100` |
| Published | Green | `text-green-600 bg-green-100` |

### Implementation

```typescript
// tailwind.config.ts
export default {
  theme: {
    extend: {
      colors: {
        border: "hsl(var(--border))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        // ... etc
      },
    },
  },
}
```

---

## 3. Typography

### Font Stack

```css
font-family: Inter, system-ui, -apple-system, sans-serif;
font-family: 'JetBrains Mono', monospace; /* for code */
```

### Type Scale

| Name | Size | Weight | Usage |
|------|------|--------|-------|
| `text-xs` | 12px | 400 | Labels, captions |
| `text-sm` | 14px | 400 | Body text, form inputs |
| `text-base` | 16px | 400 | Default body |
| `text-lg` | 18px | 500 | Card titles |
| `text-xl` | 20px | 600 | Section headings |
| `text-2xl` | 24px | 600 | Page titles |
| `text-3xl` | 30px | 700 | Hero headings |

### Guidelines

- **Body text**: `text-sm` or `text-base` with `text-foreground`
- **Secondary text**: `text-sm` with `text-muted-foreground`
- **Headings**: Use semantic HTML (`h1`, `h2`, `h3`) with appropriate size
- **Monospace**: Use for code, IDs, technical values

```tsx
// Good
<h1 className="text-2xl font-semibold">Page Title</h1>
<p className="text-sm text-muted-foreground">Description</p>

// Bad - raw colors
<h1 className="text-2xl text-gray-900">Page Title</h1>
```

---

## 4. Spacing System

### Base Unit: 4px

Use Tailwind's spacing scale (multiples of 4px):

| Class | Value | Usage |
|-------|-------|-------|
| `p-1` | 4px | Tight padding (badges) |
| `p-2` | 8px | Small padding (buttons) |
| `p-4` | 16px | Standard padding (cards) |
| `p-6` | 24px | Large padding (sections) |
| `p-8` | 32px | Page padding |
| `gap-2` | 8px | Tight spacing |
| `gap-4` | 16px | Standard spacing |
| `gap-6` | 24px | Section spacing |

### Layout Spacing

| Context | Spacing |
|---------|---------|
| Between form fields | `space-y-4` |
| Between cards in grid | `gap-4` or `gap-6` |
| Page sections | `space-y-8` |
| Page padding | `p-6` or `p-8` |
| Card padding | `p-4` or `p-6` |

---

## 5. Layout Patterns

### Page Layout

```tsx
// Standard page layout
<div className="flex min-h-screen">
  <Sidebar className="w-64 border-r" />
  <main className="flex-1">
    <Header className="h-16 border-b" />
    <div className="p-6">
      <PageContent />
    </div>
  </main>
</div>
```

### Grid Layouts

```tsx
// Template/Deployment card grid
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
  {items.map(item => <Card key={item.id} />)}
</div>

// Two-column detail layout
<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
  <div className="lg:col-span-2">
    <MainContent />
  </div>
  <div>
    <Sidebar />
  </div>
</div>
```

### Container Widths

| Context | Max Width |
|---------|-----------|
| Full page content | `max-w-7xl` (1280px) |
| Form dialogs | `max-w-md` (448px) |
| Wide dialogs | `max-w-2xl` (672px) |
| Narrow content | `max-w-prose` (65ch) |

---

## 6. Component Patterns

### Buttons

```tsx
// Primary action
<Button>Deploy Now</Button>

// Secondary action
<Button variant="outline">Cancel</Button>

// Destructive action
<Button variant="destructive">Delete</Button>

// Icon button
<Button variant="ghost" size="icon">
  <MoreHorizontal className="h-4 w-4" />
</Button>

// Loading state
<Button disabled>
  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
  Deploying...
</Button>
```

### Cards

```tsx
<Card>
  <CardHeader>
    <CardTitle>WordPress</CardTitle>
    <CardDescription>WordPress with MySQL</CardDescription>
  </CardHeader>
  <CardContent>
    {/* Content */}
  </CardContent>
  <CardFooter className="flex justify-between">
    <span className="text-lg font-semibold">$5/month</span>
    <Button>Deploy</Button>
  </CardFooter>
</Card>
```

### Status Badges

```tsx
// Component
function StatusBadge({ status }: { status: DeploymentStatus }) {
  const config = {
    running: { label: 'Running', className: 'bg-green-100 text-green-700' },
    stopped: { label: 'Stopped', className: 'bg-gray-100 text-gray-700' },
    starting: { label: 'Starting', className: 'bg-yellow-100 text-yellow-700' },
    failed: { label: 'Failed', className: 'bg-red-100 text-red-700' },
  }[status];

  return (
    <Badge className={config.className}>
      {config.label}
    </Badge>
  );
}
```

### Forms

```tsx
// Use react-hook-form + zod for all forms
const schema = z.object({
  name: z.string().min(3).max(100),
  description: z.string().optional(),
});

function TemplateForm() {
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit">Save</Button>
      </form>
    </Form>
  );
}
```

---

## 7. State Handling

### Loading States

**Always show loading indicators** for async operations:

```tsx
// Skeleton loading for cards
function TemplateCardSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-4 w-48" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-20 w-full" />
      </CardContent>
    </Card>
  );
}

// Page loading
function TemplatesPage() {
  const { data, isLoading, error } = useTemplates();

  if (isLoading) {
    return (
      <div className="grid grid-cols-3 gap-6">
        {[...Array(6)].map((_, i) => <TemplateCardSkeleton key={i} />)}
      </div>
    );
  }

  // ...
}

// Button loading
<Button disabled={isSubmitting}>
  {isSubmitting ? (
    <>
      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
      Saving...
    </>
  ) : (
    'Save'
  )}
</Button>
```

### Error States

**Always handle errors gracefully**:

```tsx
// Page-level error
function TemplatesPage() {
  const { data, error } = useTemplates();

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>
          Failed to load templates. Please try again.
          <Button variant="outline" size="sm" onClick={refetch} className="ml-4">
            Retry
          </Button>
        </AlertDescription>
      </Alert>
    );
  }
}

// Inline error (form)
<FormMessage className="text-sm text-destructive">
  {error.message}
</FormMessage>

// Toast notification for action errors
toast({
  variant: "destructive",
  title: "Failed to deploy",
  description: error.message,
});
```

### Empty States

**Always show helpful empty states**:

```tsx
function EmptyState({
  icon: Icon,
  title,
  description,
  action,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="rounded-full bg-muted p-3 mb-4">
        <Icon className="h-6 w-6 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-medium">{title}</h3>
      <p className="text-sm text-muted-foreground mt-1 max-w-sm">
        {description}
      </p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

// Usage
<EmptyState
  icon={Package}
  title="No deployments yet"
  description="Deploy your first template to get started."
  action={<Button asChild><Link to="/marketplace">Browse Marketplace</Link></Button>}
/>
```

### Success States

```tsx
// Toast for success
toast({
  title: "Deployment created",
  description: "Your deployment is starting up.",
});

// Inline success
<Alert>
  <CheckCircle className="h-4 w-4 text-green-600" />
  <AlertTitle>Success</AlertTitle>
  <AlertDescription>Template published successfully.</AlertDescription>
</Alert>
```

---

## 8. Accessibility Requirements

### WCAG 2.1 AA Compliance

| Requirement | Implementation |
|-------------|----------------|
| Color contrast | Min 4.5:1 for text, 3:1 for large text |
| Focus indicators | Visible focus rings on all interactive elements |
| Keyboard navigation | All actions accessible via keyboard |
| Screen reader support | Proper ARIA labels, semantic HTML |
| Motion preferences | Respect `prefers-reduced-motion` |

### Implementation

```tsx
// Focus rings (built into shadcn)
<Button className="focus-visible:ring-2 focus-visible:ring-ring">

// ARIA labels
<Button aria-label="Close dialog">
  <X className="h-4 w-4" />
</Button>

// Screen reader only text
<span className="sr-only">Loading</span>

// Skip link
<a href="#main-content" className="sr-only focus:not-sr-only">
  Skip to main content
</a>

// Reduced motion
<div className="animate-spin motion-reduce:animate-none">
```

### Required ARIA Patterns

| Component | ARIA |
|-----------|------|
| Dialog | `role="dialog"`, `aria-modal="true"`, `aria-labelledby` |
| Tabs | `role="tablist"`, `role="tab"`, `aria-selected` |
| Status badges | `role="status"`, `aria-live="polite"` |
| Loading | `aria-busy="true"`, `aria-live="polite"` |
| Errors | `role="alert"`, `aria-live="assertive"` |

---

## 9. Responsive Design

### Breakpoints

| Name | Min Width | Usage |
|------|-----------|-------|
| `sm` | 640px | Mobile landscape |
| `md` | 768px | Tablet |
| `lg` | 1024px | Desktop |
| `xl` | 1280px | Large desktop |
| `2xl` | 1536px | Extra large |

### Mobile-First Approach

```tsx
// Write mobile-first, add responsive overrides
<div className="
  grid grid-cols-1       // Mobile: 1 column
  md:grid-cols-2         // Tablet: 2 columns
  lg:grid-cols-3         // Desktop: 3 columns
  gap-4
">

// Hide/show based on breakpoint
<div className="hidden md:block">Desktop only</div>
<div className="md:hidden">Mobile only</div>
```

### Responsive Patterns

| Component | Mobile | Desktop |
|-----------|--------|---------|
| Sidebar | Hidden (hamburger menu) | Visible |
| Card grid | 1 column | 2-3 columns |
| Form layout | Stacked | Side-by-side labels |
| Tables | Card view / horizontal scroll | Full table |
| Dialog | Full screen | Centered modal |

---

## 10. Animation Guidelines

### Principles

1. **Purposeful**: Animations communicate state changes
2. **Subtle**: Don't distract from content
3. **Fast**: Keep under 300ms for UI feedback
4. **Respectful**: Honor `prefers-reduced-motion`

### Standard Durations

| Duration | Usage |
|----------|-------|
| 150ms | Hover states, button feedback |
| 200ms | Dropdowns, tooltips |
| 300ms | Modals, page transitions |
| 500ms | Complex animations (use sparingly) |

### Common Animations

```tsx
// Fade in (for appearing elements)
<div className="animate-in fade-in duration-200">

// Slide in (for modals, drawers)
<div className="animate-in slide-in-from-bottom-4 duration-300">

// Spin (for loading)
<Loader2 className="animate-spin" />

// Pulse (for skeleton loading)
<Skeleton className="animate-pulse" />
```

### Transition Utilities

```tsx
// Interactive elements
<Button className="transition-colors hover:bg-primary/90">

// Cards on hover
<Card className="transition-shadow hover:shadow-lg">
```

---

## 11. Dark Mode

### Implementation

```tsx
// ThemeProvider wraps app
<ThemeProvider defaultTheme="system" storageKey="hoster-theme">
  <App />
</ThemeProvider>

// Toggle component
function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  return (
    <Button variant="ghost" size="icon" onClick={() =>
      setTheme(theme === 'dark' ? 'light' : 'dark')
    }>
      <Sun className="h-4 w-4 dark:hidden" />
      <Moon className="h-4 w-4 hidden dark:block" />
    </Button>
  );
}
```

### Color Guidelines

- Use semantic tokens (`bg-background`, not `bg-white`)
- Test all components in both modes
- Ensure sufficient contrast in both modes
- Images/illustrations should work in both modes

---

## 12. Icons

### Library: Lucide React

```bash
npm install lucide-react
```

### Usage

```tsx
import { Plus, Trash2, Settings, ChevronRight } from 'lucide-react';

// Standard size
<Plus className="h-4 w-4" />

// In button
<Button>
  <Plus className="mr-2 h-4 w-4" />
  Add Template
</Button>

// Icon button
<Button variant="ghost" size="icon">
  <Settings className="h-4 w-4" />
</Button>
```

### Icon Sizes

| Context | Size | Class |
|---------|------|-------|
| Inline with text | 16px | `h-4 w-4` |
| Button icons | 16px | `h-4 w-4` |
| Card icons | 20px | `h-5 w-5` |
| Empty state | 24px | `h-6 w-6` |
| Hero icons | 48px | `h-12 w-12` |

---

## Consequences

### Positive
- **Consistency**: Same patterns everywhere
- **Speed**: Don't reinvent components
- **Quality**: Accessibility built-in
- **Maintainability**: Easy to update globally

### Negative
- **Learning curve**: Team must learn guidelines
- **Rigidity**: Less creative freedom
- **Documentation**: Must keep guidelines updated

### Neutral
- Guidelines can evolve as needs change
- shadcn components can be customized if needed

## Files to Create

- `web/src/components/ui/` - shadcn components
- `web/src/components/common/StatusBadge.tsx`
- `web/src/components/common/EmptyState.tsx`
- `web/src/components/common/LoadingSpinner.tsx`
- `web/src/components/common/PageHeader.tsx`
- `web/src/components/layout/Sidebar.tsx`
- `web/src/components/layout/Header.tsx`
- `web/src/lib/utils.ts` - `cn()` utility
- `web/tailwind.config.ts` - Theme configuration

## References

- shadcn/ui: https://ui.shadcn.com
- Radix UI: https://www.radix-ui.com
- Tailwind CSS: https://tailwindcss.com
- Lucide Icons: https://lucide.dev
- WCAG 2.1: https://www.w3.org/WAI/WCAG21/quickref/
