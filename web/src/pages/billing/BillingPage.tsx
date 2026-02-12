import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Activity, DollarSign, Layers } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { useDeployments } from '@/hooks/useDeployments';
import { useTemplates } from '@/hooks/useTemplates';
import { StatusBadge } from '@/components/common/StatusBadge';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { useUser } from '@/stores/authStore';

interface UsageEvent {
  type: string;
  id: string;
  attributes: Record<string, unknown>;
}

export function BillingPage() {
  const user = useUser();
  const { data: deployments, isLoading: deploymentsLoading } = useDeployments();
  const { data: templates, isLoading: templatesLoading } = useTemplates();
  const [usageEvents, setUsageEvents] = useState<UsageEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(true);

  useEffect(() => {
    if (!user) return;
    const token = JSON.parse(localStorage.getItem('hoster-auth') || '{}')?.state?.token;
    if (!token) { setEventsLoading(false); return; }

    fetch(`/api/v1/meter?user_id=${user.id}&page[size]=50`, {
      headers: { 'Authorization': `Bearer ${token}` },
    })
      .then((r) => r.json())
      .then((data) => {
        setUsageEvents(data.data || []);
        setEventsLoading(false);
      })
      .catch(() => setEventsLoading(false));
  }, [user]);

  const isLoading = deploymentsLoading || templatesLoading || eventsLoading;

  const { runningDeployments, monthlyCost, deploymentCosts } = useMemo(() => {
    const allDeployments = deployments ?? [];
    const allTemplates = templates ?? [];
    const running = allDeployments.filter((d) => d.attributes.status === 'running');

    const costs = running.map((d) => {
      const tmpl = allTemplates.find((t) => t.id === String(d.attributes.template_id));
      return {
        deployment: d,
        template: tmpl,
        monthlyCents: tmpl?.attributes.price_monthly_cents ?? 0,
      };
    });

    const total = costs.reduce((sum, c) => sum + c.monthlyCents, 0);

    return {
      runningDeployments: running,
      monthlyCost: total,
      deploymentCosts: costs,
    };
  }, [deployments, templates]);

  if (isLoading) return <LoadingPage />;

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Billing & Usage</h1>
        <p className="text-muted-foreground">
          Your running deployments, costs, and usage activity.
        </p>
      </div>

      {/* Summary Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Monthly Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${(monthlyCost / 100).toFixed(2)}
            </div>
            <p className="text-xs text-muted-foreground">
              From {runningDeployments.length} running deployment{runningDeployments.length !== 1 ? 's' : ''}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Deployments</CardTitle>
            <Layers className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{runningDeployments.length}</div>
            <p className="text-xs text-muted-foreground">
              {(deployments ?? []).length} total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Usage Events</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{usageEvents.length}</div>
            <p className="text-xs text-muted-foreground">Recent tracked events</p>
          </CardContent>
        </Card>
      </div>

      {/* Active Deployments with Costs */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-lg">Running Deployments</CardTitle>
        </CardHeader>
        <CardContent>
          {deploymentCosts.length === 0 ? (
            <div className="py-8 text-center">
              <Layers className="mx-auto h-8 w-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">No running deployments</p>
              <Link to="/marketplace" className="mt-1 text-sm text-primary hover:underline">
                Browse the marketplace
              </Link>
            </div>
          ) : (
            <div className="space-y-2">
              {deploymentCosts.map(({ deployment, template, monthlyCents }) => (
                <Link
                  key={deployment.id}
                  to={`/deployments/${deployment.id}`}
                  className="flex items-center justify-between rounded-md border p-3 transition-colors hover:bg-accent/50"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium">{deployment.attributes.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {template?.attributes.name ?? 'Unknown template'}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-medium">
                      {monthlyCents === 0 ? 'Free' : `$${(monthlyCents / 100).toFixed(2)}/mo`}
                    </span>
                    <StatusBadge status={deployment.attributes.status} />
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Usage Event History */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Usage History</CardTitle>
        </CardHeader>
        <CardContent>
          {usageEvents.length === 0 ? (
            <div className="py-8 text-center">
              <Activity className="mx-auto h-8 w-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">No usage events recorded yet</p>
            </div>
          ) : (
            <div className="space-y-2">
              {usageEvents.slice(0, 20).map((event) => {
                const attrs = event.attributes;
                const eventType = String(attrs.event_type || '');
                const isDeploymentEvent = eventType.startsWith('deployment.');
                const isApiRequest = !isDeploymentEvent && !!attrs.method;

                return (
                  <div
                    key={event.id}
                    className="flex items-center justify-between rounded-md border p-3"
                  >
                    <div className="min-w-0 flex items-center gap-3">
                      {isApiRequest ? (
                        <>
                          <span className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                            {String(attrs.method)}
                          </span>
                          <p className="truncate text-sm text-muted-foreground">
                            {String(attrs.path || 'request')}
                          </p>
                        </>
                      ) : (
                        <div>
                          <p className="text-sm font-medium">
                            {formatEventType(eventType)}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {String(attrs.resource_id || event.id)}
                          </p>
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-3">
                      {isApiRequest && (
                        <span className={`text-xs font-medium ${
                          Number(attrs.status_code || 200) < 400 ? 'text-green-600' : 'text-red-600'
                        }`}>
                          {String(attrs.status_code || '')}
                        </span>
                      )}
                      <span className="text-xs text-muted-foreground">
                        {formatTimestamp(String(attrs.timestamp || ''))}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function formatEventType(type: string): string {
  const labels: Record<string, string> = {
    'deployment.created': 'Deployment Created',
    'deployment.started': 'Deployment Started',
    'deployment.stopped': 'Deployment Stopped',
    'deployment.deleted': 'Deployment Deleted',
  };
  return labels[type] || type;
}

function formatTimestamp(ts: string): string {
  if (!ts) return '';
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return ts;
  }
}
