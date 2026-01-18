import { Link } from 'react-router-dom';
import { Box, Edit, Trash2, Send } from 'lucide-react';
import type { Template } from '@/api/types';
import { StatusBadge } from '@/components/common/StatusBadge';
import { usePublishTemplate, useDeleteTemplate } from '@/hooks/useTemplates';

interface TemplateCardProps {
  template: Template;
  showActions?: boolean;
}

export function TemplateCard({ template, showActions = false }: TemplateCardProps) {
  const publishTemplate = usePublishTemplate();
  const deleteTemplate = useDeleteTemplate();

  const handlePublish = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    await publishTemplate.mutateAsync(template.id);
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (confirm('Are you sure you want to delete this template?')) {
      await deleteTemplate.mutateAsync(template.id);
    }
  };

  return (
    <Link
      to={`/marketplace/${template.id}`}
      className="block rounded-lg border border-border bg-background p-4 transition-shadow hover:shadow-md"
    >
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-md bg-primary/10">
          <Box className="h-5 w-5 text-primary" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="truncate font-medium">{template.attributes.name}</h3>
            <StatusBadge status={template.attributes.status} />
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            v{template.attributes.version}
          </p>
        </div>
        <div className="text-right">
          <p className="font-semibold">
            ${(template.attributes.price_cents / 100).toFixed(2)}
          </p>
          <p className="text-xs text-muted-foreground">/month</p>
        </div>
      </div>

      <p className="mt-3 line-clamp-2 text-sm text-muted-foreground">
        {template.attributes.description}
      </p>

      {showActions && (
        <div className="mt-4 flex gap-2 border-t border-border pt-4">
          {template.attributes.status === 'draft' && (
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
            onClick={handleDelete}
            disabled={deleteTemplate.isPending}
            className="inline-flex items-center gap-1 rounded-md border border-destructive px-3 py-1.5 text-xs text-destructive hover:bg-destructive/10 disabled:opacity-50"
          >
            <Trash2 className="h-3 w-3" />
            Delete
          </button>
        </div>
      )}
    </Link>
  );
}
