import { useState, useMemo } from 'react';
import { Store, Search } from 'lucide-react';
import { useTemplates } from '@/hooks/useTemplates';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { EmptyState } from '@/components/common/EmptyState';
import { TemplateCard } from '@/components/templates/TemplateCard';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/cn';
import { pages } from '@/docs/registry';

const pageDocs = pages.marketplace;

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

export function MarketplacePage() {
  const { data: templates, isLoading, error } = useTemplates();
  const [searchQuery, setSearchQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState<string | null>(null);

  // Published templates only
  const published = useMemo(() => {
    if (!templates) return [];
    return templates.filter((t) => t.attributes.published === true);
  }, [templates]);

  // Unique categories sorted by defined order
  const categories = useMemo(() => {
    const cats = new Set(published.map((t) => t.attributes.category).filter(Boolean) as string[]);
    return [...cats].sort((a, b) => categoryOrder(a) - categoryOrder(b));
  }, [published]);

  // Apply search + category filter
  const filtered = useMemo(() => {
    let result = published;

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
  }, [published, searchQuery, activeCategory]);

  // Group by category for the browse view (no search, no category filter)
  const isBrowseView = !searchQuery.trim() && !activeCategory;

  const grouped = useMemo(() => {
    if (!isBrowseView) return [];
    const map = new Map<string, typeof filtered>();
    for (const t of filtered) {
      const cat = t.attributes.category || 'other';
      if (!map.has(cat)) map.set(cat, []);
      map.get(cat)!.push(t);
    }
    return [...map.entries()].sort(([a], [b]) => categoryOrder(a) - categoryOrder(b));
  }, [filtered, isBrowseView]);

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
      <div className="mb-2">
        <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
        <p className="text-muted-foreground">{pageDocs.subtitle}</p>
      </div>

      {/* Search */}
      <div className="mb-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search apps, tools, platforms..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Category Pills */}
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
              : undefined
          }
        />
      ) : isBrowseView ? (
        /* Grouped browse view */
        <div className="space-y-8">
          {grouped.map(([cat, items]) => (
            <section key={cat}>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-lg font-semibold">{categoryLabel(cat)}</h2>
                <span className="text-sm text-muted-foreground">
                  {items.length} template{items.length !== 1 ? 's' : ''}
                </span>
              </div>
              <div className="flex flex-col gap-2">
                {items.map((template) => (
                  <TemplateCard key={template.id} template={template} />
                ))}
              </div>
            </section>
          ))}
        </div>
      ) : (
        /* Flat filtered view */
        <div>
          <div className="mb-3 flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              {filtered.length} result{filtered.length !== 1 ? 's' : ''}
              {searchQuery && <> matching &ldquo;{searchQuery}&rdquo;</>}
            </span>
            <Button variant="ghost" size="sm" onClick={clearFilters}>
              Clear
            </Button>
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
