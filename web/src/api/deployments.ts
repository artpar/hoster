import { api } from './client';
import type { Deployment, CreateDeploymentRequest } from './types';

export const deploymentsApi = {
  list: async () => {
    const response = await api.get<Deployment[]>('/deployments');
    return response.data;
  },

  get: async (id: string) => {
    const response = await api.get<Deployment>(`/deployments/${id}`);
    return response.data;
  },

  create: async (data: CreateDeploymentRequest) => {
    const response = await api.post<Deployment>('/deployments', {
      data: {
        type: 'deployments',
        attributes: {
          name: data.name,
          template_id: data.template_id,
          variables: data.environment_variables || {},
          node_id: data.node_id || '',
        },
      },
    });
    return response.data;
  },

  delete: async (id: string) => {
    await api.delete(`/deployments/${id}`);
  },

  start: async (id: string) => {
    const response = await api.post<Deployment>(`/deployments/${id}/start`);
    return response.data;
  },

  stop: async (id: string) => {
    const response = await api.post<Deployment>(`/deployments/${id}/stop`);
    return response.data;
  },
};
