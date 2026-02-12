import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Server, KeyRound } from 'lucide-react';
import { toast } from 'sonner';
import { useNodes, useDeleteNode, useEnterMaintenanceMode, useExitMaintenanceMode } from '@/hooks/useNodes';
import { useCloudProvisions, useDeleteCloudProvision, useRetryProvision } from '@/hooks/useCloudProvisions';
import { EmptyState } from '@/components/common/EmptyState';
import { NodeCard } from './NodeCard';
import { ProvisionCard } from './ProvisionCard';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function NodesTab() {
  const { data: nodes, isLoading: nodesLoading } = useNodes({ scope: 'mine' });
  const { data: provisions, isLoading: provisionsLoading } = useCloudProvisions();

  const deleteNode = useDeleteNode();
  const enterMaintenance = useEnterMaintenanceMode();
  const exitMaintenance = useExitMaintenanceMode();
  const deleteProvision = useDeleteCloudProvision();
  const retryProvision = useRetryProvision();

  const [deleteNodeDialog, setDeleteNodeDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });
  const [destroyProvisionDialog, setDestroyProvisionDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  const activeProvisions = provisions?.filter(
    (p) => p.attributes.status !== 'ready' && p.attributes.status !== 'destroyed'
  ) || [];

  const isLoading = nodesLoading || provisionsLoading;
  const hasContent = (nodes && nodes.length > 0) || activeProvisions.length > 0;

  const findProvisionForNode = (provisionId: string | undefined) => {
    if (!provisionId || !provisions) return undefined;
    return provisions.find((p) => p.id === provisionId);
  };

  const handleDestroyCloudNode = (nodeId: string) => {
    const node = nodes?.find((n) => n.id === nodeId);
    if (!node) return;

    const provision = findProvisionForNode(node.attributes.provision_id);
    if (provision && provision.attributes.status !== 'destroyed') {
      setDestroyProvisionDialog({ open: true, id: provision.id, name: node.attributes.name });
    } else {
      setDeleteNodeDialog({ open: true, id: nodeId, name: node.attributes.name });
    }
  };

  return (
    <>
      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap gap-2">
        <Link
          to="/nodes/new"
          className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"
        >
          <Plus className="h-4 w-4" />
          Add Existing Server
        </Link>
        <Link
          to="/ssh-keys"
          className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"
        >
          <KeyRound className="h-4 w-4" />
          SSH Keys
        </Link>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : !hasContent ? (
        <EmptyState
          icon={Server}
          title="No worker nodes"
          description="Add an existing server or provision a cloud server to start deploying applications."
          action={
            <div className="flex gap-2">
              <Link
                to="/nodes/new"
                className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"
              >
                <Plus className="h-4 w-4" />
                Add Existing Server
              </Link>
              <Link
                to="/nodes/cloud/new"
                className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
              >
                Create Cloud Server
              </Link>
            </div>
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
                setDestroyProvisionDialog({ open: true, id, name: prov?.attributes.instance_name || '' });
              }}
              isRetrying={retryProvision.isPending}
            />
          ))}

          {nodes?.map((node) => (
            <NodeCard
              key={node.id}
              node={node}
              onEnterMaintenance={(id) => enterMaintenance.mutate(id)}
              onExitMaintenance={(id) => exitMaintenance.mutate(id)}
              onDelete={(id) => setDeleteNodeDialog({ open: true, id, name: node.attributes.name })}
              onDestroy={(id) => handleDestroyCloudNode(id)}
              isDeleting={deleteNode.isPending}
              isUpdating={enterMaintenance.isPending || exitMaintenance.isPending}
            />
          ))}
        </div>
      )}

      {/* Delete Node Confirmation */}
      <ConfirmDialog
        open={deleteNodeDialog.open}
        onOpenChange={(open) => setDeleteNodeDialog((prev) => ({ ...prev, open }))}
        title="Delete Node"
        description={`Delete node "${deleteNodeDialog.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        loading={deleteNode.isPending}
        onConfirm={() => deleteNode.mutate(deleteNodeDialog.id, {
          onSuccess: () => {
            setDeleteNodeDialog((prev) => ({ ...prev, open: false }));
            toast.success(`Node "${deleteNodeDialog.name}" deleted`);
          },
          onError: (err) => {
            setDeleteNodeDialog((prev) => ({ ...prev, open: false }));
            toast.error(`Failed to delete node: ${err.message}`);
          },
        })}
      />

      {/* Destroy Provision Confirmation */}
      <ConfirmDialog
        open={destroyProvisionDialog.open}
        onOpenChange={(open) => setDestroyProvisionDialog((prev) => ({ ...prev, open }))}
        title="Destroy Cloud Server"
        description={`This will destroy the cloud instance "${destroyProvisionDialog.name}" and remove the associated node. The cloud server will be permanently deleted. This action cannot be undone.`}
        confirmLabel="Destroy"
        variant="destructive"
        loading={deleteProvision.isPending}
        onConfirm={() => deleteProvision.mutate(destroyProvisionDialog.id, {
          onSuccess: () => {
            setDestroyProvisionDialog((prev) => ({ ...prev, open: false }));
            toast.success(`Cloud server "${destroyProvisionDialog.name}" scheduled for destruction`);
          },
          onError: (err) => {
            setDestroyProvisionDialog((prev) => ({ ...prev, open: false }));
            toast.error(`Failed to destroy cloud server: ${err.message}`);
          },
        })}
      />
    </>
  );
}
