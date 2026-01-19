import { createResourceApi } from './createResourceApi';
import type { Node, CreateNodeRequest, UpdateNodeRequest } from './types';

/**
 * Node API client with CRUD operations and custom maintenance actions.
 *
 * Endpoints:
 * - GET    /nodes          - List all nodes
 * - GET    /nodes/:id      - Get node by ID
 * - POST   /nodes          - Create new node
 * - PATCH  /nodes/:id      - Update node
 * - DELETE /nodes/:id      - Delete node
 * - POST   /nodes/:id/maintenance   - Enter maintenance mode
 * - DELETE /nodes/:id/maintenance   - Exit maintenance mode
 */
export const nodesApi = createResourceApi<
  Node,
  CreateNodeRequest,
  UpdateNodeRequest,
  'enterMaintenance' | 'exitMaintenance'
>({
  resourceName: 'nodes',
  customActions: {
    enterMaintenance: { method: 'POST', path: 'maintenance', requiresId: true },
    exitMaintenance: { method: 'DELETE', path: 'maintenance', requiresId: true },
  },
});
