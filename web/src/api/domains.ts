import { useAuthStore } from '../stores/authStore';

const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

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

/** Raw fetch for domain endpoints — they return plain JSON, not JSON:API wrapped. */
async function domainFetch<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };
  const token = useAuthStore.getState().token;
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const response = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    headers: { ...headers, ...options.headers },
  });
  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText);
    throw new Error(text || `Request failed: ${response.status}`);
  }
  if (response.status === 204) return null as T;
  return response.json();
}

export const domainsApi = {
  list: async (deploymentId: string): Promise<DomainInfo[]> => {
    return domainFetch<DomainInfo[]>(`/deployments/${deploymentId}/domains`);
  },

  add: async (deploymentId: string, hostname: string): Promise<DomainInfo> => {
    return domainFetch<DomainInfo>(`/deployments/${deploymentId}/domains`, {
      method: 'POST',
      body: JSON.stringify({ hostname }),
    });
  },

  remove: async (deploymentId: string, hostname: string): Promise<void> => {
    await domainFetch<void>(`/deployments/${deploymentId}/domains/${hostname}`, {
      method: 'DELETE',
    });
  },

  verify: async (deploymentId: string, hostname: string): Promise<DomainInfo> => {
    return domainFetch<DomainInfo>(`/deployments/${deploymentId}/domains/${hostname}/verify`, {
      method: 'POST',
    });
  },
};
