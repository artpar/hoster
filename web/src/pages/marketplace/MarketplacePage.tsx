import { useState, useMemo } from 'react';
import { Store, Search, SlidersHorizontal, ArrowUpDown } from 'lucide-react';
import { useTemplates } from '@/hooks/useTemplates';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { EmptyState } from '@/components/common/EmptyState';
import { TemplateCard } from '@/components/templates/TemplateCard';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { Button } from '@/components/ui/Button';

type SortOption = 'name' | 'price_asc' | 'price_desc' | 'newest';
type PriceFilter = 'all' | 'free' | 'paid';

export function MarketplacePage() {
  const { data: templates, isLoading, error } = useTemplates();

  // Search and filter state
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState<SortOption>('newest');
  const [priceFilter, setPriceFilter] = useState<PriceFilter>('all');

  // Filter and sort templates
  const filteredTemplates = useMemo(() => {
    if (!templates) return [];

    // Start with published templates only
    let result = templates.filter((t) => t.attributes.published === true);

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (t) =>
          t.attributes.name.toLowerCase().includes(query) ||
          (t.attributes.description ?? '').toLowerCase().includes(query)
      );
    }

    // Apply price filter
    if (priceFilter === 'free') {
      result = result.filter((t) => t.attributes.price_monthly_cents === 0);
    } else if (priceFilter === 'paid') {
      result = result.filter((t) => t.attributes.price_monthly_cents > 0);
    }

    // Apply sorting
    switch (sortBy) {
      case 'name':
        result.sort((a, b) => a.attributes.name.localeCompare(b.attributes.name));
        break;
      case 'price_asc':
        result.sort((a, b) => a.attributes.price_monthly_cents - b.attributes.price_monthly_cents);
        break;
      case 'price_desc':
        result.sort((a, b) => b.attributes.price_monthly_cents - a.attributes.price_monthly_cents);
        break;
      case 'newest':
        result.sort(
          (a, b) =>
            new Date(b.attributes.created_at).getTime() -
            new Date(a.attributes.created_at).getTime()
        );
        break;
    }

    return result;
  }, [templates, searchQuery, sortBy, priceFilter]);

  const clearFilters = () => {
    setSearchQuery('');
    setSortBy('newest');
    setPriceFilter('all');
  };

  const hasActiveFilters = searchQuery || sortBy !== 'newest' || priceFilter !== 'all';

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

  const totalPublished = templates?.filter((t) => t.attributes.published === true).length ?? 0;

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Marketplace</h1>
        <p className="text-muted-foreground">
          Browse and deploy from our collection of {totalPublished} templates
        </p>
      </div>

      {/* Search and Filters */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center">
        {/* Search Input */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search templates..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        {/* Sort Dropdown */}
        <div className="flex items-center gap-2">
          <ArrowUpDown className="h-4 w-4 text-muted-foreground" />
          <Select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as SortOption)}
            options={[
              { value: 'newest', label: 'Newest First' },
              { value: 'name', label: 'Name (A-Z)' },
              { value: 'price_asc', label: 'Price: Low to High' },
              { value: 'price_desc', label: 'Price: High to Low' },
            ]}
            className="w-44"
          />
        </div>

        {/* Price Filter */}
        <div className="flex items-center gap-2">
          <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
          <Select
            value={priceFilter}
            onChange={(e) => setPriceFilter(e.target.value as PriceFilter)}
            options={[
              { value: 'all', label: 'All Prices' },
              { value: 'free', label: 'Free Only' },
              { value: 'paid', label: 'Paid Only' },
            ]}
            className="w-32"
          />
        </div>

        {/* Clear Filters */}
        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            Clear Filters
          </Button>
        )}
      </div>

      {/* Results Count */}
      <div className="mb-4 text-sm text-muted-foreground">
        Showing {filteredTemplates.length} of {totalPublished} templates
        {searchQuery && ` matching "${searchQuery}"`}
      </div>

      {/* Template Grid */}
      {filteredTemplates.length === 0 ? (
        <EmptyState
          icon={Store}
          title={searchQuery ? 'No matching templates' : 'No templates available'}
          description={
            searchQuery
              ? 'Try adjusting your search or filters'
              : 'Check back later for new templates'
          }
          action={
            hasActiveFilters
              ? {
                  label: 'Clear Filters',
                  onClick: clearFilters,
                }
              : undefined
          }
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredTemplates.map((template) => (
            <TemplateCard key={template.id} template={template} />
          ))}
        </div>
      )}
    </div>
  );
}
