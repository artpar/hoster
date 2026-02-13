import type { Invoice } from '@/api/types';
import { invoicesApi } from '@/api/invoices';
import { createResourceHooks, createIdActionHook } from './createResourceHooks';

const invoiceHooks = createResourceHooks<Invoice, never, never>({
  resourceName: 'invoices',
  api: invoicesApi,
  supportsUpdate: false,
  supportsDelete: false,
});

export const invoiceKeys = invoiceHooks.keys;
export const useInvoices = invoiceHooks.useList;
export const useInvoice = invoiceHooks.useGet;

export const usePayInvoice = createIdActionHook(
  invoiceKeys,
  invoicesApi.pay
);
