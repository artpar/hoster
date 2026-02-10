import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Cloud } from 'lucide-react';
import { useCloudProvisions, useDeleteCloudProvision, useRetryProvision } from '@/hooks/useCloudProvisions';
import { EmptyState } from '@/components/common/EmptyState';
import { ProvisionCard } from './ProvisionCard';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function CloudServersTab() {
  const { data: provisions, isLoading } = useCloudProvisions();
  const deleteProvision = useDeleteCloudProvision();
  const retryProvision = useRetryProvision();

  const [destroyDialog, setDestroyDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  const activeProvisions = provisions?.filter(
    (p) => p.attributes.status !== 'ready' && p.attributes.status !== 'destroyed'
  ) || [];

  return (
    <>
      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap gap-2">
        <Link
          to="/nodes/cloud/new"
          className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
        >
          <Plus className="h-4 w-4" />
          Create Cloud Server
        </Link>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : activeProvisions.length === 0 ? (
        <EmptyState
          icon={Cloud}
          title="No cloud servers"
          description="Create a cloud server instance on AWS, DigitalOcean, or Hetzner. It will be automatically configured and registered as a node."
          action={
            <Link
              to="/nodes/cloud/new"
              className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
            >
              <Plus className="h-4 w-4" />
              Create Cloud Server
            </Link>
          }
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {activeProvisions.map((provision) => (
            <ProvisionCard
              key={provision.id}
              provision={provision}
              onRetry={(id) => retryProvision.mutate(id)}
              onDestroy={(id) => {
                const prov = activeProvisions.find((p) => p.id === id);
                setDestroyDialog({ open: true, id, name: prov?.attributes.instance_name || '' });
              }}
              isRetrying={retryProvision.isPending}
            />
          ))}
        </div>
      )}

      {/* Destroy Confirmation */}
      <ConfirmDialog
        open={destroyDialog.open}
        onOpenChange={(open) => setDestroyDialog((prev) => ({ ...prev, open }))}
        title="Destroy Cloud Server"
        description={`This will destroy the cloud instance "${destroyDialog.name}" and remove the associated node. The cloud server will be permanently deleted. This action cannot be undone.`}
        confirmLabel="Destroy"
        variant="destructive"
        onConfirm={() => deleteProvision.mutate(destroyDialog.id)}
      />
    </>
  );
}
