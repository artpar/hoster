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
    // JSON:API format (ADR-003)
    // Auth headers (X-User-ID) are automatically added here
    const headers: Record<string, string> = {
      'Content-Type': 'application/vnd.api+json',
    };

    try {
      const authData = localStorage.getItem('hoster-auth');
      if (authData) {
        const parsed = JSON.parse(authData);
        const state = parsed.state;
        if (state?.isAuthenticated && state?.userId) {
          headers['X-User-ID'] = state.userId;
        }
      }
    } catch {
      // Ignore parse errors
    }

    const response = await fetch('/api/v1/deployments', {
      method: 'POST',
      headers,
      credentials: 'include',
      body: JSON.stringify({
        data: {
          type: 'deployments',
          attributes: {
            name: data.name,
            template_id: data.template_id,
            variables: data.environment_variables || {},
          },
        },
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.errors?.[0]?.detail || error.errors?.[0]?.title || 'Failed to create deployment');
    }

    const result = await response.json();
    return result.data; // JSON:API returns { data: { type, id, attributes } }
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
