import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { useCreateCloudCredential } from '@/hooks/useCloudCredentials';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Select } from '@/components/ui/Select';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';

const providerOptions = [
  { value: 'aws', label: 'AWS' },
  { value: 'digitalocean', label: 'DigitalOcean' },
  { value: 'hetzner', label: 'Hetzner' },
];

export function AddCredentialForm() {
  const navigate = useNavigate();
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
      navigate('/nodes/credentials');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create credential');
    }
  };

  return (
    <Card>
      <CardHeader>
        <Link
          to="/nodes/credentials"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-2"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to credentials
        </Link>
        <CardTitle>Add Cloud Credential</CardTitle>
        <CardDescription>
          Store API credentials for a cloud provider. Credentials are encrypted before storage.
        </CardDescription>
      </CardHeader>

      <CardContent>
        <div className="grid gap-4 max-w-lg">
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

          {/* Actions */}
          <div className="flex gap-2 pt-2">
            <Button variant="outline" onClick={() => navigate('/nodes/credentials')} disabled={createCredential.isPending}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createCredential.isPending}>
              {createCredential.isPending ? 'Adding...' : 'Add Credential'}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
