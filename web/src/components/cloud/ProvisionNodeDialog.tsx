import { useState } from 'react';
import { useCloudCredentials, useProviderRegions, useProviderSizes } from '@/hooks/useCloudCredentials';
import { useCreateCloudProvision } from '@/hooks/useCloudProvisions';
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

interface ProvisionNodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProvisionNodeDialog({ open, onOpenChange }: ProvisionNodeDialogProps) {
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
          <DialogTitle>Create Cloud Server</DialogTitle>
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
            {createProvision.isPending ? 'Provisioning...' : 'Create Server'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
