import { sshKeysApi } from '@/api/ssh-keys';
import type { SSHKey, CreateSSHKeyRequest } from '@/api/types';
import { createResourceHooks } from './createResourceHooks';

/**
 * TanStack Query hooks for SSH Key resources.
 *
 * Generated from createResourceHooks factory.
 * Note: SSH keys are immutable - update is not supported.
 */
const sshKeyHooks = createResourceHooks<SSHKey, CreateSSHKeyRequest, never>({
  resourceName: 'ssh_keys',
  api: sshKeysApi,
  supportsUpdate: false,
});

// Export query keys for external cache management
export const sshKeyKeys = sshKeyHooks.keys;

// Export standard CRUD hooks with friendly names
export const useSSHKeys = sshKeyHooks.useList;
export const useSSHKey = sshKeyHooks.useGet;
export const useCreateSSHKey = sshKeyHooks.useCreate;
export const useDeleteSSHKey = sshKeyHooks.useDelete;

// Note: useUpdateSSHKey is intentionally not exported
// SSH keys are immutable - delete and create a new one instead
