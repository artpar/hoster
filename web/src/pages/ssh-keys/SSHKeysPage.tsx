import { useState } from 'react';
import { KeyRound, Plus } from 'lucide-react';
import { useSSHKeys, useDeleteSSHKey } from '@/hooks/useSSHKeys';
import { useNodes } from '@/hooks/useNodes';
import { EmptyState } from '@/components/common/EmptyState';
import { AddSSHKeyDialog } from '@/components/nodes/AddSSHKeyDialog';
import { Button } from '@/components/ui/Button';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function SSHKeysPage() {
  const { data: sshKeys, isLoading } = useSSHKeys();
  const { data: nodes } = useNodes();
  const deleteSSHKey = useDeleteSSHKey();

  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  // Build a map of ssh_key_id -> node names for cross-referencing
  const nodesByKeyId = new Map<string, string[]>();
  if (nodes) {
    for (const node of nodes) {
      const keyId = node.attributes.ssh_key_id;
      if (keyId) {
        const existing = nodesByKeyId.get(keyId) || [];
        existing.push(node.attributes.name);
        nodesByKeyId.set(keyId, existing);
      }
    }
  }

  const keyInUseNodes = deleteDialog.id ? nodesByKeyId.get(deleteDialog.id) || [] : [];

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">SSH Keys</h1>
          <p className="text-muted-foreground">
            Manage SSH keys used to connect to your worker nodes
          </p>
        </div>
        <Button onClick={() => setAddDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add SSH Key
        </Button>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : !sshKeys || sshKeys.length === 0 ? (
        <EmptyState
          icon={KeyRound}
          title="No SSH keys"
          description="Add an SSH key to connect to your worker nodes"
          action={{
            label: 'Add SSH Key',
            onClick: () => setAddDialogOpen(true),
          }}
        />
      ) : (
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left font-medium">Name</th>
                <th className="px-4 py-3 text-left font-medium">Fingerprint</th>
                <th className="px-4 py-3 text-left font-medium">Used By</th>
                <th className="px-4 py-3 text-left font-medium">Created</th>
                <th className="px-4 py-3 text-right font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {sshKeys.map((key) => {
                const usingNodes = nodesByKeyId.get(key.id) || [];
                return (
                  <tr key={key.id} className="border-b last:border-0">
                    <td className="px-4 py-3 font-medium">{key.attributes.name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {key.attributes.fingerprint}
                    </td>
                    <td className="px-4 py-3">
                      {usingNodes.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {usingNodes.map((name) => (
                            <span
                              key={name}
                              className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-xs font-medium"
                            >
                              {name}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="text-xs text-muted-foreground">Not in use</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {new Date(key.attributes.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleteDialog({ open: true, id: key.id, name: key.attributes.name })}
                      >
                        Delete
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Add SSH Key Dialog */}
      <AddSSHKeyDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
      />

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog((prev) => ({ ...prev, open }))}
        title="Delete SSH Key"
        description={
          keyInUseNodes.length > 0
            ? `This key is used by nodes: ${keyInUseNodes.join(', ')}. Deleting it may break connectivity. Delete "${deleteDialog.name}"?`
            : `Delete SSH key "${deleteDialog.name}"? This cannot be undone.`
        }
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => deleteSSHKey.mutate(deleteDialog.id)}
      />
    </div>
  );
}
