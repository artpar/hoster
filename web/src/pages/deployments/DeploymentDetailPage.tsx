import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft,
  Play,
  Square,
  Trash2,
  RefreshCw,
  Activity,
  FileText,
  BarChart3,
  Clock,
  ExternalLink,
  Globe,
} from 'lucide-react';
import {
  useDeployment,
  useStartDeployment,
  useStopDeployment,
  useDeleteDeployment,
} from '@/hooks/useDeployments';
import {
  useDeploymentHealth,
  useDeploymentStats,
  useDeploymentLogs,
  useDeploymentEvents,
} from '@/hooks/useMonitoring';
import { LoadingPage, LoadingSpinner } from '@/components/common/LoadingSpinner';
import { StatusBadge } from '@/components/common/StatusBadge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/Tabs';
import { Badge } from '@/components/ui/Badge';
import { Select } from '@/components/ui/Select';

export function DeploymentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: deployment, isLoading, error, refetch } = useDeployment(id ?? '');
  const { data: health, isLoading: healthLoading } = useDeploymentHealth(id ?? '');
  const { data: stats, isLoading: statsLoading } = useDeploymentStats(id ?? '');

  // Logs state
  const [logsTail, setLogsTail] = useState(100);
  const [logsContainer, setLogsContainer] = useState<string | undefined>(undefined);
  const {
    data: logs,
    isLoading: logsLoading,
    refetch: refetchLogs,
  } = useDeploymentLogs(id ?? '', { tail: logsTail, container: logsContainer });

  // Events
  const { data: events, isLoading: eventsLoading } = useDeploymentEvents(id ?? '', {
    limit: 50,
  });

  const startDeployment = useStartDeployment();
  const stopDeployment = useStopDeployment();
  const deleteDeployment = useDeleteDeployment();

  if (isLoading) {
    return <LoadingPage />;
  }

  if (error || !deployment) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-destructive">
        Deployment not found
      </div>
    );
  }

  const handleStart = async () => {
    await startDeployment.mutateAsync(deployment.id);
  };

  const handleStop = async () => {
    await stopDeployment.mutateAsync(deployment.id);
  };

  const handleDelete = async () => {
    if (confirm('Are you sure you want to delete this deployment?')) {
      await deleteDeployment.mutateAsync(deployment.id);
      navigate('/deployments');
    }
  };

  const canStart = ['stopped', 'failed'].includes(deployment.attributes.status);
  const canStop = ['running', 'starting'].includes(deployment.attributes.status);

  const formatBytes = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const containers = health?.containers ?? [];

  return (
    <div>
      {/* Back Button */}
      <button
        onClick={() => navigate('/deployments')}
        className="mb-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Deployments
      </button>

      {/* Header Card */}
      <Card className="mb-6">
        <CardContent className="pt-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold">{deployment.attributes.name}</h1>
                <StatusBadge status={deployment.attributes.status} />
                {health && <StatusBadge status={health.status} />}
              </div>
              {deployment.attributes.domain && (
                <a
                  href={`https://${deployment.attributes.domain}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-2 inline-flex items-center gap-1 text-sm text-primary hover:underline"
                >
                  <Globe className="h-4 w-4" />
                  {deployment.attributes.domain}
                  <ExternalLink className="h-3 w-3" />
                </a>
              )}
            </div>

            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                <RefreshCw className="h-4 w-4" />
              </Button>
              {canStart && (
                <Button
                  onClick={handleStart}
                  disabled={startDeployment.isPending}
                  size="sm"
                  className="bg-green-600 hover:bg-green-700"
                >
                  <Play className="mr-1 h-4 w-4" />
                  Start
                </Button>
              )}
              {canStop && (
                <Button
                  onClick={handleStop}
                  disabled={stopDeployment.isPending}
                  size="sm"
                  className="bg-yellow-600 hover:bg-yellow-700"
                >
                  <Square className="mr-1 h-4 w-4" />
                  Stop
                </Button>
              )}
              <Button
                onClick={handleDelete}
                disabled={deleteDeployment.isPending}
                variant="destructive"
                size="sm"
              >
                <Trash2 className="mr-1 h-4 w-4" />
                Delete
              </Button>
            </div>
          </div>

          {/* Error Message */}
          {deployment.attributes.error_message && (
            <div className="mt-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              <strong>Error:</strong> {deployment.attributes.error_message}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Monitoring Tabs */}
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">
            <Activity className="mr-1 h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="logs">
            <FileText className="mr-1 h-4 w-4" />
            Logs
          </TabsTrigger>
          <TabsTrigger value="stats">
            <BarChart3 className="mr-1 h-4 w-4" />
            Stats
          </TabsTrigger>
          <TabsTrigger value="events">
            <Clock className="mr-1 h-4 w-4" />
            Events
          </TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Container Health Card */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Container Health</CardTitle>
              </CardHeader>
              <CardContent>
                {healthLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <LoadingSpinner />
                  </div>
                ) : containers.length > 0 ? (
                  <div className="space-y-3">
                    {containers.map((container) => (
                      <div
                        key={container.name}
                        className="flex items-center justify-between rounded-md border p-3"
                      >
                        <div>
                          <p className="font-medium">{container.name}</p>
                          <p className="text-sm text-muted-foreground">
                            {container.status} • {container.restarts} restarts
                          </p>
                        </div>
                        <StatusBadge status={container.health} />
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No containers running</p>
                )}
              </CardContent>
            </Card>

            {/* Quick Stats Card */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Resource Usage</CardTitle>
              </CardHeader>
              <CardContent>
                {statsLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <LoadingSpinner />
                  </div>
                ) : stats && stats.containers.length > 0 ? (
                  <div className="space-y-3">
                    {stats.containers.map((container) => (
                      <div key={container.name} className="space-y-2 rounded-md border p-3">
                        <p className="font-medium">{container.name}</p>
                        <div className="grid grid-cols-2 gap-2 text-sm">
                          <div>
                            <span className="text-muted-foreground">CPU:</span>{' '}
                            {container.cpu_percent.toFixed(1)}%
                          </div>
                          <div>
                            <span className="text-muted-foreground">Memory:</span>{' '}
                            {container.memory_percent.toFixed(1)}%
                          </div>
                          <div>
                            <span className="text-muted-foreground">Net RX:</span>{' '}
                            {formatBytes(container.network_rx_bytes)}
                          </div>
                          <div>
                            <span className="text-muted-foreground">Net TX:</span>{' '}
                            {formatBytes(container.network_tx_bytes)}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No stats available</p>
                )}
              </CardContent>
            </Card>

            {/* Deployment Info Card */}
            <Card className="md:col-span-2">
              <CardHeader>
                <CardTitle className="text-lg">Deployment Info</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Created</p>
                    <p className="font-medium">{formatDate(deployment.attributes.created_at)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Last Updated</p>
                    <p className="font-medium">{formatDate(deployment.attributes.updated_at)}</p>
                  </div>
                  {deployment.attributes.started_at && (
                    <div>
                      <p className="text-sm text-muted-foreground">Started At</p>
                      <p className="font-medium">{formatDate(deployment.attributes.started_at)}</p>
                    </div>
                  )}
                  {deployment.attributes.stopped_at && (
                    <div>
                      <p className="text-sm text-muted-foreground">Stopped At</p>
                      <p className="font-medium">{formatDate(deployment.attributes.stopped_at)}</p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Logs Tab */}
        <TabsContent value="logs">
          <Card>
            <CardHeader>
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <CardTitle className="text-lg">Container Logs</CardTitle>
                <div className="flex items-center gap-2">
                  <Select
                    value={logsContainer ?? 'all'}
                    onChange={(e) =>
                      setLogsContainer(e.target.value === 'all' ? undefined : e.target.value)
                    }
                    options={[
                      { value: 'all', label: 'All Containers' },
                      ...containers.map((c) => ({ value: c.name, label: c.name })),
                    ]}
                    className="w-40"
                  />
                  <Select
                    value={String(logsTail)}
                    onChange={(e) => setLogsTail(Number(e.target.value))}
                    options={[
                      { value: '50', label: 'Last 50' },
                      { value: '100', label: 'Last 100' },
                      { value: '200', label: 'Last 200' },
                      { value: '500', label: 'Last 500' },
                    ]}
                    className="w-28"
                  />
                  <Button variant="outline" size="sm" onClick={() => refetchLogs()}>
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {logsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <LoadingSpinner />
                </div>
              ) : logs && logs.logs.length > 0 ? (
                <div className="max-h-[500px] overflow-auto rounded-md bg-muted p-4 font-mono text-xs">
                  {logs.logs.map((log, idx) => (
                    <div
                      key={idx}
                      className={`flex gap-2 ${
                        log.stream === 'stderr' ? 'text-red-500' : ''
                      }`}
                    >
                      <span className="text-muted-foreground">
                        {new Date(log.timestamp).toLocaleTimeString()}
                      </span>
                      <Badge variant="outline" className="h-5 px-1 text-xs">
                        {log.container}
                      </Badge>
                      <span className="flex-1 whitespace-pre-wrap break-all">
                        {log.message}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No logs available</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Stats Tab */}
        <TabsContent value="stats">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Resource Statistics</CardTitle>
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <LoadingSpinner />
                </div>
              ) : stats && stats.containers.length > 0 ? (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="border-b bg-muted/50">
                      <tr>
                        <th className="px-4 py-2 text-left font-medium">Container</th>
                        <th className="px-4 py-2 text-right font-medium">CPU %</th>
                        <th className="px-4 py-2 text-right font-medium">Memory</th>
                        <th className="px-4 py-2 text-right font-medium">Memory %</th>
                        <th className="px-4 py-2 text-right font-medium">Network RX</th>
                        <th className="px-4 py-2 text-right font-medium">Network TX</th>
                        <th className="px-4 py-2 text-right font-medium">Block R/W</th>
                        <th className="px-4 py-2 text-right font-medium">PIDs</th>
                      </tr>
                    </thead>
                    <tbody>
                      {stats.containers.map((container) => (
                        <tr key={container.name} className="border-b last:border-0">
                          <td className="px-4 py-2 font-medium">{container.name}</td>
                          <td className="px-4 py-2 text-right">{container.cpu_percent.toFixed(2)}%</td>
                          <td className="px-4 py-2 text-right">
                            {formatBytes(container.memory_usage_bytes)} /{' '}
                            {formatBytes(container.memory_limit_bytes)}
                          </td>
                          <td className="px-4 py-2 text-right">
                            {container.memory_percent.toFixed(1)}%
                          </td>
                          <td className="px-4 py-2 text-right">
                            {formatBytes(container.network_rx_bytes)}
                          </td>
                          <td className="px-4 py-2 text-right">
                            {formatBytes(container.network_tx_bytes)}
                          </td>
                          <td className="px-4 py-2 text-right">
                            {formatBytes(container.block_read_bytes)} /{' '}
                            {formatBytes(container.block_write_bytes)}
                          </td>
                          <td className="px-4 py-2 text-right">{container.pids}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  <p className="mt-2 text-xs text-muted-foreground">
                    Last updated: {formatDate(stats.collected_at)}
                  </p>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No stats available</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Events Tab */}
        <TabsContent value="events">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Deployment Events</CardTitle>
            </CardHeader>
            <CardContent>
              {eventsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <LoadingSpinner />
                </div>
              ) : events && events.events.length > 0 ? (
                <div className="space-y-2">
                  {events.events.map((event) => (
                    <div
                      key={event.id}
                      className="flex items-start gap-3 rounded-md border p-3"
                    >
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <Badge
                            variant={
                              event.type.includes('error') || event.type.includes('fail')
                                ? 'destructive'
                                : event.type.includes('start')
                                ? 'success'
                                : 'secondary'
                            }
                          >
                            {event.type}
                          </Badge>
                          <span className="text-sm text-muted-foreground">
                            {event.container}
                          </span>
                        </div>
                        <p className="mt-1 text-sm">{event.message}</p>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {formatDate(event.timestamp)}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No events recorded</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
