import { api } from './client';
import type {
  HealthAttributes,
  StatsAttributes,
  LogsAttributes,
  EventsAttributes,
} from './types';

interface HealthResponse {
  type: 'deployment-health';
  id: string;
  attributes: HealthAttributes;
}

interface StatsResponse {
  type: 'deployment-stats';
  id: string;
  attributes: StatsAttributes;
}

interface LogsResponse {
  type: 'deployment-logs';
  id: string;
  attributes: LogsAttributes;
}

interface EventsResponse {
  type: 'deployment-events';
  id: string;
  attributes: EventsAttributes;
}

export interface LogsQueryParams {
  tail?: number;
  container?: string;
  since?: string;
}

export interface EventsQueryParams {
  limit?: number;
  type?: string;
}

export const monitoringApi = {
  getHealth: async (deploymentId: string) => {
    const response = await api.get<HealthResponse>(
      `/deployments/${deploymentId}/monitoring/health`
    );
    return response.data.attributes;
  },

  getStats: async (deploymentId: string) => {
    const response = await api.get<StatsResponse>(
      `/deployments/${deploymentId}/monitoring/stats`
    );
    return response.data.attributes;
  },

  getLogs: async (deploymentId: string, params?: LogsQueryParams) => {
    const searchParams = new URLSearchParams();
    if (params?.tail) searchParams.set('tail', String(params.tail));
    if (params?.container) searchParams.set('container', params.container);
    if (params?.since) searchParams.set('since', params.since);

    const query = searchParams.toString();
    const endpoint = `/deployments/${deploymentId}/monitoring/logs${query ? `?${query}` : ''}`;

    const response = await api.get<LogsResponse>(endpoint);
    return response.data.attributes;
  },

  getEvents: async (deploymentId: string, params?: EventsQueryParams) => {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.type) searchParams.set('type', params.type);

    const query = searchParams.toString();
    const endpoint = `/deployments/${deploymentId}/monitoring/events${query ? `?${query}` : ''}`;

    const response = await api.get<EventsResponse>(endpoint);
    return response.data.attributes;
  },
};
