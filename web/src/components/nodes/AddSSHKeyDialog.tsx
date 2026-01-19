import { useState } from 'react';
import { Key, AlertTriangle } from 'lucide-react';
import { useCreateSSHKey } from '@/hooks/useSSHKeys';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';

interface AddSSHKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (keyId: string) => void;
}

export function AddSSHKeyDialog({
  open,
  onOpenChange,
  onSuccess,
}: AddSSHKeyDialogProps) {
  const createSSHKey = useCreateSSHKey();

  const [name, setName] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    setError(null);

    // Validate
    if (!name.trim()) {
      setError('Key name is required');
      return;
    }

    if (!privateKey.trim()) {
      setError('Private key is required');
      return;
    }

    // Basic validation that it looks like a PEM key
    if (!privateKey.includes('-----BEGIN') || !privateKey.includes('PRIVATE KEY-----')) {
      setError('Invalid private key format. Must be a PEM-encoded private key.');
      return;
    }

    try {
      const key = await createSSHKey.mutateAsync({
        name: name.trim(),
        private_key: privateKey.trim(),
      });
      onOpenChange(false);
      onSuccess?.(key.id);
      // Reset form
      setName('');
      setPrivateKey('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create SSH key');
    }
  };

  const handleClose = () => {
    if (!createSSHKey.isPending) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Add SSH Key</DialogTitle>
          <DialogDescription>
            Upload a private SSH key for connecting to your worker nodes. The key will be
            encrypted before storage.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
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
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={createSSHKey.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={createSSHKey.isPending}>
            <Key className="mr-2 h-4 w-4" />
            {createSSHKey.isPending ? 'Adding...' : 'Add SSH Key'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
