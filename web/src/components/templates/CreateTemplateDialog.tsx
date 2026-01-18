import { useState } from 'react';
import { Plus } from 'lucide-react';
import { useCreateTemplate } from '@/hooks/useTemplates';
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

interface CreateTemplateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: (templateId: string) => void;
}

const DEFAULT_COMPOSE = `version: "3.8"

services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"
    environment:
      - NODE_ENV=production
`;

export function CreateTemplateDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateTemplateDialogProps) {
  const createTemplate = useCreateTemplate();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [version, setVersion] = useState('1.0.0');
  const [composeSpec, setComposeSpec] = useState(DEFAULT_COMPOSE);
  const [priceCents, setPriceCents] = useState('0');
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    setError(null);

    // Validate
    if (!name.trim()) {
      setError('Template name is required');
      return;
    }

    if (name.length < 3) {
      setError('Template name must be at least 3 characters');
      return;
    }

    if (!description.trim()) {
      setError('Description is required');
      return;
    }

    if (!/^\d+\.\d+\.\d+$/.test(version)) {
      setError('Version must be in semver format (e.g., 1.0.0)');
      return;
    }

    if (!composeSpec.trim()) {
      setError('Docker Compose specification is required');
      return;
    }

    const price = parseFloat(priceCents);
    if (isNaN(price) || price < 0) {
      setError('Price must be a non-negative number');
      return;
    }

    try {
      const template = await createTemplate.mutateAsync({
        name: name.trim(),
        description: description.trim(),
        version,
        compose_spec: composeSpec,
        price_cents: Math.round(price * 100), // Convert dollars to cents
      });
      onOpenChange(false);
      onSuccess(template.id);
      // Reset form
      setName('');
      setDescription('');
      setVersion('1.0.0');
      setComposeSpec(DEFAULT_COMPOSE);
      setPriceCents('0');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create template');
    }
  };

  const handleClose = () => {
    if (!createTemplate.isPending) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Create New Template</DialogTitle>
          <DialogDescription>
            Create a deployment template that others can use to launch applications.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4 max-h-[60vh] overflow-y-auto">
          {/* Template Name */}
          <div className="grid gap-2">
            <Label htmlFor="name">Template Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Awesome App"
              disabled={createTemplate.isPending}
            />
          </div>

          {/* Description */}
          <div className="grid gap-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="A brief description of what this template provides..."
              rows={3}
              disabled={createTemplate.isPending}
            />
          </div>

          {/* Version and Price */}
          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="version">Version</Label>
              <Input
                id="version"
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                placeholder="1.0.0"
                disabled={createTemplate.isPending}
              />
              <p className="text-xs text-muted-foreground">Semver format (X.Y.Z)</p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="price">Monthly Price (USD)</Label>
              <Input
                id="price"
                type="number"
                min="0"
                step="0.01"
                value={priceCents}
                onChange={(e) => setPriceCents(e.target.value)}
                placeholder="0.00"
                disabled={createTemplate.isPending}
              />
              <p className="text-xs text-muted-foreground">Set to 0 for free templates</p>
            </div>
          </div>

          {/* Docker Compose Spec */}
          <div className="grid gap-2">
            <Label htmlFor="compose">Docker Compose Specification</Label>
            <Textarea
              id="compose"
              value={composeSpec}
              onChange={(e) => setComposeSpec(e.target.value)}
              rows={12}
              disabled={createTemplate.isPending}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              Standard docker-compose.yml format. Ensure all images are publicly available.
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
            disabled={createTemplate.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={createTemplate.isPending}>
            <Plus className="mr-2 h-4 w-4" />
            {createTemplate.isPending ? 'Creating...' : 'Create Template'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
