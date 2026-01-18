import { Link } from 'react-router-dom';
import { Layers, ExternalLink } from 'lucide-react';
import type { Deployment } from '@/api/types';
import { StatusBadge } from '@/components/common/StatusBadge';

interface DeploymentCardProps {
  deployment: Deployment;
}

export function DeploymentCard({ deployment }: DeploymentCardProps) {
  return (
    <Link
      to={`/deployments/${deployment.id}`}
      className="block rounded-lg border border-border bg-background p-4 transition-shadow hover:shadow-md"
    >
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-md bg-primary/10">
          <Layers className="h-5 w-5 text-primary" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="truncate font-medium">{deployment.attributes.name}</h3>
            <StatusBadge status={deployment.attributes.status} />
          </div>
          {deployment.attributes.domain && (
            <a
              href={`https://${deployment.attributes.domain}`}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="mt-1 inline-flex items-center gap-1 text-sm text-primary hover:underline"
            >
              {deployment.attributes.domain}
              <ExternalLink className="h-3 w-3" />
            </a>
          )}
        </div>
      </div>

      {deployment.attributes.containers && deployment.attributes.containers.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1">
          {deployment.attributes.containers.map((container) => (
            <span
              key={container.id}
              className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
            >
              {container.service_name}
            </span>
          ))}
        </div>
      )}

      <div className="mt-3 text-xs text-muted-foreground">
        Created {new Date(deployment.attributes.created_at).toLocaleDateString()}
      </div>
    </Link>
  );
}
