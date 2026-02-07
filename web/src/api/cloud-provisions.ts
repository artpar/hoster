import { createResourceApi } from './createResourceApi';
import type { CloudProvision, CreateCloudProvisionRequest } from './types';

export const cloudProvisionsApi = createResourceApi<
  CloudProvision,
  CreateCloudProvisionRequest,
  never,
  'retry'
>({
  resourceName: 'cloud_provisions',
  supportsUpdate: false,
  customActions: {
    retry: { method: 'POST', path: 'retry', requiresId: true },
  },
});
