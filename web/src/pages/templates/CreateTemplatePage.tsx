import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Plus } from 'lucide-react';
import { useCreateTemplate } from '@/hooks/useTemplates';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';

const DEFAULT_COMPOSE = `version: "3.8"

services:
  app:
    image: nginx:alpine
    ports:
      - "80:80"
    environment:
      - NODE_ENV=production
`;

export function CreateTemplatePage() {
  const navigate = useNavigate();
  const createTemplate = useCreateTemplate();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [version, setVersion] = useState('1.0.0');
  const [composeSpec, setComposeSpec] = useState(DEFAULT_COMPOSE);
  const [priceCents, setPriceCents] = useState('0');
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    setError(null);

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
        price_monthly_cents: Math.round(price * 100),
      });
      navigate(`/templates/${template.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create template');
    }
  };

  return (
    <div className="mx-auto max-w-2xl">
      <button
        onClick={() => navigate('/templates')}
        className="mb-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Templates
      </button>

      <Card>
        <CardHeader>
          <CardTitle>Create New Template</CardTitle>
          <p className="text-sm text-muted-foreground">
            Define a docker-compose spec, set a price, and publish to the marketplace.
          </p>
        </CardHeader>
        <CardContent className="space-y-6">
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

          <div className="grid gap-2">
            <Label htmlFor="compose">Docker Compose Specification</Label>
            <Textarea
              id="compose"
              value={composeSpec}
              onChange={(e) => setComposeSpec(e.target.value)}
              rows={14}
              disabled={createTemplate.isPending}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              Standard docker-compose.yml format. Ensure all images are publicly available.
            </p>
          </div>

          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button
              variant="outline"
              onClick={() => navigate('/templates')}
              disabled={createTemplate.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createTemplate.isPending}>
              <Plus className="mr-2 h-4 w-4" />
              {createTemplate.isPending ? 'Creating...' : 'Create Template'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
