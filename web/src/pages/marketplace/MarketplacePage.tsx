import { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Store, Search, Plus } from 'lucide-react';
import { useTemplates } from '@/hooks/useTemplates';
import { useIsAuthenticated } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { EmptyState } from '@/components/common/EmptyState';
import { TemplateCard } from '@/components/templates/TemplateCard';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/cn';
import { pages } from '@/docs/registry';

const pageDocs = pages.templates;

const categoryMeta: Record<string, { label: string; order: number }> = {
  web:         { label: 'Web Apps',    order: 1 },
  monitoring:  { label: 'Monitoring',  order: 2 },
  development: { label: 'Dev Tools',   order: 3 },
  automation:  { label: 'Automation',  order: 4 },
  analytics:   { label: 'Analytics',   order: 5 },
};

function categoryLabel(key: string): string {
  return categoryMeta[key]?.label ?? key.charAt(0).toUpperCase() + key.slice(1);
}

function categoryOrder(key: string): number {
  return categoryMeta[key]?.order ?? 99;
}

type ViewMode = 'browse' | 'mine';

export function MarketplacePage() {
  const isAuthenticated = useIsAuthenticated();
  const [viewMode, setViewMode] = useState<ViewMode>('browse');

  const browseQuery = useTemplates();
  const mineQuery = useTemplates({ scope: 'mine' });

  const activeQuery = viewMode === 'mine' ? mineQuery : browseQuery;
  const { data: templates, isLoading, error } = activeQuery;

  const [searchQuery, setSearchQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState<string | null>(null);

  // For browse mode: published only. For mine mode: all user's templates.
  const baseTemplates = useMemo(() => {
    if (!templates) return [];
    if (viewMode === 'mine') return templates;
    return templates.filter((t) => t.attributes.published === true);
  }, [templates, viewMode]);

  // Unique categories sorted by defined order
  const categories = useMemo(() => {
    const cats = new Set(baseTemplates.map((t) => t.attributes.category).filter(Boolean) as string[]);
    return [...cats].sort((a, b) => categoryOrder(a) - categoryOrder(b));
  }, [baseTemplates]);

  // Apply search + category filter
  const filtered = useMemo(() => {
    let result = baseTemplates;

    if (activeCategory) {
      result = result.filter((t) => t.attributes.category === activeCategory);
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (t) =>
          t.attributes.name.toLowerCase().includes(q) ||
          (t.attributes.description ?? '').toLowerCase().includes(q) ||
          (t.attributes.category ?? '').toLowerCase().includes(q)
      );
    }

    return result;
  }, [baseTemplates, searchQuery, activeCategory]);

  const clearFilters = () => {
    setSearchQuery('');
    setActiveCategory(null);
  };

  if (isLoading) {
    return <LoadingPage />;
  }

  if (error) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-destructive">
        Failed to load templates: {error.message}
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-2 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
          <p className="text-muted-foreground">{pageDocs.subtitle}</p>
        </div>
        {isAuthenticated && (
          <Link
            to="/templates/new"
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            Create Template
          </Link>
        )}
      </div>

      {/* View toggle + Search row */}
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
        {/* View mode toggle */}
        {isAuthenticated && (
          <div className="flex shrink-0 rounded-lg border border-border p-0.5">
            <button
              onClick={() => { setViewMode('browse'); clearFilters(); }}
              className={cn(
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                viewMode === 'browse'
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              Browse All
            </button>
            <button
              onClick={() => { setViewMode('mine'); clearFilters(); }}
              className={cn(
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                viewMode === 'mine'
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              My Templates
            </button>
          </div>
        )}

        {/* Search */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder={viewMode === 'mine' ? 'Search your templates...' : 'Search apps, tools, platforms...'}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Category Pills */}
      {categories.length > 0 && (
        <div className="mb-6 flex flex-wrap items-center gap-2">
          <button
            onClick={() => setActiveCategory(null)}
            className={cn(
              'rounded-full px-3 py-1 text-sm transition-colors',
              !activeCategory
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground'
            )}
          >
            All
          </button>
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setActiveCategory(activeCategory === cat ? null : cat)}
              className={cn(
                'rounded-full px-3 py-1 text-sm transition-colors',
                activeCategory === cat
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              )}
            >
              {categoryLabel(cat)}
            </button>
          ))}
        </div>
      )}

      {/* Results */}
      {filtered.length === 0 ? (
        <EmptyState
          icon={Store}
          title={searchQuery ? 'No matching templates' : pageDocs.emptyState.label}
          description={
            searchQuery
              ? 'Try adjusting your search or filters'
              : pageDocs.emptyState.description
          }
          action={
            (searchQuery || activeCategory)
              ? { label: 'Clear Filters', onClick: clearFilters }
              : viewMode === 'mine'
                ? (
                    <Link
                      to="/templates/new"
                      className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
                    >
                      Create Template
                    </Link>
                  )
                : undefined
          }
        />
      ) : (
        <div>
          <div className="mb-3 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              {filtered.length} template{filtered.length !== 1 ? 's' : ''}
              {searchQuery && <> matching &ldquo;{searchQuery}&rdquo;</>}
            </span>
            {(searchQuery || activeCategory) && (
              <Button variant="ghost" size="sm" onClick={clearFilters}>
                Clear
              </Button>
            )}
          </div>
          <div className="flex flex-col gap-2">
            {filtered.map((template) => (
              <TemplateCard key={template.id} template={template} />
            ))}
          </div>
        </div>
      )}

    </div>
  );
}
