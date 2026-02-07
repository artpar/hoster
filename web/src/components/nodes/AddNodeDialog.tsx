import { useState } from 'react';
import { Plus, Key } from 'lucide-react';
import { useCreateNode } from '@/hooks/useNodes';
import { useSSHKeys } from '@/hooks/useSSHKeys';
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
import { Select } from '@/components/ui/Select';

interface AddNodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (nodeId: string) => void;
  onAddSSHKey?: () => void;
}

const STANDARD_CAPABILITIES = [
  'standard',
  'gpu',
  'high-memory',
  'high-cpu',
  'ssd',
  'nvme',
];

export function AddNodeDialog({
  open,
  onOpenChange,
  onSuccess,
  onAddSSHKey,
}: AddNodeDialogProps) {
  const createNode = useCreateNode();
  const { data: sshKeys } = useSSHKeys();

  const [name, setName] = useState('');
  const [sshHost, setSshHost] = useState('');
  const [sshPort, setSshPort] = useState('22');
  const [sshUser, setSshUser] = useState('');
  const [sshKeyId, setSshKeyId] = useState('');
  const [dockerSocket, setDockerSocket] = useState('/var/run/docker.sock');
  const [location, setLocation] = useState('');
  const [baseDomain, setBaseDomain] = useState('');
  const [capabilities, setCapabilities] = useState<string[]>(['standard']);
  const [error, setError] = useState<string | null>(null);

  const handleCapabilityToggle = (cap: string) => {
    setCapabilities((prev) =>
      prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap]
    );
  };

  const handleCreate = async () => {
    setError(null);

    // Validate
    if (!name.trim()) {
      setError('Node name is required');
      return;
    }

    if (name.length < 3) {
      setError('Node name must be at least 3 characters');
      return;
    }

    if (!sshHost.trim()) {
      setError('SSH host is required');
      return;
    }

    if (!sshUser.trim()) {
      setError('SSH user is required');
      return;
    }

    const port = parseInt(sshPort, 10);
    if (isNaN(port) || port < 1 || port > 65535) {
      setError('SSH port must be between 1 and 65535');
      return;
    }

    if (capabilities.length === 0) {
      setError('At least one capability is required');
      return;
    }

    try {
      const node = await createNode.mutateAsync({
        name: name.trim(),
        ssh_host: sshHost.trim(),
        ssh_port: port,
        ssh_user: sshUser.trim(),
        ssh_key_id: sshKeyId || undefined,
        docker_socket: dockerSocket.trim() || undefined,
        capabilities,
        location: location.trim() || undefined,
        base_domain: baseDomain.trim() || undefined,
      });
      onOpenChange(false);
      onSuccess?.(node.id);
      // Reset form
      setName('');
      setSshHost('');
      setSshPort('22');
      setSshUser('');
      setSshKeyId('');
      setDockerSocket('/var/run/docker.sock');
      setLocation('');
      setBaseDomain('');
      setCapabilities(['standard']);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create node');
    }
  };

  const handleClose = () => {
    if (!createNode.isPending) {
      onOpenChange(false);
    }
  };

  const sshKeyOptions = [
    { value: '', label: 'Select SSH Key...' },
    ...(sshKeys?.map((key) => ({
      value: key.id,
      label: `${key.attributes.name} (${key.attributes.fingerprint.substring(0, 16)}...)`,
    })) || []),
  ];

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Add Worker Node</DialogTitle>
          <DialogDescription>
            Register a VPS server to run deployments. The server must have Docker installed and be
            accessible via SSH.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4 max-h-[60vh] overflow-y-auto">
          {/* Node Name */}
          <div className="grid gap-2">
            <Label htmlFor="name">Node Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Production Server 1"
              disabled={createNode.isPending}
            />
          </div>

          {/* SSH Host and Port */}
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2 grid gap-2">
              <Label htmlFor="ssh-host">SSH Host</Label>
              <Input
                id="ssh-host"
                value={sshHost}
                onChange={(e) => setSshHost(e.target.value)}
                placeholder="192.168.1.100 or server.example.com"
                disabled={createNode.isPending}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="ssh-port">Port</Label>
              <Input
                id="ssh-port"
                type="number"
                min="1"
                max="65535"
                value={sshPort}
                onChange={(e) => setSshPort(e.target.value)}
                disabled={createNode.isPending}
              />
            </div>
          </div>

          {/* SSH User */}
          <div className="grid gap-2">
            <Label htmlFor="ssh-user">SSH User</Label>
            <Input
              id="ssh-user"
              value={sshUser}
              onChange={(e) => setSshUser(e.target.value)}
              placeholder="deploy"
              disabled={createNode.isPending}
            />
            <p className="text-xs text-muted-foreground">
              User must have access to Docker (typically in the docker group)
            </p>
          </div>

          {/* SSH Key */}
          <div className="grid gap-2">
            <Label htmlFor="ssh-key">SSH Key</Label>
            <div className="flex gap-2">
              <Select
                value={sshKeyId}
                onChange={(e) => setSshKeyId(e.target.value)}
                options={sshKeyOptions}
                disabled={createNode.isPending}
                className="flex-1"
              />
              <Button
                type="button"
                variant="outline"
                onClick={onAddSSHKey}
                disabled={createNode.isPending}
              >
                <Key className="mr-2 h-4 w-4" />
                Add Key
              </Button>
            </div>
          </div>

          {/* Docker Socket */}
          <div className="grid gap-2">
            <Label htmlFor="docker-socket">Docker Socket Path</Label>
            <Input
              id="docker-socket"
              value={dockerSocket}
              onChange={(e) => setDockerSocket(e.target.value)}
              placeholder="/var/run/docker.sock"
              disabled={createNode.isPending}
            />
          </div>

          {/* Location */}
          <div className="grid gap-2">
            <Label htmlFor="location">Location (optional)</Label>
            <Input
              id="location"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="us-east-1"
              disabled={createNode.isPending}
            />
          </div>

          {/* Base Domain */}
          <div className="grid gap-2">
            <Label htmlFor="base-domain">Base Domain (optional)</Label>
            <Input
              id="base-domain"
              value={baseDomain}
              onChange={(e) => setBaseDomain(e.target.value)}
              placeholder="apps.example.com"
              disabled={createNode.isPending}
            />
            <p className="text-xs text-muted-foreground">
              Deployments on this node will get subdomains under this base domain (e.g., myapp.apps.example.com)
            </p>
          </div>

          {/* Capabilities */}
          <div className="grid gap-2">
            <Label>Capabilities</Label>
            <div className="flex flex-wrap gap-2">
              {STANDARD_CAPABILITIES.map((cap) => (
                <button
                  key={cap}
                  type="button"
                  onClick={() => handleCapabilityToggle(cap)}
                  disabled={createNode.isPending}
                  className={`rounded-full px-3 py-1 text-sm font-medium transition-colors ${
                    capabilities.includes(cap)
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
                  }`}
                >
                  {cap}
                </button>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              Select capabilities that describe this node's hardware
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
            disabled={createNode.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={createNode.isPending}>
            <Plus className="mr-2 h-4 w-4" />
            {createNode.isPending ? 'Adding...' : 'Add Node'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
