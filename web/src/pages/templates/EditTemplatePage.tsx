import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Save } from 'lucide-react';
import { useTemplate, useUpdateTemplate } from '@/hooks/useTemplates';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';

export function EditTemplatePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: template, isLoading, error: loadError } = useTemplate(id ?? '');
  const updateTemplate = useUpdateTemplate();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [version, setVersion] = useState('');
  const [composeSpec, setComposeSpec] = useState('');
  const [priceCents, setPriceCents] = useState('0');
  const [error, setError] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  // Pre-fill form when template loads
  useEffect(() => {
    if (template && !initialized) {
      setName(template.attributes.name);
      setDescription(template.attributes.description || '');
      setVersion(template.attributes.version);
      setComposeSpec(template.attributes.compose_spec);
      setPriceCents(String(template.attributes.price_monthly_cents / 100));
      setInitialized(true);
    }
  }, [template, initialized]);

  if (isLoading) {
    return <LoadingPage />;
  }

  if (loadError || !template) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-destructive">
        Template not found. It may have been removed.
      </div>
    );
  }

  const handleSave = async () => {
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
      await updateTemplate.mutateAsync({
        id: template.id,
        data: {
          name: name.trim(),
          description: description.trim(),
          version,
          compose_spec: composeSpec,
          price_monthly_cents: Math.round(price * 100),
        },
      });
      navigate(`/templates/${template.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update template');
    }
  };

  return (
    <div className="mx-auto max-w-2xl">
      <button
        onClick={() => navigate(`/templates/${template.id}`)}
        className="mb-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Template
      </button>

      <Card>
        <CardHeader>
          <CardTitle>Edit Template</CardTitle>
          <p className="text-sm text-muted-foreground">
            Update the template details. Changes are saved when you click Save.
          </p>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-2">
            <Label htmlFor="edit-name">Template Name</Label>
            <Input
              id="edit-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={updateTemplate.isPending}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-description">Description</Label>
            <Textarea
              id="edit-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              disabled={updateTemplate.isPending}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-version">Version</Label>
              <Input
                id="edit-version"
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                disabled={updateTemplate.isPending}
              />
              <p className="text-xs text-muted-foreground">Semver format (X.Y.Z)</p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="edit-price">Monthly Price (USD)</Label>
              <Input
                id="edit-price"
                type="number"
                min="0"
                step="0.01"
                value={priceCents}
                onChange={(e) => setPriceCents(e.target.value)}
                disabled={updateTemplate.isPending}
              />
              <p className="text-xs text-muted-foreground">Set to 0 for free templates</p>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-compose">Docker Compose Specification</Label>
            <Textarea
              id="edit-compose"
              value={composeSpec}
              onChange={(e) => setComposeSpec(e.target.value)}
              rows={14}
              disabled={updateTemplate.isPending}
              className="font-mono text-sm"
            />
          </div>

          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button
              variant="outline"
              onClick={() => navigate(`/templates/${template.id}`)}
              disabled={updateTemplate.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={updateTemplate.isPending}>
              <Save className="mr-2 h-4 w-4" />
              {updateTemplate.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
