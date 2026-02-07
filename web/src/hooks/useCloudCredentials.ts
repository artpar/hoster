import { cloudCredentialsApi, listRegions, listSizes } from '@/api/cloud-credentials';
import type { CloudCredential, CreateCloudCredentialRequest } from '@/api/types';
import { createResourceHooks } from './createResourceHooks';
import { useQuery } from '@tanstack/react-query';

const hooks = createResourceHooks<CloudCredential, CreateCloudCredentialRequest, never>({
  resourceName: 'cloud_credentials',
  api: cloudCredentialsApi,
  supportsUpdate: false,
});

export const cloudCredentialKeys = hooks.keys;
export const useCloudCredentials = hooks.useList;
export const useCloudCredential = hooks.useGet;
export const useCreateCloudCredential = hooks.useCreate;
export const useDeleteCloudCredential = hooks.useDelete;

export function useProviderRegions(credentialId: string | undefined) {
  return useQuery({
    queryKey: ['cloud_credentials', credentialId, 'regions'],
    queryFn: () => listRegions(credentialId!),
    enabled: !!credentialId,
  });
}

export function useProviderSizes(credentialId: string | undefined) {
  return useQuery({
    queryKey: ['cloud_credentials', credentialId, 'sizes'],
    queryFn: () => listSizes(credentialId!),
    enabled: !!credentialId,
  });
}
