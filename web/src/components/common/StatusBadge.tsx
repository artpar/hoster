import { cn } from '@/lib/cn';
import { allStatuses } from '@/docs/registry';

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const doc = allStatuses[status];
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        doc?.style ?? 'bg-gray-100 text-gray-800',
        className
      )}
    >
      {doc?.label ?? status}
    </span>
  );
}
