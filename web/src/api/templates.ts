import { createResourceApi } from './createResourceApi';
import type { Template, CreateTemplateRequest } from './types';

/**
 * Template API client with CRUD operations and publish action.
 *
 * Endpoints:
 * - GET    /templates          - List all templates
 * - GET    /templates/:id      - Get template by ID
 * - POST   /templates          - Create new template
 * - PATCH  /templates/:id      - Update template
 * - DELETE /templates/:id      - Delete template
 * - POST   /templates/:id/publish - Publish template
 */
export const templatesApi = createResourceApi<
  Template,
  CreateTemplateRequest,
  Partial<CreateTemplateRequest>,
  'publish'
>({
  resourceName: 'templates',
  customActions: {
    publish: { method: 'POST', path: 'publish', requiresId: true },
  },
});
