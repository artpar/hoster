import { useState } from 'react';
import { Cloud, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { useCloudCredentials, useDeleteCloudCredential, useProviderRegions, useProviderSizes } from '@/hooks/useCloudCredentials';
import { useCloudProvisions, useDeleteCloudProvision, useRetryProvision, useCreateCloudProvision } from '@/hooks/useCloudProvisions';
import { useCreateCloudCredential } from '@/hooks/useCloudCredentials';
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
  DialogFooter,
} from '@/components/ui/Dialog';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Select } from '@/components/ui/Select';
import type { ProvisionStatus } from '@/api/types';

const providerOptions = [
  { value: 'aws', label: 'AWS' },
  { value: 'digitalocean', label: 'DigitalOcean' },
  { value: 'hetzner', label: 'Hetzner' },
];

function provisionStatusBadge(status: ProvisionStatus) {
  switch (status) {
    case 'pending':
    case 'creating':
    case 'configuring':
      return <Badge variant="warning">{status}</Badge>;
    case 'ready':
      return <Badge variant="success">{status}</Badge>;
    case 'failed':
      return <Badge variant="destructive">{status}</Badge>;
    case 'destroying':
    case 'destroyed':
      return <Badge variant="secondary">{status}</Badge>;
  }
}

// --- Add Credential Dialog ---

interface AddCredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function AddCredentialDialog({ open, onOpenChange }: AddCredentialDialogProps) {
  const createCredential = useCreateCloudCredential();

  const [name, setName] = useState('');
  const [provider, setProvider] = useState<'aws' | 'digitalocean' | 'hetzner'>('aws');
  const [defaultRegion, setDefaultRegion] = useState('');
  const [error, setError] = useState<string | null>(null);

  // AWS fields
  const [accessKeyId, setAccessKeyId] = useState('');
  const [secretAccessKey, setSecretAccessKey] = useState('');
  // DO / Hetzner fields
  const [apiToken, setApiToken] = useState('');

  const resetForm = () => {
    setName('');
    setProvider('aws');
    setDefaultRegion('');
    setAccessKeyId('');
    setSecretAccessKey('');
    setApiToken('');
    setError(null);
  };

  const handleCreate = async () => {
    setError(null);

    if (!name.trim()) {
      setError('Name is required');
      return;
    }

    let credentials: string;
    if (provider === 'aws') {
      if (!accessKeyId.trim() || !secretAccessKey.trim()) {
        setError('Access Key ID and Secret Access Key are required');
        return;
      }
      credentials = JSON.stringify({
        access_key_id: accessKeyId.trim(),
        secret_access_key: secretAccessKey.trim(),
      });
    } else {
      if (!apiToken.trim()) {
        setError('API Token is required');
        return;
      }
      credentials = JSON.stringify({
        api_token: apiToken.trim(),
      });
    }

    try {
      await createCredential.mutateAsync({
        name: name.trim(),
        provider,
        credentials,
        default_region: defaultRegion.trim() || undefined,
      });
      onOpenChange(false);
      resetForm();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create credential');
    }
  };

  const handleClose = () => {
    if (!createCredential.isPending) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Add Cloud Credential</DialogTitle>
          <DialogDescription>
            Store API credentials for a cloud provider. Credentials are encrypted before storage.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* Name */}
          <div className="grid gap-2">
            <Label htmlFor="cred-name">Name</Label>
            <Input
              id="cred-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My AWS Account"
              disabled={createCredential.isPending}
            />
          </div>

          {/* Provider */}
          <div className="grid gap-2">
            <Label htmlFor="cred-provider">Provider</Label>
            <Select
              id="cred-provider"
              options={providerOptions}
              value={provider}
              onChange={(e) => {
                setProvider(e.target.value as 'aws' | 'digitalocean' | 'hetzner');
                setAccessKeyId('');
                setSecretAccessKey('');
                setApiToken('');
              }}
              disabled={createCredential.isPending}
            />
          </div>

