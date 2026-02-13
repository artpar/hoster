import { createResourceApi } from './createResourceApi';
import type { Invoice } from './types';

/**
 * Invoice API client with list/get and pay action.
 *
 * Endpoints:
 * - GET    /invoices          - List all invoices
 * - GET    /invoices/:id      - Get invoice by ID
 * - POST   /invoices/:id/pay  - Create Stripe payment session
 */
export const invoicesApi = createResourceApi<Invoice, never, never, 'pay'>({
  resourceName: 'invoices',
  customActions: {
    pay: { method: 'POST', path: 'pay', requiresId: true },
  },
});
