import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { deploymentsApi } from '@/api/deployments';
import type { CreateDeploymentRequest } from '@/api/types';

export const deploymentKeys = {
  all: ['deployments'] as const,
  lists: () => [...deploymentKeys.all, 'list'] as const,
  details: () => [...deploymentKeys.all, 'detail'] as const,
  detail: (id: string) => [...deploymentKeys.details(), id] as const,
};

export function useDeployments() {
  return useQuery({
    queryKey: deploymentKeys.lists(),
    queryFn: deploymentsApi.list,
  });
}

export function useDeployment(id: string) {
  return useQuery({
    queryKey: deploymentKeys.detail(id),
    queryFn: () => deploymentsApi.get(id),
    enabled: !!id,
  });
}

export function useCreateDeployment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateDeploymentRequest) => deploymentsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() });
    },
  });
}

export function useDeleteDeployment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deploymentsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() });
    },
  });
}

export function useStartDeployment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deploymentsApi.start(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: deploymentKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() });
    },
  });
}

export function useStopDeployment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deploymentsApi.stop(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: deploymentKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() });
    },
  });
}
