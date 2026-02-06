import type { DocEntry } from '@/docs/types';
import { cn } from '@/lib/cn';

interface InfoPanelProps {
  doc: DocEntry;
  className?: string;
}

export function InfoPanel({ doc, className }: InfoPanelProps) {
  return (
    <div className={cn('rounded-lg border bg-muted/30 p-4', className)}>
      <h4 className="text-sm font-medium">{doc.label}</h4>
      <p className="mt-1 text-sm text-muted-foreground">{doc.description}</p>
    </div>
  );
}
