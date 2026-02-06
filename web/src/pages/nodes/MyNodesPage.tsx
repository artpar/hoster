import { useState } from 'react';
import { Plus, Server, Key } from 'lucide-react';
import { useNodes, useDeleteNode, useEnterMaintenanceMode, useExitMaintenanceMode } from '@/hooks/useNodes';
import { useSSHKeys, useDeleteSSHKey } from '@/hooks/useSSHKeys';
import { EmptyState } from '@/components/common/EmptyState';
import { NodeCard, AddNodeDialog, AddSSHKeyDialog } from '@/components/nodes';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function MyNodesPage() {
  const { data: nodes, isLoading: nodesLoading } = useNodes();
  const { data: sshKeys } = useSSHKeys();

  const deleteNode = useDeleteNode();
  const enterMaintenance = useEnterMaintenanceMode();
  const exitMaintenance = useExitMaintenanceMode();
  const deleteSSHKey = useDeleteSSHKey();

  const [addNodeDialogOpen, setAddNodeDialogOpen] = useState(false);
  const [addSSHKeyDialogOpen, setAddSSHKeyDialogOpen] = useState(false);
  const [deleteSSHKeyDialog, setDeleteSSHKeyDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });
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
          <Button variant="outline" onClick={() => setAddSSHKeyDialogOpen(true)}>
            <Key className="mr-2 h-4 w-4" />
            Manage SSH Keys
          </Button>
          <Button onClick={() => setAddNodeDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add Node
          </Button>
        </div>
      </div>

      {/* SSH Keys Summary */}
      {sshKeys && sshKeys.length > 0 && (
        <div className="mb-4 rounded-lg border bg-card p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Key className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">SSH Keys</span>
            </div>
            <div className="flex flex-wrap gap-2">
              {sshKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center gap-1 rounded-full bg-secondary px-2 py-1 text-xs"
                >
                  <span className="font-medium">{key.attributes.name}</span>
                  <span className="text-muted-foreground">
                    ({key.attributes.fingerprint.substring(0, 12)}...)
                  </span>
                  <button
                    onClick={() => setDeleteSSHKeyDialog({ open: true, id: key.id, name: key.attributes.name })}
                    className="ml-1 text-muted-foreground hover:text-destructive"
                  >
                    &times;
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

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

      {/* Delete SSH Key Confirmation */}
      <ConfirmDialog
        open={deleteSSHKeyDialog.open}
        onOpenChange={(open) => setDeleteSSHKeyDialog((prev) => ({ ...prev, open }))}
        title="Delete SSH Key"
        description={`Delete SSH key "${deleteSSHKeyDialog.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => deleteSSHKey.mutate(deleteSSHKeyDialog.id)}
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
