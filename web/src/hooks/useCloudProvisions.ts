import { cloudProvisionsApi } from '@/api/cloud-provisions';
import type { CloudProvision, CreateCloudProvisionRequest } from '@/api/types';
import { createResourceHooks, createIdActionHook } from './createResourceHooks';

const hooks = createResourceHooks<CloudProvision, CreateCloudProvisionRequest, never>({
  resourceName: 'cloud_provisions',
  api: cloudProvisionsApi,
  supportsUpdate: false,
});

export const cloudProvisionKeys = hooks.keys;
export const useCloudProvisions = hooks.useList;
export const useCloudProvision = hooks.useGet;
export const useCreateCloudProvision = hooks.useCreate;
export const useDeleteCloudProvision = hooks.useDelete;

export const useRetryProvision = createIdActionHook(
  hooks.keys,
  cloudProvisionsApi.retry
);
