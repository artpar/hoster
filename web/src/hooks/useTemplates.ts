import { templatesApi } from '@/api/templates';
import type { Template, CreateTemplateRequest } from '@/api/types';
import { createResourceHooks, createIdActionHook } from './createResourceHooks';

/**
 * TanStack Query hooks for Template resources.
 *
 * Generated from createResourceHooks factory with custom publish action.
 */
const templateHooks = createResourceHooks<Template, CreateTemplateRequest, Partial<CreateTemplateRequest>>({
  resourceName: 'templates',
  api: templatesApi,
});

// Export query keys for external cache management
export const templateKeys = templateHooks.keys;

// Export standard CRUD hooks with friendly names
export const useTemplates = templateHooks.useList;
export const useTemplate = templateHooks.useGet;
export const useCreateTemplate = templateHooks.useCreate;
export const useUpdateTemplate = templateHooks.useUpdate;
export const useDeleteTemplate = templateHooks.useDelete;

// Custom action hook for publishing
export const usePublishTemplate = createIdActionHook(
  templateKeys,
  templatesApi.publish
);
