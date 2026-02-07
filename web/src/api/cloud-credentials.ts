import { createResourceApi } from './createResourceApi';
import { api } from './client';
import type { CloudCredential, CreateCloudCredentialRequest, ProviderRegion, ProviderInstanceSize } from './types';

export const cloudCredentialsApi = createResourceApi<
  CloudCredential,
  CreateCloudCredentialRequest,
  never,
  never
>({
  resourceName: 'cloud_credentials',
  supportsUpdate: false,
});

// Custom actions for listing regions/sizes
export async function listRegions(credentialId: string): Promise<ProviderRegion[]> {
  const response = await api.get<ProviderRegion[]>(`/cloud_credentials/${credentialId}/regions`);
  return response.data;
}

export async function listSizes(credentialId: string): Promise<ProviderInstanceSize[]> {
  const response = await api.get<ProviderInstanceSize[]>(`/cloud_credentials/${credentialId}/sizes`);
  return response.data;
}
