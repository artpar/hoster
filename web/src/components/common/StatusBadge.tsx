import { cn } from '@/lib/cn';

type DeploymentStatus =
  | 'pending'
  | 'scheduled'
  | 'starting'
  | 'running'
  | 'stopping'
  | 'stopped'
  | 'deleting'
  | 'deleted'
  | 'failed';

type HealthStatus = 'healthy' | 'unhealthy' | 'degraded' | 'unknown';

type TemplateStatus = 'draft' | 'published' | 'deprecated';

type StatusType = DeploymentStatus | HealthStatus | TemplateStatus;

interface StatusBadgeProps {
  status: StatusType;
  className?: string;
}

const statusStyles: Record<StatusType, string> = {
  // Deployment statuses
  pending: 'bg-blue-100 text-blue-800',
  scheduled: 'bg-blue-100 text-blue-800',
  starting: 'bg-yellow-100 text-yellow-800',
  running: 'bg-green-100 text-green-800',
  stopping: 'bg-yellow-100 text-yellow-800',
  stopped: 'bg-gray-100 text-gray-800',
  deleting: 'bg-red-100 text-red-800',
  deleted: 'bg-gray-100 text-gray-800',
  failed: 'bg-red-100 text-red-800',

  // Health statuses
  healthy: 'bg-green-100 text-green-800',
  unhealthy: 'bg-red-100 text-red-800',
  degraded: 'bg-yellow-100 text-yellow-800',
  unknown: 'bg-gray-100 text-gray-800',

  // Template statuses
  draft: 'bg-gray-100 text-gray-800',
  published: 'bg-green-100 text-green-800',
  deprecated: 'bg-yellow-100 text-yellow-800',
};

const statusLabels: Record<StatusType, string> = {
  pending: 'Pending',
  scheduled: 'Scheduled',
  starting: 'Starting',
  running: 'Running',
  stopping: 'Stopping',
  stopped: 'Stopped',
  deleting: 'Deleting',
  deleted: 'Deleted',
  failed: 'Failed',
  healthy: 'Healthy',
  unhealthy: 'Unhealthy',
  degraded: 'Degraded',
  unknown: 'Unknown',
  draft: 'Draft',
  published: 'Published',
  deprecated: 'Deprecated',
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        statusStyles[status],
        className
      )}
    >
      {statusLabels[status]}
    </span>
  );
}
