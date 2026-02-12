import { useEffect, useState } from 'react';
import { CreditCard, Activity, TrendingUp, Clock } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { useUser } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';

interface UsageEvent {
  type: string;
  id: string;
  attributes: Record<string, unknown>;
}

interface UsageData {
  data: UsageEvent[];
}

export function BillingPage() {
  const user = useUser();
  const [usage, setUsage] = useState<UsageEvent[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!user) return;
    const token = JSON.parse(localStorage.getItem('hoster-auth') || '{}')?.state?.token;
    if (!token) return;
    const headers = { 'Authorization': `Bearer ${token}` };

    fetch(`/api/v1/meter?user_id=${user.id}&page[size]=50`, { headers })
      .then((r) => r.json())
      .then((data: UsageData) => {
        setUsage(data.data || []);
        setIsLoading(false);
      })
      .catch(() => setIsLoading(false));
  }, [user]);

  if (isLoading) return <LoadingPage />;

  const deploymentEvents = usage.filter((e) => {
    const et = String(e.attributes.event_type || '');
    return et.startsWith('deployment.');
  });
  const apiEvents = usage.filter((e) => {
    const et = String(e.attributes.event_type || '');
    return !et.startsWith('deployment.') || et === '';
  });

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Billing & Usage</h1>
        <p className="text-muted-foreground">
          Your plan, usage events, and billing activity.
        </p>
      </div>

      {/* Plan Card */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Current Plan</CardTitle>
            <CreditCard className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold capitalize">{user?.plan_id || 'Free'}</div>
            <p className="text-xs text-muted-foreground">
              {user?.plan_limits.max_deployments ?? 1} deployment{(user?.plan_limits.max_deployments ?? 1) !== 1 ? 's' : ''} max
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">API Requests</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{apiEvents.length}</div>
            <p className="text-xs text-muted-foreground">Recent tracked requests</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Deployment Events</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{deploymentEvents.length}</div>
            <p className="text-xs text-muted-foreground">Create, start, stop, delete</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Plan Limits</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-sm space-y-1">
              <div className="flex justify-between">
                <span className="text-muted-foreground">CPU</span>
                <span className="font-medium">{user?.plan_limits.max_cpu_cores ?? 1} core{(user?.plan_limits.max_cpu_cores ?? 1) !== 1 ? 's' : ''}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Memory</span>
                <span className="font-medium">{user?.plan_limits.max_memory_mb ?? 1024} MB</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Disk</span>
                <span className="font-medium">{user?.plan_limits.max_disk_gb ?? 5} GB</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Deployment Events */}
      {deploymentEvents.length > 0 && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-lg">Deployment Events</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {deploymentEvents.map((event) => (
                <div
                  key={event.id}
                  className="flex items-center justify-between rounded-md border p-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium">
                      {formatEventType(String(event.attributes.event_type || 'unknown'))}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {String(event.attributes.resource_id || event.id)}
                    </p>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {formatTimestamp(String(event.attributes.timestamp || ''))}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Recent API Activity */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Recent API Activity</CardTitle>
        </CardHeader>
        <CardContent>
          {apiEvents.length === 0 ? (
            <div className="py-8 text-center">
              <Activity className="mx-auto h-8 w-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">No API activity recorded yet</p>
            </div>
          ) : (
            <div className="space-y-2">
              {apiEvents.slice(0, 20).map((event) => (
                <div
                  key={event.id}
                  className="flex items-center justify-between rounded-md border p-3"
                >
                  <div className="min-w-0 flex items-center gap-3">
                    <span className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                      {String(event.attributes.method || 'GET')}
                    </span>
                    <p className="truncate text-sm text-muted-foreground">
                      {String(event.attributes.path || event.attributes.event_type || 'request')}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className={`text-xs font-medium ${
                      Number(event.attributes.status_code || 200) < 400
                        ? 'text-green-600'
                        : 'text-red-600'
                    }`}>
                      {String(event.attributes.status_code || '')}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {formatTimestamp(String(event.attributes.timestamp || ''))}
                    </span>
                  </div>
                </div>
              ))}
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
