import type { JsonApiResponse, JsonApiErrorResponse, JsonApiError } from './types';

const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

export class ApiError extends Error {
  errors: JsonApiError[];
  status: number;

  constructor(response: JsonApiErrorResponse, status: number) {
    const message = response.errors[0]?.detail || response.errors[0]?.title || 'Unknown error';
    super(message);
    this.name = 'ApiError';
    this.errors = response.errors;
    this.status = status;
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

export async function apiClient<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<JsonApiResponse<T>> {
  const { body, ...restOptions } = options;

  const response = await fetch(`${BASE_URL}${endpoint}`, {
    ...restOptions,
    headers: {
      'Content-Type': 'application/vnd.api+json',
      'Accept': 'application/vnd.api+json',
      ...options.headers,
    },
    credentials: 'include',
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    let errorData: JsonApiErrorResponse;
    try {
      errorData = await response.json();
    } catch {
      errorData = {
        errors: [{
          status: String(response.status),
          title: response.statusText,
          detail: 'Failed to parse error response',
        }],
      };
    }
    throw new ApiError(errorData, response.status);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return { data: null as T };
  }

  return response.json();
}

// HTTP method helpers
export const api = {
  get: <T>(endpoint: string) => apiClient<T>(endpoint, { method: 'GET' }),

  post: <T>(endpoint: string, data?: unknown) =>
    apiClient<T>(endpoint, { method: 'POST', body: data }),

  patch: <T>(endpoint: string, data?: unknown) =>
    apiClient<T>(endpoint, { method: 'PATCH', body: data }),

  delete: <T>(endpoint: string) => apiClient<T>(endpoint, { method: 'DELETE' }),
};
