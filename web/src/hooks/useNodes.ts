import { nodesApi } from '@/api/nodes';
import type { Node, CreateNodeRequest, UpdateNodeRequest } from '@/api/types';
import { createResourceHooks, createIdActionHook } from './createResourceHooks';

/**
 * TanStack Query hooks for Node resources.
 *
 * Generated from createResourceHooks factory with custom maintenance actions.
 */
const nodeHooks = createResourceHooks<Node, CreateNodeRequest, UpdateNodeRequest>({
  resourceName: 'nodes',
  api: nodesApi,
});

// Export query keys for external cache management
export const nodeKeys = nodeHooks.keys;

// Export standard CRUD hooks with friendly names
export const useNodes = nodeHooks.useList;
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
