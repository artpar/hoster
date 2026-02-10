import { useState, useMemo } from 'react';
import {
  Package,
  Plus,
  Search,
  Filter,
} from 'lucide-react';
import { useTemplates } from '@/hooks/useTemplates';
import { useIsAuthenticated, useUser } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { EmptyState } from '@/components/common/EmptyState';
import { TemplateCard } from '@/components/templates/TemplateCard';
import { CreateTemplateDialog } from '@/components/templates/CreateTemplateDialog';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { pages } from '@/docs/registry';

const pageDocs = pages.appTemplates;

type StatusFilter = 'all' | 'draft' | 'published' | 'deprecated';

export function AppTemplatesPage() {
  const isAuthenticated = useIsAuthenticated();
  const user = useUser();
  const userId = user?.id ?? null;
  const { data: templates, isLoading, error } = useTemplates();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  // Filter templates created by this user
  const myTemplates = useMemo(() => {
    if (!templates) return [];
    let result = templates.filter((t) => t.attributes.creator_id === userId);

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (t) =>
          t.attributes.name.toLowerCase().includes(query) ||
          (t.attributes.description?.toLowerCase().includes(query) ?? false)
      );
    }

    if (statusFilter !== 'all') {
      if (statusFilter === 'published') {
        result = result.filter((t) => t.attributes.published);
      } else if (statusFilter === 'draft') {
        result = result.filter((t) => !t.attributes.published);
      }
    }

    return result;
  }, [templates, userId, searchQuery, statusFilter]);

  const handleCreateSuccess = (templateId: string) => {
    console.log('Template created:', templateId);
  };

  if (!isAuthenticated) {
    return (
      <EmptyState
        icon={Package}
        title="Sign in required"
        description="Please sign in to manage your templates"
      />
    );
  }

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
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
          <p className="text-muted-foreground">{pageDocs.subtitle}</p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Template
        </Button>
      </div>

      {/* Search and Filters */}
      <div className="mb-4 flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search your templates..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <Select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            options={[
              { value: 'all', label: 'All Status' },
              { value: 'draft', label: 'Drafts' },
              { value: 'published', label: 'Published' },
              { value: 'deprecated', label: 'Deprecated' },
            ]}
            className="w-36"
          />
        </div>
      </div>

      {/* Templates Grid */}
      {myTemplates.length === 0 ? (
        <EmptyState
          icon={Package}
          title={searchQuery || statusFilter !== 'all' ? 'No matching templates' : pageDocs.emptyState.label}
          description={
            searchQuery || statusFilter !== 'all'
              ? 'Try adjusting your search or filters'
              : pageDocs.emptyState.description
          }
          action={
            !searchQuery && statusFilter === 'all'
              ? {
                  label: 'Create Template',
                  onClick: () => setCreateDialogOpen(true),
                }
              : undefined
          }
        />
      ) : (
        <div className="flex flex-col gap-2">
          {myTemplates.map((template) => (
            <TemplateCard key={template.id} template={template} showActions />
          ))}
        </div>
      )}

      {/* Create Template Dialog */}
      <CreateTemplateDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSuccess={handleCreateSuccess}
      />
    </div>
  );
}
