import { useQuery } from '@tanstack/react-query';
import { nodesApi } from '@/api/nodes';
import type { Node, CreateNodeRequest, UpdateNodeRequest } from '@/api/types';
import { createResourceHooks, createIdActionHook } from './createResourceHooks';

/**
 * TanStack Query hooks for Node resources.
 *
 * Uses smart polling: 15s when any node is offline, 60s otherwise.
 */
const nodeHooks = createResourceHooks<Node, CreateNodeRequest, UpdateNodeRequest>({
  resourceName: 'nodes',
  api: nodesApi,
});

// Export query keys for external cache management
export const nodeKeys = nodeHooks.keys;

// Override useList with smart polling
export function useNodes() {
  return useQuery({
    queryKey: nodeKeys.lists(),
    queryFn: nodesApi.list,
    refetchInterval: (query) => {
      const nodes = query.state.data || [];
      const hasOffline = nodes.some((n) => n.attributes.status === 'offline');
      return hasOffline ? 15000 : 60000;
    },
  });
}

export const useNode = nodeHooks.useGet;
export const useCreateNode = nodeHooks.useCreate;
export const useUpdateNode = nodeHooks.useUpdate;
export const useDeleteNode = nodeHooks.useDelete;

// Custom action hooks for maintenance mode
export const useEnterMaintenanceMode = createIdActionHook(
  nodeKeys,
  nodesApi.enterMaintenance
);

export const useExitMaintenanceMode = createIdActionHook(
  nodeKeys,
  nodesApi.exitMaintenance
);
