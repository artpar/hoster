import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Key, AlertTriangle } from 'lucide-react';
import { useCreateSSHKey } from '@/hooks/useSSHKeys';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';

export function AddSSHKeyForm() {
  const navigate = useNavigate();
  const createSSHKey = useCreateSSHKey();

  const [name, setName] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    setError(null);

    if (!name.trim()) {
      setError('Key name is required');
      return;
    }

    if (!privateKey.trim()) {
      setError('Private key is required');
      return;
    }

    if (!privateKey.includes('-----BEGIN') || !privateKey.includes('PRIVATE KEY-----')) {
      setError('Invalid private key format. Must be a PEM-encoded private key.');
      return;
    }

    try {
      await createSSHKey.mutateAsync({
        name: name.trim(),
        private_key: privateKey.trim(),
      });
      navigate('/nodes/new');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create SSH key');
    }
  };

  return (
    <Card>
      <CardHeader>
        <Link
          to="/nodes/new"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-2"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to add node
        </Link>
        <CardTitle>Add SSH Key</CardTitle>
        <CardDescription>
          Upload a private SSH key for connecting to your worker nodes. The key will be
          encrypted before storage.
        </CardDescription>
      </CardHeader>

      <CardContent>
        <div className="grid gap-4 max-w-lg">
          {/* Security Notice */}
          <div className="flex items-start gap-2 rounded-md bg-yellow-50 p-3 text-sm text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-200">
            <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
            <div>
              <p className="font-medium">Security Notice</p>
              <p className="mt-1">
                Your private key will be encrypted with AES-256-GCM before storage and is never
                returned by the API. Only use keys specifically generated for Hoster.
              </p>
            </div>
          </div>

          {/* Key Name */}
          <div className="grid gap-2">
            <Label htmlFor="key-name">Key Name</Label>
            <Input
              id="key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Production Server Key"
              disabled={createSSHKey.isPending}
            />
            <p className="text-xs text-muted-foreground">
              A descriptive name to identify this key
            </p>
          </div>

          {/* Private Key */}
          <div className="grid gap-2">
            <Label htmlFor="private-key">Private Key (PEM format)</Label>
            <Textarea
              id="private-key"
              value={privateKey}
              onChange={(e) => setPrivateKey(e.target.value)}
              placeholder={`-----BEGIN OPENSSH PRIVATE KEY-----
...
-----END OPENSSH PRIVATE KEY-----`}
              rows={10}
              disabled={createSSHKey.isPending}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              Paste your private key including the BEGIN and END lines
            </p>
          </div>

          {/* Generation Instructions */}
          <div className="rounded-md bg-secondary/50 p-3 text-sm">
            <p className="font-medium">Generate a new key pair:</p>
            <code className="mt-2 block rounded bg-secondary p-2 text-xs">
              ssh-keygen -t ed25519 -C "hoster-node" -f ~/.ssh/hoster_key
            </code>
            <p className="mt-2 text-muted-foreground">
              Then add the public key (~/.ssh/hoster_key.pub) to your server's authorized_keys
            </p>
          </div>

          {/* Error Message */}
          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2 pt-2">
            <Button
              variant="outline"
              onClick={() => navigate('/nodes/new')}
              disabled={createSSHKey.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createSSHKey.isPending}>
              <Key className="mr-2 h-4 w-4" />
              {createSSHKey.isPending ? 'Adding...' : 'Add SSH Key'}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
