import { api } from './client';
import type { Template, CreateTemplateRequest } from './types';

export const templatesApi = {
  list: async () => {
    const response = await api.get<Template[]>('/templates');
    return response.data;
  },

  get: async (id: string) => {
    const response = await api.get<Template>(`/templates/${id}`);
    return response.data;
  },

  create: async (data: CreateTemplateRequest) => {
    const response = await api.post<Template>('/templates', {
      data: {
        type: 'templates',
        attributes: data,
      },
    });
    return response.data;
  },

  update: async (id: string, data: Partial<CreateTemplateRequest>) => {
    const response = await api.patch<Template>(`/templates/${id}`, {
      data: {
        type: 'templates',
        id,
        attributes: data,
      },
    });
    return response.data;
  },

  delete: async (id: string) => {
    await api.delete(`/templates/${id}`);
  },

  publish: async (id: string) => {
    const response = await api.post<Template>(`/templates/${id}/publish`);
    return response.data;
  },
};
