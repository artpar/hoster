import { api } from './client';

export interface DNSInstruction {
  type: string;
  name: string;
  value: string;
  priority: string;
}

export interface DomainInfo {
  hostname: string;
  type: 'auto' | 'custom';
  ssl_enabled: boolean;
  verification_status?: string;
  verification_method?: string;
  verified_at?: string;
  last_check_error?: string;
  instructions?: DNSInstruction[];
}

export const domainsApi = {
  list: async (deploymentId: string): Promise<DomainInfo[]> => {
    const response = await api.get<DomainInfo[]>(`/deployments/${deploymentId}/domains`);
    return response.data;
  },

  add: async (deploymentId: string, hostname: string): Promise<DomainInfo> => {
    const response = await api.post<DomainInfo>(`/deployments/${deploymentId}/domains`, { hostname });
    return response.data;
  },

  remove: async (deploymentId: string, hostname: string): Promise<void> => {
    await api.delete(`/deployments/${deploymentId}/domains/${hostname}`);
  },

  verify: async (deploymentId: string, hostname: string): Promise<DomainInfo> => {
    const response = await api.post<DomainInfo>(`/deployments/${deploymentId}/domains/${hostname}/verify`);
    return response.data;
  },
};
