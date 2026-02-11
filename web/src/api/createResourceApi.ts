import { api } from './client';
import type { JsonApiResource } from './types';

/**
 * Configuration for custom actions beyond standard CRUD.
 */
export interface CustomAction {
  /** HTTP method for the action */
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  /** Path suffix after the resource ID (e.g., 'maintenance' -> /nodes/:id/maintenance) */
  path: string;
  /** Whether this action requires an ID */
  requiresId?: boolean;
}

/**
 * Options for creating a resource API.
 */
export interface CreateResourceApiOptions<CustomActions extends string = never> {
  /** The resource name (e.g., 'nodes', 'ssh_keys') */
  resourceName: string;
  /** Custom actions beyond CRUD operations */
  customActions?: Record<CustomActions, CustomAction>;
  /** Whether to support update (PATCH) - default true */
  supportsUpdate?: boolean;
  /** Whether to support delete - default true */
  supportsDelete?: boolean;
}

/**
 * Standard CRUD operations for a JSON:API resource.
 */
export interface ResourceApi<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>
> {
  /** List all resources, with optional query params (e.g. { scope: 'mine' }) */
  list: (params?: Record<string, string>) => Promise<Resource[]>;
  /** Get a single resource by ID */
  get: (id: string) => Promise<Resource>;
  /** Create a new resource */
  create: (data: CreateRequest) => Promise<Resource>;
  /** Update an existing resource */
  update: (id: string, data: UpdateRequest) => Promise<Resource>;
  /** Delete a resource */
  delete: (id: string) => Promise<void>;
}

/**
 * Creates a type-safe API client for a JSON:API resource.
 *
 * @example
 * ```ts
 * // Simple CRUD resource
 * export const nodesApi = createResourceApi<Node, CreateNodeRequest, UpdateNodeRequest>({
 *   resourceName: 'nodes',
 * });
 *
 * // Resource with custom actions
 * export const nodesApi = createResourceApi<Node, CreateNodeRequest, UpdateNodeRequest>({
 *   resourceName: 'nodes',
 *   customActions: {
 *     enterMaintenance: { method: 'POST', path: 'maintenance', requiresId: true },
 *     exitMaintenance: { method: 'DELETE', path: 'maintenance', requiresId: true },
 *   },
 * });
 * ```
 */
export function createResourceApi<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>,
  CustomActions extends string = never
>(
  options: CreateResourceApiOptions<CustomActions>
): ResourceApi<Resource, CreateRequest, UpdateRequest> &
   Record<CustomActions, (id: string) => Promise<Resource>> {

  const { resourceName, customActions, supportsUpdate = true, supportsDelete = true } = options;
  const basePath = `/${resourceName}`;

  const baseApi: ResourceApi<Resource, CreateRequest, UpdateRequest> = {
    list: async (params?: Record<string, string>) => {
      const qs = params ? '?' + new URLSearchParams(params).toString() : '';
      const response = await api.get<Resource[]>(`${basePath}${qs}`);
      return response.data;
    },

    get: async (id: string) => {
      const response = await api.get<Resource>(`${basePath}/${id}`);
      return response.data;
    },

    create: async (data: CreateRequest) => {
      const response = await api.post<Resource>(basePath, {
        data: {
          type: resourceName,
          attributes: data,
        },
      });
      return response.data;
    },

    update: async (id: string, data: UpdateRequest) => {
      if (!supportsUpdate) {
        throw new Error(`Update not supported for ${resourceName}`);
      }
      const response = await api.patch<Resource>(`${basePath}/${id}`, {
        data: {
          type: resourceName,
          id,
          attributes: data,
        },
      });
      return response.data;
    },

    delete: async (id: string) => {
      if (!supportsDelete) {
        throw new Error(`Delete not supported for ${resourceName}`);
      }
      await api.delete(`${basePath}/${id}`);
    },
  };

  // Add custom actions
  const customActionMethods: Record<string, (id: string) => Promise<Resource>> = {};

  if (customActions) {
    for (const [actionName, config] of Object.entries(customActions) as [CustomActions, CustomAction][]) {
      customActionMethods[actionName] = async (id: string) => {
        const path = config.requiresId !== false
          ? `${basePath}/${id}/${config.path}`
          : `${basePath}/${config.path}`;

        let response;
        switch (config.method) {
          case 'GET':
            response = await api.get<Resource>(path);
            break;
          case 'POST':
            response = await api.post<Resource>(path);
            break;
          case 'PATCH':
            response = await api.patch<Resource>(path);
            break;
          case 'DELETE':
            response = await api.delete<Resource>(path);
            break;
        }
        return response.data;
      };
    }
  }

  return {
    ...baseApi,
    ...customActionMethods,
  } as ResourceApi<Resource, CreateRequest, UpdateRequest> &
     Record<CustomActions, (id: string) => Promise<Resource>>;
}

/**
 * Creates a simple CRUD API without custom actions.
 * Shorthand for createResourceApi when no custom actions are needed.
 */
export function createCrudApi<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>
>(resourceName: string, options?: { supportsUpdate?: boolean; supportsDelete?: boolean }) {
  return createResourceApi<Resource, CreateRequest, UpdateRequest>({
    resourceName,
    ...options,
  });
}