          {/* Provider-specific fields */}
          {provider === 'aws' ? (
            <>
              <div className="grid gap-2">
                <Label htmlFor="aws-access-key">Access Key ID</Label>
                <Input
                  id="aws-access-key"
                  value={accessKeyId}
                  onChange={(e) => setAccessKeyId(e.target.value)}
                  placeholder="AKIA..."
                  disabled={createCredential.isPending}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="aws-secret-key">Secret Access Key</Label>
                <Input
                  id="aws-secret-key"
                  type="password"
                  value={secretAccessKey}
                  onChange={(e) => setSecretAccessKey(e.target.value)}
                  placeholder="Secret key"
                  disabled={createCredential.isPending}
                />
              </div>
            </>
          ) : (
            <div className="grid gap-2">
              <Label htmlFor="api-token">API Token</Label>
              <Input
                id="api-token"
                type="password"
                value={apiToken}
                onChange={(e) => setApiToken(e.target.value)}
                placeholder="Token"
                disabled={createCredential.isPending}
              />
            </div>
          )}

          {/* Default Region */}
          <div className="grid gap-2">
            <Label htmlFor="cred-region">Default Region (optional)</Label>
            <Input
              id="cred-region"
              value={defaultRegion}
              onChange={(e) => setDefaultRegion(e.target.value)}
              placeholder={provider === 'aws' ? 'us-east-1' : provider === 'digitalocean' ? 'nyc1' : 'fsn1'}
              disabled={createCredential.isPending}
            />
          </div>

