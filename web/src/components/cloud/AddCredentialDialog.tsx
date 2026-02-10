import { useState } from 'react';
import { useCreateCloudCredential } from '@/hooks/useCloudCredentials';
import { Button } from '@/components/ui/Button';
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

const providerOptions = [
  { value: 'aws', label: 'AWS' },
  { value: 'digitalocean', label: 'DigitalOcean' },
  { value: 'hetzner', label: 'Hetzner' },
];

interface AddCredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddCredentialDialog({ open, onOpenChange }: AddCredentialDialogProps) {
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
