import { Link } from 'react-router-dom';
import { Layers, Plus } from 'lucide-react';
import { useDeployments } from '@/hooks/useDeployments';
import { useIsAuthenticated } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { EmptyState } from '@/components/common/EmptyState';
import { DeploymentCard } from '@/components/deployments/DeploymentCard';
import { pages } from '@/docs/registry';

const pageDocs = pages.deployments;

export function MyDeploymentsPage() {
  const isAuthenticated = useIsAuthenticated();
  const { data: deployments, isLoading, error } = useDeployments();

  if (!isAuthenticated) {
    return (
      <EmptyState
        icon={Layers}
        title="Sign in required"
        description="Please sign in to view your deployments"
      />
    );
  }

  if (isLoading) {
    return <LoadingPage />;
  }

  if (error) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-destructive">
        Failed to load deployments: {error.message}
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
          <p className="text-muted-foreground">
            {pageDocs.subtitle}
          </p>
        </div>
        <Link
          to="/marketplace"
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          New Deployment
        </Link>
      </div>

      {!deployments || deployments.length === 0 ? (
        <EmptyState
          icon={Layers}
          title={pageDocs.emptyState.label}
          description={pageDocs.emptyState.description}
          action={
            <Link
              to="/marketplace"
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              Browse Marketplace
            </Link>
          }
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {deployments.map((deployment) => (
            <DeploymentCard key={deployment.id} deployment={deployment} />
          ))}
        </div>
      )}
    </div>
  );
}
