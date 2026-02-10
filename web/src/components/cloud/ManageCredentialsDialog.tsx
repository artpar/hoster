import { useState } from 'react';
import { Plus, Cloud } from 'lucide-react';
import { useCloudCredentials, useDeleteCloudCredential } from '@/hooks/useCloudCredentials';
import { useCloudProvisions } from '@/hooks/useCloudProvisions';
import { EmptyState } from '@/components/common/EmptyState';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/Dialog';
import { AddCredentialDialog } from './AddCredentialDialog';

interface ManageCredentialsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ManageCredentialsDialog({ open, onOpenChange }: ManageCredentialsDialogProps) {
  const { data: credentials, isLoading } = useCloudCredentials();
  const { data: provisions } = useCloudProvisions();
  const deleteCredential = useDeleteCloudCredential();

  const [addCredentialOpen, setAddCredentialOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  const credentialInUse = (credId: string): string[] => {
    if (!provisions) return [];
    return provisions
      .filter((p) => p.attributes.credential_id === credId && p.attributes.status !== 'destroyed')
      .map((p) => p.attributes.instance_name);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Cloud Credentials</DialogTitle>
            <DialogDescription>
              API credentials for cloud providers. Used to provision and manage cloud server instances.
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
              </div>
            ) : !credentials || credentials.length === 0 ? (
              <EmptyState
                icon={Cloud}
                title="No cloud credentials"
                description="Add API credentials for AWS, DigitalOcean, or Hetzner to start provisioning cloud servers."
                action={{
                  label: 'Add Credential',
                  onClick: () => setAddCredentialOpen(true),
                }}
              />
            ) : (
              <div className="space-y-4">
                <div className="overflow-hidden rounded-lg border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-4 py-3 text-left font-medium">Name</th>
                        <th className="px-4 py-3 text-left font-medium">Provider</th>
                        <th className="px-4 py-3 text-left font-medium">Region</th>
                        <th className="px-4 py-3 text-right font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {credentials.map((cred) => (
                        <tr key={cred.id} className="border-b last:border-0">
                          <td className="px-4 py-3 font-medium">{cred.attributes.name}</td>
                          <td className="px-4 py-3">
                            <Badge variant="outline">{cred.attributes.provider}</Badge>
                          </td>
                          <td className="px-4 py-3 text-muted-foreground">
                            {cred.attributes.default_region || '--'}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-destructive hover:text-destructive"
                              onClick={() => setDeleteDialog({ open: true, id: cred.id, name: cred.attributes.name })}
                            >
                              Delete
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <Button variant="outline" size="sm" onClick={() => setAddCredentialOpen(true)}>
                  <Plus className="mr-2 h-4 w-4" />
                  Add Credential
                </Button>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      <AddCredentialDialog open={addCredentialOpen} onOpenChange={setAddCredentialOpen} />

      <ConfirmDialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog((prev) => ({ ...prev, open }))}
        title="Delete Cloud Credential"
        description={
          (() => {
            const inUse = credentialInUse(deleteDialog.id);
            return inUse.length > 0
              ? `This credential is used by active provisions: ${inUse.join(', ')}. Deleting it may prevent management of those instances. Delete "${deleteDialog.name}"?`
              : `Delete cloud credential "${deleteDialog.name}"? This cannot be undone.`;
          })()
        }
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => deleteCredential.mutate(deleteDialog.id)}
      />
    </>
  );
}
