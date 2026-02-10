import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Server, KeyRound, Settings, Cloud } from 'lucide-react';
import { useNodes, useDeleteNode, useEnterMaintenanceMode, useExitMaintenanceMode } from '@/hooks/useNodes';
import { useCloudProvisions, useDeleteCloudProvision, useRetryProvision } from '@/hooks/useCloudProvisions';
import { EmptyState } from '@/components/common/EmptyState';
import { NodeCard, ProvisionCard, AddNodeDialog, AddSSHKeyDialog } from '@/components/nodes';
import { ProvisionNodeDialog, ManageCredentialsDialog } from '@/components/cloud';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { pages } from '@/docs/registry';

const pageDocs = pages.nodes;

export function MyNodesPage() {
  const { data: nodes, isLoading: nodesLoading } = useNodes();
  const { data: provisions, isLoading: provisionsLoading } = useCloudProvisions();

  const deleteNode = useDeleteNode();
  const enterMaintenance = useEnterMaintenanceMode();
  const exitMaintenance = useExitMaintenanceMode();
  const deleteProvision = useDeleteCloudProvision();
  const retryProvision = useRetryProvision();

  const [addNodeDialogOpen, setAddNodeDialogOpen] = useState(false);
  const [addSSHKeyDialogOpen, setAddSSHKeyDialogOpen] = useState(false);
  const [provisionDialogOpen, setProvisionDialogOpen] = useState(false);
  const [credentialsDialogOpen, setCredentialsDialogOpen] = useState(false);
  const [deleteNodeDialog, setDeleteNodeDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });
  const [destroyProvisionDialog, setDestroyProvisionDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  // Filter provisions: exclude ready (they're nodes now) and destroyed (terminal)
  const activeProvisions = provisions?.filter(
    (p) => p.attributes.status !== 'ready' && p.attributes.status !== 'destroyed'
  ) || [];

  const hasActiveProvisions = activeProvisions.length > 0;
  const isLoading = nodesLoading || provisionsLoading;
  const hasContent = (nodes && nodes.length > 0) || hasActiveProvisions;

  // Find linked provision for a cloud node
  const findProvisionForNode = (provisionId: string | undefined) => {
    if (!provisionId || !provisions) return undefined;
    return provisions.find((p) => p.id === provisionId);
  };

  const handleDestroyCloudNode = (nodeId: string) => {
    const node = nodes?.find((n) => n.id === nodeId);
    if (!node) return;

    const provision = findProvisionForNode(node.attributes.provision_id);
    if (provision) {
      // Destroy via provision (which will also delete the node)
      setDestroyProvisionDialog({ open: true, id: provision.id, name: node.attributes.name });
    } else {
      // No linked provision, just delete the node
      setDeleteNodeDialog({ open: true, id: nodeId, name: node.attributes.name });
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
          <p className="text-muted-foreground">
            Servers where your deployments run
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => setAddNodeDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add Existing Server
          </Button>
          <Button onClick={() => setProvisionDialogOpen(true)}>
            <Cloud className="mr-2 h-4 w-4" />
            Create Cloud Server
          </Button>
          <Link
            to="/ssh-keys"
            className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"
          >
            <KeyRound className="h-4 w-4" />
            SSH Keys
          </Link>
          <Button variant="ghost" size="icon" onClick={() => setCredentialsDialogOpen(true)} title="Manage Credentials">
            <Settings className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Main Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : !hasContent ? (
        <EmptyState
          icon={Server}
          title={pageDocs.emptyState.label}
          description="Add an existing server or provision a cloud server to start deploying applications."
          action={
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setAddNodeDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Add Existing Server
              </Button>
              <Button onClick={() => setProvisionDialogOpen(true)}>
                <Cloud className="mr-2 h-4 w-4" />
                Create Cloud Server
              </Button>
            </div>
          }
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {/* Provision cards first (in-progress) */}
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

          {/* Node cards */}
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

      {/* Node Setup Guide */}
      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-lg">Node Setup Guide</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Before adding an existing server, ensure it is properly configured:
          </p>
          <div className="rounded-md bg-secondary/50 p-4 font-mono text-xs">
            <p className="text-muted-foreground mb-2"># 1. Create deploy user with Docker access</p>
            <p>sudo useradd -m -s /bin/bash deploy</p>
            <p>sudo usermod -aG docker deploy</p>
            <p className="text-muted-foreground mt-4 mb-2"># 2. Set up SSH key authentication</p>
            <p>sudo mkdir -p /home/deploy/.ssh</p>
            <p>echo "YOUR_PUBLIC_KEY" | sudo tee /home/deploy/.ssh/authorized_keys</p>
            <p>sudo chmod 700 /home/deploy/.ssh</p>
            <p>sudo chmod 600 /home/deploy/.ssh/authorized_keys</p>
            <p>sudo chown -R deploy:deploy /home/deploy/.ssh</p>
          </div>
          <p className="text-sm text-muted-foreground">
            Or use <strong>Create Cloud Server</strong> to automatically provision and configure a node on AWS, DigitalOcean, or Hetzner.
          </p>
        </CardContent>
      </Card>

      {/* Dialogs */}
      <AddNodeDialog
        open={addNodeDialogOpen}
        onOpenChange={setAddNodeDialogOpen}
        onSuccess={(nodeId) => {
          console.log('Node created:', nodeId);
        }}
        onAddSSHKey={() => {
          setAddNodeDialogOpen(false);
          setAddSSHKeyDialogOpen(true);
        }}
      />

      <AddSSHKeyDialog
        open={addSSHKeyDialogOpen}
        onOpenChange={setAddSSHKeyDialogOpen}
        onSuccess={(keyId) => {
          console.log('SSH key created:', keyId);
          setAddNodeDialogOpen(true);
        }}
      />

      <ProvisionNodeDialog
        open={provisionDialogOpen}
        onOpenChange={setProvisionDialogOpen}
      />

      <ManageCredentialsDialog
        open={credentialsDialogOpen}
        onOpenChange={setCredentialsDialogOpen}
      />

      {/* Delete Node Confirmation (manual nodes) */}
      <ConfirmDialog
        open={deleteNodeDialog.open}
        onOpenChange={(open) => setDeleteNodeDialog((prev) => ({ ...prev, open }))}
        title="Delete Node"
        description={`Delete node "${deleteNodeDialog.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => deleteNode.mutate(deleteNodeDialog.id)}
      />

      {/* Destroy Provision Confirmation (cloud nodes) */}
      <ConfirmDialog
        open={destroyProvisionDialog.open}
        onOpenChange={(open) => setDestroyProvisionDialog((prev) => ({ ...prev, open }))}
        title="Destroy Cloud Server"
        description={`This will destroy the cloud instance "${destroyProvisionDialog.name}" and remove the associated node. The cloud server will be permanently deleted. This action cannot be undone.`}
        confirmLabel="Destroy"
        variant="destructive"
        onConfirm={() => deleteProvision.mutate(destroyProvisionDialog.id)}
      />
    </div>
  );
}
