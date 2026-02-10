import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Plus, Cloud } from 'lucide-react';
import { toast } from 'sonner';
import { useCloudCredentials, useDeleteCloudCredential } from '@/hooks/useCloudCredentials';
import { useCloudProvisions } from '@/hooks/useCloudProvisions';
import { EmptyState } from '@/components/common/EmptyState';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

export function CredentialsTab() {
  const { data: credentials, isLoading } = useCloudCredentials();
  const { data: provisions } = useCloudProvisions();
  const deleteCredential = useDeleteCloudCredential();

  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  const credentialInUse = (credId: string): string[] => {
    if (!provisions) return [];
    return provisions
      .filter((p) => p.attributes.credential_id === credId && p.attributes.status !== 'destroyed')
      .map((p) => p.attributes.instance_name);
  };

  return (
    <>
      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap gap-2">
        <Link
          to="/nodes/credentials/new"
          className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-4 py-2"
        >
          <Plus className="h-4 w-4" />
          Add Credential
        </Link>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : !credentials || credentials.length === 0 ? (
        <EmptyState
          icon={Cloud}
          title="No cloud credentials"
          description="Add API credentials for AWS, DigitalOcean, or Hetzner to start provisioning cloud servers."
          action={
            <Link
              to="/nodes/credentials/new"
              className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
            >
              <Plus className="h-4 w-4" />
              Add Credential
            </Link>
          }
        />
      ) : (
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
      )}

      {/* Delete Confirmation */}
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
        loading={deleteCredential.isPending}
        onConfirm={() => deleteCredential.mutate(deleteDialog.id, {
          onSuccess: () => {
            setDeleteDialog((prev) => ({ ...prev, open: false }));
            toast.success(`Credential "${deleteDialog.name}" deleted`);
          },
          onError: (err) => {
            setDeleteDialog((prev) => ({ ...prev, open: false }));
            toast.error(`Failed to delete credential: ${err.message}`);
          },
        })}
      />
    </>
  );
}
