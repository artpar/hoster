import { useEffect, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { cloudProvisionsApi } from '@/api/cloud-provisions';
import type { CloudProvision, CreateCloudProvisionRequest } from '@/api/types';
import { createResourceHooks, createIdActionHook } from './createResourceHooks';
import { nodeKeys } from './useNodes';

const hooks = createResourceHooks<CloudProvision, CreateCloudProvisionRequest, never>({
  resourceName: 'cloud_provisions',
  api: cloudProvisionsApi,
  supportsUpdate: false,
});

export const cloudProvisionKeys = hooks.keys;

const ACTIVE_STATUSES = ['pending', 'creating', 'configuring', 'destroying'];

/**
 * Lists cloud provisions with smart polling.
 * Polls every 3s when any provision is in an active state.
 * Invalidates nodes query when a provision transitions to ready.
 */
export function useCloudProvisions() {
  const queryClient = useQueryClient();
  const prevStatusesRef = useRef<Record<string, string>>({});

  const query = useQuery({
    queryKey: cloudProvisionKeys.lists(),
    queryFn: () => cloudProvisionsApi.list(),
    refetchInterval: (q) => {
      const provisions = q.state.data || [];
      const hasActive = provisions.some((p) =>
        ACTIVE_STATUSES.includes(p.attributes.status)
      );
      return hasActive ? 3000 : false;
    },
  });

  // Invalidate nodes when a provision transitions to ready
  useEffect(() => {
    const provisions = query.data || [];
    const currentStatuses: Record<string, string> = {};
    let shouldInvalidateNodes = false;

    for (const p of provisions) {
      currentStatuses[p.id] = p.attributes.status;
      const prev = prevStatusesRef.current[p.id];
      if (prev && prev !== 'ready' && p.attributes.status === 'ready') {
        shouldInvalidateNodes = true;
      }
    }

    prevStatusesRef.current = currentStatuses;

    if (shouldInvalidateNodes) {
      queryClient.invalidateQueries({ queryKey: nodeKeys.lists() });
    }
  }, [query.data, queryClient]);

  return query;
}

export const useCloudProvision = hooks.useGet;
export const useCreateCloudProvision = hooks.useCreate;
export const useDeleteCloudProvision = hooks.useDelete;

export const useRetryProvision = createIdActionHook(
  hooks.keys,
  cloudProvisionsApi.retry
);
