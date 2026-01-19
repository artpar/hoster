import { createResourceApi } from './createResourceApi';
import type { SSHKey, CreateSSHKeyRequest } from './types';

/**
 * SSH Key API client with limited CRUD operations.
 * SSH keys are immutable - update is not supported.
 *
 * Endpoints:
 * - GET    /ssh_keys       - List all SSH keys
 * - GET    /ssh_keys/:id   - Get SSH key by ID (fingerprint only, no private key)
 * - POST   /ssh_keys       - Create new SSH key (private key encrypted on server)
 * - DELETE /ssh_keys/:id   - Delete SSH key
 */
export const sshKeysApi = createResourceApi<
  SSHKey,
  CreateSSHKeyRequest,
  never  // No update support - SSH keys are immutable
>({
  resourceName: 'ssh_keys',
  supportsUpdate: false,
});
