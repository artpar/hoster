import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Cloud } from 'lucide-react';
import { toast } from 'sonner';
import { useCloudProvisions, useDeleteCloudProvision, useRetryProvision } from '@/hooks/useCloudProvisions';
import { EmptyState } from '@/components/common/EmptyState';
import { ProvisionCard } from './ProvisionCard';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { Badge } from '@/components/ui/Badge';
import { provisionStatusBadge } from '@/components/cloud';

const providerLabels: Record<string, string> = {
  aws: 'AWS',
  digitalocean: 'DigitalOcean',
  hetzner: 'Hetzner',
};

export function CloudServersTab() {
  const { data: provisions, isLoading } = useCloudProvisions();
  const deleteProvision = useDeleteCloudProvision();
  const retryProvision = useRetryProvision();

  const [destroyDialog, setDestroyDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  const activeProvisions = provisions?.filter(
    (p) => !['ready', 'destroyed'].includes(p.attributes.status)
  ) || [];

  const completedProvisions = provisions?.filter(
    (p) => p.attributes.status === 'ready'
  ) || [];

  const destroyedProvisions = provisions?.filter(
    (p) => p.attributes.status === 'destroyed'
  ) || [];

  const hasAny = (provisions?.length || 0) > 0;

  return (
    <>
      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap gap-2">
        <Link
          to="/nodes/cloud/new"
          className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
        >
          <Plus className="h-4 w-4" />
          Provision Cloud Server
        </Link>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : !hasAny ? (
        <EmptyState
          icon={Cloud}
          title="No provisioning history"
          description="Provision a cloud server on AWS, DigitalOcean, or Hetzner. It will be automatically configured and added to your Nodes."
          action={
            <Link
              to="/nodes/cloud/new"
              className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
            >
              <Plus className="h-4 w-4" />
              Provision Cloud Server
            </Link>
          }
        />
      ) : (
        <div className="space-y-6">
          {/* Active provisions */}
          {activeProvisions.length > 0 && (
            <div>
              <h3 className="mb-3 text-sm font-medium text-muted-foreground uppercase tracking-wider">In Progress</h3>
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
            </div>
          )}

          {/* Completed provisions — became nodes */}
          {completedProvisions.length > 0 && (
            <div>
              <h3 className="mb-3 text-sm font-medium text-muted-foreground uppercase tracking-wider">Completed</h3>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {completedProvisions.map((provision) => (
                  <ProvisionCard
                    key={provision.id}
                    provision={provision}
                    onDestroy={(id) => {
                      const prov = completedProvisions.find((p) => p.id === id);
                      setDestroyDialog({ open: true, id, name: prov?.attributes.instance_name || '' });
                    }}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Destroyed provisions */}
          {destroyedProvisions.length > 0 && (
            <div>
              <h3 className="mb-3 text-sm font-medium text-muted-foreground uppercase tracking-wider">Destroyed</h3>
              <div className="space-y-2">
                {destroyedProvisions.map((p) => (
                  <div key={p.id} className="flex items-center justify-between rounded-lg border border-dashed px-4 py-3 opacity-60">
                    <div className="flex items-center gap-3">
                      <Cloud className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <span className="font-medium">{p.attributes.instance_name}</span>
                        <div className="flex items-center gap-2 mt-0.5">
                          <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                            {providerLabels[p.attributes.provider] || p.attributes.provider}
                          </Badge>
                          <span className="text-xs text-muted-foreground">{p.attributes.region}</span>
                        </div>
                      </div>
                    </div>
                    {provisionStatusBadge(p.attributes.status)}
                  </div>
                ))}
              </div>
            </div>
          )}
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
        loading={deleteProvision.isPending}
        onConfirm={() => deleteProvision.mutate(destroyDialog.id, {
          onSuccess: () => {
            setDestroyDialog((prev) => ({ ...prev, open: false }));
            toast.success(`Cloud server "${destroyDialog.name}" scheduled for destruction`);
          },
          onError: (err) => {
            setDestroyDialog((prev) => ({ ...prev, open: false }));
            toast.error(`Failed to destroy cloud server: ${err.message}`);
          },
        })}
      />
    </>
  );
}
