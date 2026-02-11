import { useQuery, useMutation, useQueryClient, UseQueryResult, UseMutationResult } from '@tanstack/react-query';
import type { JsonApiResource } from '@/api/types';
import type { ResourceApi } from '@/api/createResourceApi';

/**
 * Query keys for a resource.
 */
export interface ResourceQueryKeys {
  all: readonly [string];
  lists: () => readonly [string, 'list'];
  list: (filters: string) => readonly [string, 'list', { filters: string }];
  details: () => readonly [string, 'detail'];
  detail: (id: string) => readonly [string, 'detail', string];
}

/**
 * Creates query keys for a resource.
 */
export function createQueryKeys(resourceName: string): ResourceQueryKeys {
  return {
    all: [resourceName] as const,
    lists: () => [resourceName, 'list'] as const,
    list: (filters: string) => [resourceName, 'list', { filters }] as const,
    details: () => [resourceName, 'detail'] as const,
    detail: (id: string) => [resourceName, 'detail', id] as const,
  };
}

/**
 * Standard hooks returned by createResourceHooks.
 */
export interface ResourceHooks<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>
> {
  /** Query keys for cache management */
  keys: ResourceQueryKeys;
  /** Hook to list all resources, with optional query params */
  useList: (params?: Record<string, string>) => UseQueryResult<Resource[], Error>;
  /** Hook to get a single resource by ID */
  useGet: (id: string) => UseQueryResult<Resource, Error>;
  /** Hook to create a new resource */
  useCreate: () => UseMutationResult<Resource, Error, CreateRequest>;
  /** Hook to update an existing resource */
  useUpdate: () => UseMutationResult<Resource, Error, { id: string; data: UpdateRequest }>;
  /** Hook to delete a resource */
  useDelete: () => UseMutationResult<void, Error, string>;
}

/**
 * Options for creating resource hooks.
 */
export interface CreateResourceHooksOptions<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>
> {
  /** The resource name (used for query keys) */
  resourceName: string;
  /** The API client for the resource */
  api: ResourceApi<Resource, CreateRequest, UpdateRequest>;
  /** Whether the resource supports update operations */
  supportsUpdate?: boolean;
  /** Whether the resource supports delete operations */
  supportsDelete?: boolean;
}

/**
 * Creates TanStack Query hooks for a JSON:API resource.
 *
 * @example
 * ```ts
 * const nodesApi = createResourceApi<Node, CreateNodeRequest, UpdateNodeRequest>({
 *   resourceName: 'nodes',
 * });
 *
 * const { useList, useGet, useCreate, useUpdate, useDelete, keys } = createResourceHooks({
 *   resourceName: 'nodes',
 *   api: nodesApi,
 * });
 *
 * // Export individual hooks
 * export const useNodes = useList;
 * export const useNode = useGet;
 * export const useCreateNode = useCreate;
 * // etc.
 * ```
 */
export function createResourceHooks<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>
>(
  options: CreateResourceHooksOptions<Resource, CreateRequest, UpdateRequest>
): ResourceHooks<Resource, CreateRequest, UpdateRequest> {
  const { resourceName, api, supportsUpdate = true, supportsDelete = true } = options;
  const keys = createQueryKeys(resourceName);

  return {
    keys,

    useList: (params?: Record<string, string>) => {
      return useQuery({
        queryKey: [...keys.lists(), params ?? {}],
        queryFn: () => api.list(params),
      });
    },

    useGet: (id: string) => {
      return useQuery({
        queryKey: keys.detail(id),
        queryFn: () => api.get(id),
        enabled: !!id,
      });
    },

    useCreate: () => {
      const queryClient = useQueryClient();
      return useMutation({
        mutationFn: (data: CreateRequest) => api.create(data),
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: keys.lists() });
        },
      });
    },

    useUpdate: () => {
      const queryClient = useQueryClient();
      return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateRequest }) => {
          if (!supportsUpdate) {
            throw new Error(`Update not supported for ${resourceName}`);
          }
          return api.update(id, data);
        },
        onSuccess: (_, { id }) => {
          queryClient.invalidateQueries({ queryKey: keys.detail(id) });
          queryClient.invalidateQueries({ queryKey: keys.lists() });
        },
      });
    },

    useDelete: () => {
      const queryClient = useQueryClient();
      return useMutation({
        mutationFn: (id: string) => {
          if (!supportsDelete) {
            throw new Error(`Delete not supported for ${resourceName}`);
          }
          return api.delete(id);
        },
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: keys.lists() });
        },
      });
    },
  };
}

/**
 * Creates a custom action hook for a resource.
 * Use this for actions beyond standard CRUD.
 *
 * @example
 * ```ts
 * export const useEnterMaintenanceMode = createActionHook(
 *   nodeKeys,
 *   (id: string) => nodesApi.enterMaintenance(id)
 * );
 * ```
 */
export function createActionHook<T, TArg>(
  keys: ResourceQueryKeys,
  actionFn: (arg: TArg) => Promise<T>,
  options?: {
    /** Whether to invalidate detail queries for specific IDs */
    invalidateDetail?: boolean;
    /** Whether to invalidate list queries */
    invalidateList?: boolean;
  }
) {
  const { invalidateDetail = true, invalidateList = true } = options ?? {};

  return () => {
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn: actionFn,
      onSuccess: (_, arg) => {
        // Assume argument is the ID if invalidating detail and it's a string
        if (invalidateDetail && typeof arg === 'string') {
          queryClient.invalidateQueries({ queryKey: keys.detail(arg) });
        }
        if (invalidateList) {
          queryClient.invalidateQueries({ queryKey: keys.lists() });
        }
      },
    });
  };
}

/**
 * Creates a custom action hook that takes an ID parameter.
 * Convenience wrapper around createActionHook for common id-based actions.
 */
export function createIdActionHook<T>(
  keys: ResourceQueryKeys,
  actionFn: (id: string) => Promise<T>
) {
  return () => {
    const queryClient = useQueryClient();
    return useMutation({
      mutationFn: actionFn,
      onSuccess: (_, id) => {
        queryClient.invalidateQueries({ queryKey: keys.detail(id) });
        queryClient.invalidateQueries({ queryKey: keys.lists() });
      },
    });
  };
}
