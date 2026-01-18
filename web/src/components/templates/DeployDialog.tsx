import { useState } from 'react';
import { Rocket } from 'lucide-react';
import type { Template } from '@/api/types';
import { useCreateDeployment } from '@/hooks/useDeployments';
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

interface DeployDialogProps {
  template: Template;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: (deploymentId: string) => void;
}

export function DeployDialog({
  template,
  open,
  onOpenChange,
  onSuccess,
}: DeployDialogProps) {
  const createDeployment = useCreateDeployment();

  // Generate default name from template slug
  const defaultName = `${template.attributes.slug}-${Date.now().toString(36)}`;

  const [name, setName] = useState(defaultName);
  const [customDomain, setCustomDomain] = useState('');
  const [envVars, setEnvVars] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleDeploy = async () => {
    setError(null);

    // Validate name
    if (!name.trim()) {
      setError('Deployment name is required');
      return;
    }

    // Validate name format (slug-friendly)
    if (!/^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$/.test(name.toLowerCase())) {
      setError(
        'Name must be lowercase alphanumeric with optional hyphens (no leading/trailing hyphens)'
      );
      return;
    }

    // Parse env vars if provided
    let configOverrides: Record<string, string> | undefined;
    if (envVars.trim()) {
      configOverrides = {};
      const lines = envVars.split('\n');
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) continue;
        const eqIndex = trimmed.indexOf('=');
        if (eqIndex === -1) {
          setError(`Invalid environment variable format: ${trimmed}`);
          return;
        }
        const key = trimmed.slice(0, eqIndex);
        const value = trimmed.slice(eqIndex + 1);
        configOverrides[key] = value;
      }
    }

    try {
      const deployment = await createDeployment.mutateAsync({
        name: name.toLowerCase(),
        template_id: template.id,
        custom_domain: customDomain || undefined,
        config_overrides: configOverrides,
      });
      onOpenChange(false);
      onSuccess(deployment.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create deployment');
    }
  };

  const handleClose = () => {
    if (!createDeployment.isPending) {
      onOpenChange(false);
      // Reset form
      setName(defaultName);
      setCustomDomain('');
      setEnvVars('');
      setError(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Deploy {template.attributes.name}</DialogTitle>
          <DialogDescription>
            Configure your deployment settings. Your app will be available at a generated
            domain or your custom domain.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {/* Deployment Name */}
          <div className="grid gap-2">
            <Label htmlFor="name">Deployment Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-app"
              disabled={createDeployment.isPending}
            />
            <p className="text-xs text-muted-foreground">
              Used for the auto-generated domain: {name.toLowerCase()}.yourdomain.com
            </p>
          </div>

          {/* Custom Domain (Optional) */}
          <div className="grid gap-2">
            <Label htmlFor="domain">Custom Domain (Optional)</Label>
            <Input
              id="domain"
              value={customDomain}
              onChange={(e) => setCustomDomain(e.target.value)}
              placeholder="app.example.com"
              disabled={createDeployment.isPending}
            />
            <p className="text-xs text-muted-foreground">
              Point your DNS CNAME to our servers to use a custom domain
            </p>
          </div>

          {/* Environment Variables (Optional) */}
          <div className="grid gap-2">
            <Label htmlFor="envvars">Environment Overrides (Optional)</Label>
            <Textarea
              id="envvars"
              value={envVars}
              onChange={(e) => setEnvVars(e.target.value)}
              placeholder={`# One per line\nDATABASE_URL=postgres://...\nAPI_KEY=secret`}
              rows={4}
              disabled={createDeployment.isPending}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              KEY=value format, one per line. Lines starting with # are ignored.
            </p>
          </div>

          {/* Price Info */}
          <div className="rounded-md bg-muted p-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">Monthly Cost</span>
              <span className="font-semibold">
                ${(template.attributes.price_cents / 100).toFixed(2)}/mo
              </span>
            </div>
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
            disabled={createDeployment.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleDeploy}
            disabled={createDeployment.isPending}
          >
            <Rocket className="mr-2 h-4 w-4" />
            {createDeployment.isPending ? 'Deploying...' : 'Deploy'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