          {/* Error */}
          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={createCredential.isPending}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={createCredential.isPending}>
            {createCredential.isPending ? 'Adding...' : 'Add Credential'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// --- Provision Node Dialog ---

interface ProvisionNodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function ProvisionNodeDialog({ open, onOpenChange }: ProvisionNodeDialogProps) {
  const { data: credentials } = useCloudCredentials();
  const createProvision = useCreateCloudProvision();

  const [credentialId, setCredentialId] = useState('');
  const [instanceName, setInstanceName] = useState('');
  const [region, setRegion] = useState('');
  const [size, setSize] = useState('');
  const [error, setError] = useState<string | null>(null);

  const { data: regions, isLoading: regionsLoading } = useProviderRegions(credentialId || undefined);
  const { data: sizes, isLoading: sizesLoading } = useProviderSizes(credentialId || undefined);

  const credentialOptions = [
    { value: '', label: 'Select a credential...' },
    ...(credentials?.map((c) => ({ value: c.id, label: `${c.attributes.name} (${c.attributes.provider})` })) || []),
  ];

  const regionOptions = [
    { value: '', label: regionsLoading ? 'Loading regions...' : 'Select a region...' },
    ...(regions?.filter((r) => r.available).map((r) => ({ value: r.id, label: r.name })) || []),
  ];

  const sizeOptions = [
    { value: '', label: sizesLoading ? 'Loading sizes...' : 'Select a size...' },
    ...(sizes?.map((s) => ({
      value: s.id,
      label: `${s.name} (${s.cpu_cores} CPU, ${s.memory_mb}MB RAM, ${s.disk_gb}GB) - $${s.price_hourly.toFixed(3)}/hr`,
    })) || []),
  ];

  const resetForm = () => {
    setCredentialId('');
    setInstanceName('');
    setRegion('');
    setSize('');
    setError(null);
  };

  const handleCreate = async () => {
    setError(null);

    if (!credentialId) {
      setError('Select a credential');
      return;
    }
    if (!instanceName.trim()) {
      setError('Instance name is required');
      return;
    }
    if (!region) {
      setError('Select a region');
      return;
    }
    if (!size) {
      setError('Select a size');
      return;
    }

    try {
      await createProvision.mutateAsync({
        credential_id: credentialId,
        instance_name: instanceName.trim(),
        region,
        size,
      });
      onOpenChange(false);
      resetForm();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create provision');
    }
  };

  const handleClose = () => {
    if (!createProvision.isPending) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Provision Cloud Node</DialogTitle>
          <DialogDescription>
            Create a new server instance on a cloud provider. It will be automatically configured and registered as a node.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* Credential */}
          <div className="grid gap-2">
            <Label htmlFor="prov-credential">Cloud Credential</Label>
            <Select
              id="prov-credential"
              options={credentialOptions}
              value={credentialId}
              onChange={(e) => {
                setCredentialId(e.target.value);
                setRegion('');
                setSize('');
              }}
              disabled={createProvision.isPending}
            />
          </div>

          {/* Instance Name */}
          <div className="grid gap-2">
            <Label htmlFor="prov-name">Instance Name</Label>
            <Input
              id="prov-name"
              value={instanceName}
              onChange={(e) => setInstanceName(e.target.value)}
              placeholder="my-node-01"
              disabled={createProvision.isPending}
            />
          </div>

          {/* Region */}
          <div className="grid gap-2">
            <Label htmlFor="prov-region">Region</Label>
            <Select
              id="prov-region"
              options={regionOptions}
              value={region}
              onChange={(e) => setRegion(e.target.value)}
              disabled={!credentialId || createProvision.isPending}
            />
          </div>

          {/* Size */}
          <div className="grid gap-2">
            <Label htmlFor="prov-size">Instance Size</Label>
            <Select
              id="prov-size"
              options={sizeOptions}
              value={size}
              onChange={(e) => setSize(e.target.value)}
              disabled={!credentialId || createProvision.isPending}
            />
          </div>

          {/* Error */}
          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={createProvision.isPending}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={createProvision.isPending}>
            {createProvision.isPending ? 'Provisioning...' : 'Provision Node'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// --- Main Page ---

export function CloudProvidersPage() {
  const { data: credentials, isLoading: credentialsLoading } = useCloudCredentials();
  const { data: provisions, isLoading: provisionsLoading } = useCloudProvisions();
  const deleteCredential = useDeleteCloudCredential();
  const deleteProvision = useDeleteCloudProvision();
  const retryProvision = useRetryProvision();

  const [addCredentialOpen, setAddCredentialOpen] = useState(false);
  const [provisionDialogOpen, setProvisionDialogOpen] = useState(false);
  const [deleteCredentialDialog, setDeleteCredentialDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });
  const [deleteProvisionDialog, setDeleteProvisionDialog] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: '', name: '' });

  // Build credential name lookup for provisions table
  const credentialNames = new Map<string, string>();
  if (credentials) {
    for (const cred of credentials) {
      credentialNames.set(cred.id, cred.attributes.name);
    }
  }

  // Check if a credential is used by any provision
  const credentialInUse = (credId: string): string[] => {
    if (!provisions) return [];
    return provisions
      .filter((p) => p.attributes.credential_id === credId && p.attributes.status !== 'destroyed')
      .map((p) => p.attributes.instance_name);
  };

  const isLoading = credentialsLoading || provisionsLoading;

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Cloud Providers</h1>
          <p className="text-muted-foreground">
            Manage cloud credentials and provision server nodes directly from cloud providers.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setAddCredentialOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add Credential
          </Button>
          <Button onClick={() => setProvisionDialogOpen(true)} disabled={!credentials || credentials.length === 0}>
            <Plus className="mr-2 h-4 w-4" />
            Provision Node
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      ) : (
        <div className="space-y-8">
          {/* Cloud Credentials Section */}
          <section>
            <h2 className="mb-4 text-lg font-semibold">Cloud Credentials</h2>
            {!credentials || credentials.length === 0 ? (
              <EmptyState
                icon={Cloud}
                title="No cloud credentials"
                description="Add API credentials for AWS, DigitalOcean, or Hetzner to start provisioning nodes."
                action={{
                  label: 'Add Credential',
                  onClick: () => setAddCredentialOpen(true),
                }}
              />
            ) : (
              <div className="overflow-hidden rounded-lg border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left font-medium">Name</th>
                      <th className="px-4 py-3 text-left font-medium">Provider</th>
                      <th className="px-4 py-3 text-left font-medium">Default Region</th>
                      <th className="px-4 py-3 text-left font-medium">Created</th>
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
                        <td className="px-4 py-3 text-muted-foreground">
                          {new Date(cred.attributes.created_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            onClick={() => setDeleteCredentialDialog({ open: true, id: cred.id, name: cred.attributes.name })}
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
          </section>

          {/* Cloud Provisions Section */}
          <section>
            <h2 className="mb-4 text-lg font-semibold">Provisioned Nodes</h2>
            {!provisions || provisions.length === 0 ? (
              <EmptyState
                icon={Cloud}
                title="No provisioned nodes"
                description="Provision a cloud server and it will be automatically registered as a deployment node."
                action={
                  credentials && credentials.length > 0
                    ? { label: 'Provision Node', onClick: () => setProvisionDialogOpen(true) }
                    : { label: 'Add Credential First', onClick: () => setAddCredentialOpen(true) }
                }
              />
            ) : (
              <div className="overflow-hidden rounded-lg border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left font-medium">Instance</th>
                      <th className="px-4 py-3 text-left font-medium">Provider</th>
                      <th className="px-4 py-3 text-left font-medium">Region</th>
                      <th className="px-4 py-3 text-left font-medium">Size</th>
                      <th className="px-4 py-3 text-left font-medium">Status</th>
                      <th className="px-4 py-3 text-left font-medium">Public IP</th>
                      <th className="px-4 py-3 text-right font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {provisions.map((prov) => (
                      <tr key={prov.id} className="border-b last:border-0">
                        <td className="px-4 py-3 font-medium">{prov.attributes.instance_name}</td>
                        <td className="px-4 py-3">
                          <Badge variant="outline">{prov.attributes.provider}</Badge>
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">{prov.attributes.region}</td>
                        <td className="px-4 py-3 text-muted-foreground">{prov.attributes.size}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            {provisionStatusBadge(prov.attributes.status)}
                            {prov.attributes.current_step && (
                              <span className="text-xs text-muted-foreground">
                                {prov.attributes.current_step}
                              </span>
                            )}
                          </div>
                          {prov.attributes.error_message && (
                            <p className="mt-1 text-xs text-destructive">{prov.attributes.error_message}</p>
                          )}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                          {prov.attributes.public_ip || '--'}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-1">
                            {prov.attributes.status === 'failed' && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => retryProvision.mutate(prov.id)}
                                disabled={retryProvision.isPending}
                              >
                                <RefreshCw className="mr-1 h-3 w-3" />
                                Retry
                              </Button>
                            )}
                            {prov.attributes.status !== 'destroying' && prov.attributes.status !== 'destroyed' && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-destructive hover:text-destructive"
                                onClick={() => setDeleteProvisionDialog({ open: true, id: prov.id, name: prov.attributes.instance_name })}
                              >
                                <Trash2 className="mr-1 h-3 w-3" />
                                Destroy
                              </Button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>
      )}

      {/* Dialogs */}
      <AddCredentialDialog open={addCredentialOpen} onOpenChange={setAddCredentialOpen} />
      <ProvisionNodeDialog open={provisionDialogOpen} onOpenChange={setProvisionDialogOpen} />

      {/* Delete Credential Confirmation */}
      <ConfirmDialog
        open={deleteCredentialDialog.open}
        onOpenChange={(open) => setDeleteCredentialDialog((prev) => ({ ...prev, open }))}
        title="Delete Cloud Credential"
        description={
          (() => {
            const inUse = credentialInUse(deleteCredentialDialog.id);
            return inUse.length > 0
              ? `This credential is used by active provisions: ${inUse.join(', ')}. Deleting it may prevent management of those instances. Delete "${deleteCredentialDialog.name}"?`
              : `Delete cloud credential "${deleteCredentialDialog.name}"? This cannot be undone.`;
          })()
        }
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => deleteCredential.mutate(deleteCredentialDialog.id)}
      />

      {/* Delete Provision Confirmation */}
      <ConfirmDialog
        open={deleteProvisionDialog.open}
        onOpenChange={(open) => setDeleteProvisionDialog((prev) => ({ ...prev, open }))}
        title="Destroy Cloud Instance"
        description={`This will destroy the cloud instance "${deleteProvisionDialog.name}" and remove the associated node. This action cannot be undone.`}
        confirmLabel="Destroy"
        variant="destructive"
        onConfirm={() => deleteProvision.mutate(deleteProvisionDialog.id)}
      />
    </div>
  );
}
