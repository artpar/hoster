import { useQuery } from '@tanstack/react-query';
import { monitoringApi } from '@/api/monitoring';
import type { LogsQueryParams, EventsQueryParams } from '@/api/monitoring';

export const monitoringKeys = {
  all: ['monitoring'] as const,
  health: (deploymentId: string) => [...monitoringKeys.all, 'health', deploymentId] as const,
  stats: (deploymentId: string) => [...monitoringKeys.all, 'stats', deploymentId] as const,
  logs: (deploymentId: string, params?: LogsQueryParams) =>
    [...monitoringKeys.all, 'logs', deploymentId, params] as const,
  events: (deploymentId: string, params?: EventsQueryParams) =>
    [...monitoringKeys.all, 'events', deploymentId, params] as const,
};

export function useDeploymentHealth(deploymentId: string, status?: string) {
  const isTransitioning = status && ['pending', 'scheduled', 'starting', 'stopping', 'deleting'].includes(status);
  return useQuery({
    queryKey: monitoringKeys.health(deploymentId),
    queryFn: () => monitoringApi.getHealth(deploymentId),
    enabled: !!deploymentId,
    refetchInterval: isTransitioning ? 3000 : 30000,
  });
}

export function useDeploymentStats(deploymentId: string, status?: string) {
  const isTransitioning = status && ['pending', 'scheduled', 'starting', 'stopping', 'deleting'].includes(status);
  return useQuery({
    queryKey: monitoringKeys.stats(deploymentId),
    queryFn: () => monitoringApi.getStats(deploymentId),
    enabled: !!deploymentId,
    refetchInterval: isTransitioning ? 3000 : 10000,
  });
}

export function useDeploymentLogs(deploymentId: string, params?: LogsQueryParams) {
  return useQuery({
    queryKey: monitoringKeys.logs(deploymentId, params),
    queryFn: () => monitoringApi.getLogs(deploymentId, params),
    enabled: !!deploymentId,
  });
}

export function useDeploymentEvents(deploymentId: string, params?: EventsQueryParams) {
  return useQuery({
    queryKey: monitoringKeys.events(deploymentId, params),
    queryFn: () => monitoringApi.getEvents(deploymentId, params),
    enabled: !!deploymentId,
    refetchInterval: 5000, // Refresh every 5 seconds to catch startup events
  });
}
