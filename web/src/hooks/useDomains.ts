import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { domainsApi } from '@/api/domains';

export function useDomains(deploymentId: string) {
  return useQuery({
    queryKey: ['deployments', deploymentId, 'domains'],
    queryFn: () => domainsApi.list(deploymentId),
    enabled: !!deploymentId,
  });
}

export function useAddDomain(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (hostname: string) => domainsApi.add(deploymentId, hostname),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deployments', deploymentId, 'domains'] });
    },
  });
}

export function useRemoveDomain(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (hostname: string) => domainsApi.remove(deploymentId, hostname),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deployments', deploymentId, 'domains'] });
    },
  });
}

export function useVerifyDomain(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (hostname: string) => domainsApi.verify(deploymentId, hostname),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deployments', deploymentId, 'domains'] });
    },
  });
}
