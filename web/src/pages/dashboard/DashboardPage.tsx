import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import {
  Layers,
  Package,
  Server,
  TrendingUp,
  ArrowRight,
} from 'lucide-react';
import { useTemplates } from '@/hooks/useTemplates';
import { useDeployments } from '@/hooks/useDeployments';
import { useNodes } from '@/hooks/useNodes';
import { useUser } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { StatusBadge } from '@/components/common/StatusBadge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { pages } from '@/docs/registry';

const pageDocs = pages.dashboard;

export function DashboardPage() {
  const user = useUser();
  const userId = user?.id ?? null;
  const { data: templates, isLoading: templatesLoading } = useTemplates();
  const { data: deployments, isLoading: deploymentsLoading } = useDeployments();
  const { data: nodes, isLoading: nodesLoading } = useNodes();

  const isLoading = templatesLoading || deploymentsLoading || nodesLoading;

  const myTemplates = useMemo(() => {
    if (!templates) return [];
    return templates.filter((t) => t.attributes.creator_id === userId);
  }, [templates, userId]);

  const stats = useMemo(() => {
    const allDeployments = deployments ?? [];
    const runningDeployments = allDeployments.filter((d) => d.attributes.status === 'running');
    const allNodes = nodes ?? [];
    const onlineNodes = allNodes.filter((n) => n.attributes.status === 'online');
    const publishedTemplates = myTemplates.filter((t) => t.attributes.published);

    const templateIds = new Set(myTemplates.map((t) => t.id));
    const monthlyRevenue = allDeployments
      .filter((d) => d.attributes.status === 'running' && templateIds.has(d.attributes.template_id))
      .reduce((sum, d) => {
        const tmpl = myTemplates.find((t) => t.id === d.attributes.template_id);
        return sum + (tmpl?.attributes.price_monthly_cents ?? 0);
      }, 0);

    return {
      totalDeployments: allDeployments.length,
      runningDeployments: runningDeployments.length,
      totalTemplates: myTemplates.length,
      publishedTemplates: publishedTemplates.length,
      totalNodes: allNodes.length,
      onlineNodes: onlineNodes.length,
      monthlyRevenue,
    };
  }, [deployments, nodes, myTemplates]);

  const recentDeployments = useMemo(() => {
    if (!deployments) return [];
    return [...deployments]
      .sort((a, b) => new Date(b.attributes.created_at).getTime() - new Date(a.attributes.created_at).getTime())
      .slice(0, 5);
  }, [deployments]);

  if (isLoading) {
    return <LoadingPage />;
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
        <p className="text-muted-foreground">{pageDocs.subtitle}</p>
      </div>

      {/* Stats Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Deployments</CardTitle>
            <Layers className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.runningDeployments}</div>
            <p className="text-xs text-muted-foreground">
              {stats.runningDeployments} running / {stats.totalDeployments} total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">App Templates</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.publishedTemplates}</div>
            <p className="text-xs text-muted-foreground">
              {stats.publishedTemplates} published / {stats.totalTemplates} total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Nodes</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.onlineNodes}</div>
            <p className="text-xs text-muted-foreground">
              {stats.onlineNodes} online / {stats.totalNodes} total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Monthly Revenue</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${(stats.monthlyRevenue / 100).toFixed(2)}</div>
            <p className="text-xs text-muted-foreground">From running deployments</p>
          </CardContent>
        </Card>
      </div>

      {/* Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent Deployments */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">Recent Deployments</CardTitle>
            <Link to="/deployments" className="flex items-center gap-1 text-sm text-primary hover:underline">
              View All <ArrowRight className="h-3 w-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {recentDeployments.length === 0 ? (
              <div className="py-8 text-center">
                <Layers className="mx-auto h-8 w-8 text-muted-foreground/50" />
                <p className="mt-2 text-sm text-muted-foreground">No deployments yet</p>
                <Link to="/marketplace" className="mt-1 text-sm text-primary hover:underline">
                  Browse the marketplace
                </Link>
              </div>
            ) : (
              <div className="space-y-3">
                {recentDeployments.map((d) => {
                  const tmpl = templates?.find((t) => t.id === d.attributes.template_id);
                  return (
                    <Link
                      key={d.id}
                      to={`/deployments/${d.id}`}
                      className="flex items-center justify-between rounded-md border p-3 transition-colors hover:bg-accent/50"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium">{d.attributes.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {tmpl?.attributes.name ?? 'Unknown template'}
                        </p>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-xs text-muted-foreground">
                          {new Date(d.attributes.created_at).toLocaleDateString()}
                        </span>
                        <StatusBadge status={d.attributes.status} />
                      </div>
                    </Link>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Node Health Summary */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">Node Health</CardTitle>
            <Link to="/nodes" className="flex items-center gap-1 text-sm text-primary hover:underline">
              Manage Nodes <ArrowRight className="h-3 w-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {!nodes || nodes.length === 0 ? (
              <div className="py-8 text-center">
                <Server className="mx-auto h-8 w-8 text-muted-foreground/50" />
                <p className="mt-2 text-sm text-muted-foreground">No nodes configured</p>
                <Link to="/nodes" className="mt-1 text-sm text-primary hover:underline">
                  Add a node
                </Link>
              </div>
            ) : (
              <div className="space-y-3">
                {nodes.map((n) => (
                  <div key={n.id} className="flex items-center justify-between rounded-md border p-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{n.attributes.name}</p>
                      <p className="text-xs text-muted-foreground">{n.attributes.ssh_host}</p>
                    </div>
                    <StatusBadge status={n.attributes.status} />
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Template Performance */}
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">Template Performance</CardTitle>
            <Link to="/templates" className="flex items-center gap-1 text-sm text-primary hover:underline">
              Manage Templates <ArrowRight className="h-3 w-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {myTemplates.length === 0 ? (
              <div className="py-8 text-center">
                <Package className="mx-auto h-8 w-8 text-muted-foreground/50" />
                <p className="mt-2 text-sm text-muted-foreground">No templates yet</p>
                <Link to="/templates" className="mt-1 text-sm text-primary hover:underline">
                  Create a template
                </Link>
              </div>
            ) : (
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {myTemplates.map((t) => {
                  const depCount = deployments?.filter((d) => d.attributes.template_id === t.id).length ?? 0;
                  const activeCount = deployments?.filter(
                    (d) => d.attributes.template_id === t.id && d.attributes.status === 'running'
                  ).length ?? 0;
                  const activeRevenue = activeCount * (t.attributes.price_monthly_cents ?? 0);
                  return (
                    <div key={t.id} className="rounded-md border p-3">
                      <p className="truncate text-sm font-medium">{t.attributes.name}</p>
                      <div className="mt-1 flex items-center justify-between text-xs text-muted-foreground">
                        <span>{depCount} deployment{depCount !== 1 ? 's' : ''}</span>
                        <span>${(activeRevenue / 100).toFixed(2)}/mo</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
