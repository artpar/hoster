import type { StatusDoc } from '@/docs/types';
import { cn } from '@/lib/cn';

interface StatusLegendProps {
  statuses: Record<string, StatusDoc>;
  className?: string;
}

export function StatusLegend({ statuses, className }: StatusLegendProps) {
  return (
    <div className={cn('grid gap-2 sm:grid-cols-2', className)}>
      {Object.entries(statuses).map(([key, doc]) => (
        <div key={key} className="flex items-start gap-2 rounded-md border p-2">
          <span
            className={cn(
              'mt-0.5 inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-xs font-medium',
              doc.style
            )}
          >
            {doc.label}
          </span>
          <span className="text-xs text-muted-foreground">{doc.description}</span>
        </div>
      ))}
    </div>
  );
}
