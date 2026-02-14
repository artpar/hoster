/**
 * Global teardown for E2E tests.
 *
 * Destroys the shared DigitalOcean droplet and cleans up all resources.
 * Reads infrastructure state from .e2e-infra.json.
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BASE = 'http://localhost:8082';
const API = `${BASE}/api/v1`;
const INFRA_STATE_PATH = path.join(__dirname, '.e2e-infra.json');

async function apiRequest(method: string, url: string, token: string, body?: unknown): Promise<Record<string, unknown>> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/vnd.api+json',
    Accept: 'application/vnd.api+json',
    Authorization: `Bearer ${token}`,
  };
  const res = await fetch(url, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Global teardown: ${method} ${url} failed (${res.status}): ${text}`);
  }
  if (res.status === 204) return {};
  return res.json();
}

async function globalTeardown() {
  console.log('[global-teardown] Starting cleanup...');

  if (!fs.existsSync(INFRA_STATE_PATH)) {
    console.log('[global-teardown] No infra state file found — nothing to clean up');
    return;
  }

  const state = JSON.parse(fs.readFileSync(INFRA_STATE_PATH, 'utf-8'));
  const { token, provisionId, templateId, credentialId } = state;

  if (!token) {
    console.log('[global-teardown] No token in state — skipping');
    return;
  }

  // 1. Delete any remaining deployments
  try {
    const deplRes = await apiRequest('GET', `${API}/deployments`, token);
    const deployments = ((deplRes as any).data ?? []) as Array<{ id: string; attributes: Record<string, unknown> }>;
    for (const depl of deployments) {
      const status = depl.attributes.status as string;
      console.log(`[global-teardown] Cleaning deployment ${depl.id} (status: ${status})`);
      if (status === 'running' || status === 'starting') {
        await apiRequest('POST', `${API}/deployments/${depl.id}/stop`, token).catch(() => {});
        // Wait briefly for stop
        await new Promise(r => setTimeout(r, 5000));
      }
      await apiRequest('DELETE', `${API}/deployments/${depl.id}`, token).catch(() => {});
    }
  } catch (err) {
    console.warn('[global-teardown] Error cleaning deployments:', err);
  }

  // 2. Destroy the provision (droplet)
  if (provisionId) {
    try {
      console.log(`[global-teardown] Destroying provision: ${provisionId}`);
      await apiRequest('POST', `${API}/cloud_provisions/${provisionId}/transition/destroying`, token);

      // Poll until destroyed (timeout: 3 min)
      const deadline = Date.now() + 3 * 60 * 1000;
      while (Date.now() < deadline) {
        const res = await apiRequest('GET', `${API}/cloud_provisions/${provisionId}`, token);
        const status = (res as any).data?.attributes?.status;
        console.log(`[global-teardown] Provision status: ${status}`);
        if (status === 'destroyed') break;
        await new Promise(r => setTimeout(r, 5000));
      }
    } catch (err) {
      console.warn('[global-teardown] Error destroying provision:', err);
    }
  }

  // 3. Clean up template (best-effort)
  if (templateId) {
    await apiRequest('DELETE', `${API}/templates/${templateId}`, token).catch(() => {});
  }

  // 4. Clean up credential (best-effort)
  if (credentialId) {
    await apiRequest('DELETE', `${API}/cloud_credentials/${credentialId}`, token).catch(() => {});
  }

  // 5. Remove state file
  fs.unlinkSync(INFRA_STATE_PATH);
  console.log('[global-teardown] Cleanup complete.');
}

export default globalTeardown;
