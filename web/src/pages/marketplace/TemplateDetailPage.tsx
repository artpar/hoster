import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Rocket, Server, Calendar } from 'lucide-react';
import { useTemplate } from '@/hooks/useTemplates';
import { useIsAuthenticated } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { StatusBadge } from '@/components/common/StatusBadge';
import { DeployDialog } from '@/components/templates/DeployDialog';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { AlertDialog } from '@/components/ui/AlertDialog';
import { pages } from '@/docs/registry';

const pageDocs = pages.templateDetail;

export function TemplateDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isAuthenticated = useIsAuthenticated();
  const { data: template, isLoading, error } = useTemplate(id ?? '');
  const [deployDialogOpen, setDeployDialogOpen] = useState(false);
  const [signInAlertOpen, setSignInAlertOpen] = useState(false);

  if (isLoading) {
    return <LoadingPage />;
  }

  if (error || !template) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-destructive">
        {pageDocs.emptyState.label}
      </div>
    );
  }

  const handleDeployClick = () => {
    if (!isAuthenticated) {
      setSignInAlertOpen(true);
      return;
    }
    setDeployDialogOpen(true);
  };

  const handleDeploySuccess = (deploymentId: string) => {
    navigate(`/deployments/${deploymentId}`);
  };

  const isZeroDate = (dateString: string) => {
    return !dateString || dateString.startsWith('0001-01-01');
  };

  const formatDate = (dateString: string) => {
    if (isZeroDate(dateString)) return null;
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  // Parse services from compose spec (basic parsing)
  const parseServices = (composeSpec: string): string[] => {
    try {
      // Very basic YAML parsing for services
      const lines = composeSpec.split('\n');
      let inServices = false;
      const services: string[] = [];
      for (const line of lines) {
        if (line.trim() === 'services:') {
          inServices = true;
          continue;
        }
        if (inServices && line.match(/^  [a-zA-Z0-9_-]+:$/)) {
          services.push(line.trim().replace(':', ''));
        }
        if (inServices && line.match(/^[a-zA-Z]/) && !line.startsWith(' ')) {
          break;
        }
      }
      return services;
    } catch {
      return [];
    }
  };

  const services = parseServices(template.attributes.compose_spec);

  return (
    <div>
      {/* Back Button */}
      <button
        onClick={() => navigate('/templates')}
        className="mb-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Templates
      </button>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main Content */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div>
                  <div className="flex items-center gap-3">
                    <CardTitle className="text-2xl">{template.attributes.name}</CardTitle>
                    <StatusBadge status={template.attributes.published ? 'published' : 'draft'} />
                  </div>
                  <p className="mt-1 text-muted-foreground">
                    Version {template.attributes.version}
                  </p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Description */}
              <div>
                <h3 className="mb-2 font-semibold">Description</h3>
                <p className="text-muted-foreground">{template.attributes.description}</p>
              </div>

              {/* Services */}
              {services.length > 0 && (
                <div>
                  <h3 className="mb-2 font-semibold">{pageDocs.sections.services.label}</h3>
                  <div className="flex flex-wrap gap-2">
                    {services.map((service) => (
                      <Badge key={service} variant="secondary">
                        <Server className="mr-1 h-3 w-3" />
                        {service}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Compose Spec */}
              <div>
                <h3 className="mb-2 font-semibold">{pageDocs.sections.composeSpec.label}</h3>
                <pre className="max-h-96 overflow-auto rounded-md bg-muted p-4 text-sm font-mono">
                  {template.attributes.compose_spec}
                </pre>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Pricing Card */}
          <Card>
            <CardContent className="pt-6">
              <div className="text-center">
                <p className="text-3xl font-bold">
                  {template.attributes.price_monthly_cents === 0
                    ? 'Free'
                    : `$${(template.attributes.price_monthly_cents / 100).toFixed(2)}`}
                  <span className="text-base font-normal text-muted-foreground">/month</span>
                </p>
                <Button
                  onClick={handleDeployClick}
                  className="mt-4 w-full"
                  size="lg"
                >
                  <Rocket className="mr-2 h-4 w-4" />
                  Deploy Now
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Info Card */}
          <Card>
            <CardContent className="pt-6 space-y-4">
              {!isZeroDate(template.attributes.created_at) && (
                <div className="flex items-center gap-3 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <div>
                    <p className="text-muted-foreground">Published</p>
                    <p className="font-medium">{formatDate(template.attributes.created_at)}</p>
                  </div>
                </div>
              )}
              {!isZeroDate(template.attributes.updated_at) && template.attributes.updated_at !== template.attributes.created_at && (
                <div className="flex items-center gap-3 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <div>
                    <p className="text-muted-foreground">Last Updated</p>
                    <p className="font-medium">{formatDate(template.attributes.updated_at)}</p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Deploy Dialog */}
      <DeployDialog
        template={template}
        open={deployDialogOpen}
        onOpenChange={setDeployDialogOpen}
        onSuccess={handleDeploySuccess}
      />

      {/* Sign In Alert */}
      <AlertDialog
        open={signInAlertOpen}
        onOpenChange={setSignInAlertOpen}
        title="Authentication Required"
        description="You need to sign in to deploy templates. Your session may have expired. Please sign in to continue."
        buttonLabel="Sign In"
        onConfirm={() => { window.location.href = '/login'; }}
      />
    </div>
  );
}
