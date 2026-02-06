import type { MetricDoc } from '@/docs/types';
import { cn } from '@/lib/cn';

interface MetricDefinitionsProps {
  metrics: Record<string, MetricDoc>;
  className?: string;
}

export function MetricDefinitions({ metrics, className }: MetricDefinitionsProps) {
  return (
    <div className={cn('grid gap-2 sm:grid-cols-2', className)}>
      {Object.entries(metrics).map(([key, doc]) => (
        <div key={key} className="rounded-md border p-2">
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-medium">{doc.label}</span>
            {doc.unit && (
              <span className="text-xs text-muted-foreground">({doc.unit})</span>
            )}
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">{doc.description}</p>
        </div>
      ))}
    </div>
  );
}
