import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Box, Edit, Trash2, Send } from 'lucide-react';
import type { Template } from '@/api/types';
import { StatusBadge } from '@/components/common/StatusBadge';
import { usePublishTemplate, useDeleteTemplate } from '@/hooks/useTemplates';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

interface TemplateCardProps {
  template: Template;
  showActions?: boolean;
}

export function TemplateCard({ template, showActions = false }: TemplateCardProps) {
  const publishTemplate = usePublishTemplate();
  const deleteTemplate = useDeleteTemplate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handlePublish = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    await publishTemplate.mutateAsync(template.id);
  };

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    await deleteTemplate.mutateAsync(template.id);
  };

  const price = template.attributes.price_monthly_cents === 0
    ? 'Free'
    : `$${(template.attributes.price_monthly_cents / 100).toFixed(2)}`;

  const resources = template.attributes.resource_requirements;

  return (
    <Link
      to={`/marketplace/${template.id}`}
      className="flex items-center gap-4 rounded-lg border border-border bg-background px-5 py-4 transition-colors hover:bg-accent/40"
    >
      {/* Icon */}
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-primary/10">
        <Box className="h-5 w-5 text-primary" />
      </div>

      {/* Name + description */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <h3 className="font-medium">{template.attributes.name}</h3>
          <StatusBadge status={template.attributes.published ? 'published' : 'draft'} />
          <span className="text-xs text-muted-foreground">v{template.attributes.version}</span>
        </div>
        <p className="mt-0.5 text-sm text-muted-foreground line-clamp-1">
          {template.attributes.description || 'No description available'}
        </p>
      </div>

      {/* Resources */}
      {resources && (
        <div className="hidden shrink-0 items-center gap-3 text-xs text-muted-foreground lg:flex">
          {resources.memory_mb > 0 && <span>{resources.memory_mb} MB</span>}
          {resources.cpu_cores > 0 && <span>{resources.cpu_cores} CPU</span>}
        </div>
      )}

      {/* Actions (app templates page) */}
      {showActions && (
        <div className="flex shrink-0 items-center gap-1.5">
          {!template.attributes.published && (
            <button
              onClick={handlePublish}
              disabled={publishTemplate.isPending}
              className="inline-flex items-center gap-1 rounded-md bg-green-600 px-3 py-1.5 text-xs text-white hover:bg-green-700 disabled:opacity-50"
            >
              <Send className="h-3 w-3" />
              Publish
            </button>
          )}
          <button className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs hover:bg-muted">
            <Edit className="h-3 w-3" />
            Edit
          </button>
          <button
            onClick={handleDeleteClick}
            disabled={deleteTemplate.isPending}
            className="inline-flex items-center gap-1 rounded-md border border-destructive px-3 py-1.5 text-xs text-destructive hover:bg-destructive/10 disabled:opacity-50"
          >
            <Trash2 className="h-3 w-3" />
            Delete
          </button>
        </div>
      )}

      {/* Price */}
      <div className="shrink-0 text-right">
        <p className="font-semibold">{price}</p>
        {template.attributes.price_monthly_cents > 0 && (
          <p className="text-xs text-muted-foreground">/month</p>
        )}
      </div>

      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Delete Template"
        description="Are you sure you want to delete this template? This action cannot be undone."
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
      />
    </Link>
  );
}
