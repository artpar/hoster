import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Globe,
  Activity,
  Code2,
  Workflow,
  BarChart3,
  Box,
  Edit,
  Trash2,
  Send,
} from 'lucide-react';
import type { Template } from '@/api/types';
import { StatusBadge } from '@/components/common/StatusBadge';
import { usePublishTemplate, useDeleteTemplate } from '@/hooks/useTemplates';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { cn } from '@/lib/cn';

const categoryStyle: Record<string, { icon: typeof Box; bg: string; fg: string }> = {
  web:         { icon: Globe,      bg: 'bg-emerald-50', fg: 'text-emerald-600' },
  monitoring:  { icon: Activity,   bg: 'bg-rose-50',    fg: 'text-rose-600' },
  development: { icon: Code2,      bg: 'bg-sky-50',     fg: 'text-sky-600' },
  automation:  { icon: Workflow,    bg: 'bg-orange-50',  fg: 'text-orange-600' },
  analytics:   { icon: BarChart3,  bg: 'bg-indigo-50',  fg: 'text-indigo-600' },
};

const defaultStyle = { icon: Box, bg: 'bg-gray-50', fg: 'text-gray-600' };

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

  const cat = template.attributes.category || '';
  const style = categoryStyle[cat] || defaultStyle;
  const Icon = style.icon;

  const priceCents = template.attributes.price_monthly_cents;
  const priceLabel = priceCents === 0
    ? 'Free'
    : `$${(priceCents / 100).toFixed(2)}`;

  const resources = template.attributes.resource_requirements;
  const isDraft = !template.attributes.published;

  return (
    <Link
      to={`/marketplace/${template.id}`}
      className="group flex items-center gap-4 rounded-lg border border-border bg-background px-5 py-3.5 transition-all hover:border-primary/20 hover:shadow-sm"
    >
      {/* Category icon */}
      <div className={cn('flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', style.bg)}>
        <Icon className={cn('h-5 w-5', style.fg)} />
      </div>

      {/* Name + description */}
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <h3 className="font-medium group-hover:text-primary transition-colors">
            {template.attributes.name}
          </h3>
          {isDraft && <StatusBadge status="draft" />}
          <span className="text-xs text-muted-foreground/60">
            v{template.attributes.version}
          </span>
        </div>
        <p className="mt-0.5 text-sm text-muted-foreground line-clamp-1">
          {template.attributes.description || 'No description available'}
        </p>
      </div>

      {/* Resources */}
      {resources && (
        <div className="hidden shrink-0 lg:flex">
          <div className="flex items-center gap-1 rounded-md bg-muted/60 px-2.5 py-1 text-xs text-muted-foreground">
            {resources.memory_mb > 0 && <span>{resources.memory_mb} MB</span>}
            {resources.memory_mb > 0 && resources.cpu_cores > 0 && (
              <span className="text-border">&middot;</span>
            )}
            {resources.cpu_cores > 0 && <span>{resources.cpu_cores} CPU</span>}
          </div>
        </div>
      )}

      {/* Actions (app templates page) */}
      {showActions && (
        <div className="flex shrink-0 items-center gap-1.5">
          {isDraft && (
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
      <div className="w-20 shrink-0 text-right">
        <span className="text-sm font-semibold">{priceLabel}</span>
        {priceCents > 0 && (
          <span className="text-xs text-muted-foreground">/mo</span>
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
