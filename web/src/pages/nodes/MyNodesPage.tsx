import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Server, KeyRound } from 'lucide-react';
import { useNodes, useDeleteNode, useEnterMaintenanceMode, useExitMaintenanceMode } from '@/hooks/useNodes';
import { EmptyState } from '@/components/common/EmptyState';
import { NodeCard, AddNodeDialog, AddSSHKeyDialog } from '@/components/nodes';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function MyNodesPage() {
  const { data: nodes, isLoading: nodesLoading } = useNodes();

  const deleteNode = useDeleteNode();
  const enterMaintenance = useEnterMaintenanceMode();
  const exitMaintenance = useExitMaintenanceMode();

  const [addNodeDialogOpen, setAddNodeDialogOpen] = useState(false);
  const [addSSHKeyDialogOpen, setAddSSHKeyDialogOpen] = useState(false);
  const [deleteNodeDialog, setDeleteNodeDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">My Nodes</h1>
          <p className="text-muted-foreground">
            Manage your VPS servers for running deployments
          </p>
        </div>
        <div className="flex gap-2">
          <Link
            to="/ssh-keys"
            className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"
          >
            <KeyRound className="h-4 w-4" />
            Manage SSH Keys
          </Link>
          <Button onClick={() => setAddNodeDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add Node
          </Button>
        </div>
      </div>

      {/* Nodes Grid */}
      {nodesLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : !nodes || nodes.length === 0 ? (
        <EmptyState
          icon={Server}
          title="No worker nodes"
          description="Add your first VPS server to start running deployments on your own infrastructure"
          action={{
            label: 'Add Node',
            onClick: () => setAddNodeDialogOpen(true),
          }}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {nodes.map((node) => (
            <NodeCard
              key={node.id}
              node={node}
              onEnterMaintenance={(id) => enterMaintenance.mutate(id)}
              onExitMaintenance={(id) => exitMaintenance.mutate(id)}
              onDelete={(id) => setDeleteNodeDialog({ open: true, id, name: node.attributes.name })}
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
            Before adding a node, ensure your VPS is properly configured:
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
        </CardContent>
      </Card>

      {/* Add Node Dialog */}
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

      {/* Add SSH Key Dialog */}
      <AddSSHKeyDialog
        open={addSSHKeyDialogOpen}
        onOpenChange={setAddSSHKeyDialogOpen}
        onSuccess={(keyId) => {
          console.log('SSH key created:', keyId);
          setAddNodeDialogOpen(true);
        }}
      />

      {/* Delete Node Confirmation */}
      <ConfirmDialog
        open={deleteNodeDialog.open}
        onOpenChange={(open) => setDeleteNodeDialog((prev) => ({ ...prev, open }))}
        title="Delete Node"
        description={`Delete node "${deleteNodeDialog.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => deleteNode.mutate(deleteNodeDialog.id)}
      />
    </div>
  );
}
