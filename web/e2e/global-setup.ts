/**
 * Global setup for E2E tests.
 *
 * Provisions a real DigitalOcean droplet shared by all test suites.
 * Writes infrastructure state to .e2e-infra.json for tests to read.
 *
 * Requires:
 *   - APIGate (:8082) and Hoster (:8080) running
 *   - TEST_DO_API_KEY environment variable or hardcoded in test-data.ts
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { TEST_DO_API_KEY, TEST_PASSWORD, TEST_TEMPLATE_COMPOSE, uniqueEmail, uniqueName, uniqueSlug } from './fixtures/test-data';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BASE = 'http://localhost:8082';
const API = `${BASE}/api/v1`;
const INFRA_STATE_PATH = path.join(__dirname, '.e2e-infra.json');

interface InfraState {
  token: string;
  email: string;
  nodeId: string;
  templateId: string;
  provisionId: string;
  credentialId: string;
  sshKeyId: string;
  dropletIp: string;
}

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
    throw new Error(`Global setup: ${method} ${url} failed (${res.status}): ${text}`);
  }
  if (res.status === 204) return {};
  return res.json();
}

async function globalSetup() {
  console.log('[global-setup] Starting real infrastructure provisioning...');

  // 1. Sign up test user
  const email = uniqueEmail();
  console.log(`[global-setup] Registering user: ${email}`);
  const signupRes = await fetch(`${BASE}/mod/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: TEST_PASSWORD, name: email.split('@')[0] }),
  });
  if (!signupRes.ok) {
    throw new Error(`[global-setup] Signup failed: ${await signupRes.text()}`);
  }
  const signupData = await signupRes.json();
  const token = signupData.token;
  if (!token) throw new Error('[global-setup] No token from signup');
  console.log('[global-setup] User registered successfully');

  // 2. Create cloud credential with real DO API key
  const credName = uniqueName('e2e-cred');
  console.log(`[global-setup] Creating cloud credential: ${credName}`);
  const credRes = await apiRequest('POST', `${API}/cloud_credentials`, token, {
    data: {
      type: 'cloud_credentials',
      attributes: {
        name: credName,
        provider: 'digitalocean',
        credentials: JSON.stringify({ api_token: TEST_DO_API_KEY }),
      },
    },
  });
  const credentialId = (credRes as any).data.id;
  console.log(`[global-setup] Credential created: ${credentialId}`);

  // 3. Create cloud provision (real DO droplet)
  const instanceName = `e2e-${Date.now()}`;
  console.log(`[global-setup] Creating cloud provision: ${instanceName} (sfo3, s-1vcpu-1gb)`);
  const provRes = await apiRequest('POST', `${API}/cloud_provisions`, token, {
    data: {
      type: 'cloud_provisions',
      attributes: {
        credential_id: credentialId,
        instance_name: instanceName,
        region: 'sfo3',
        size: 's-1vcpu-1gb',
      },
    },
  });
  const provisionId = (provRes as any).data.id;
  console.log(`[global-setup] Provision created: ${provisionId}`);

  // 4. Poll until provision is ready (timeout: 5 min)
  const deadline = Date.now() + 5 * 60 * 1000;
  let provisionData: any;
  while (Date.now() < deadline) {
    const res = await apiRequest('GET', `${API}/cloud_provisions/${provisionId}`, token);
    provisionData = (res as any).data;
    const status = provisionData.attributes.status;
    const step = provisionData.attributes.current_step || '';
    console.log(`[global-setup] Provision status: ${status} (step: ${step})`);

    if (status === 'ready') break;
    if (status === 'failed') {
      throw new Error(`[global-setup] Provision failed: ${provisionData.attributes.error_message}`);
    }

    await new Promise(r => setTimeout(r, 5000));
  }

  if (!provisionData || provisionData.attributes.status !== 'ready') {
    throw new Error('[global-setup] Provision did not reach ready state within 5 minutes');
  }

  const nodeId = provisionData.attributes.node_id;
  const sshKeyId = provisionData.attributes.ssh_key_id;
  const dropletIp = provisionData.attributes.public_ip;
  console.log(`[global-setup] Provision ready: node=${nodeId}, ip=${dropletIp}`);

  // 5. Create and publish a template (nginx:alpine)
  const tmplName = uniqueName('e2e-tmpl');
  const tmplSlug = uniqueSlug('e2e-tmpl');
  console.log(`[global-setup] Creating template: ${tmplName}`);
  const tmplRes = await apiRequest('POST', `${API}/templates`, token, {
    data: {
      type: 'templates',
      attributes: {
        name: tmplName,
        slug: tmplSlug,
        description: 'E2E test template - nginx:alpine',
        version: '1.0.0',
        compose_spec: TEST_TEMPLATE_COMPOSE,
        category: 'web',
        price_monthly_cents: 500,
      },
    },
  });
  const templateId = (tmplRes as any).data.id;
  await apiRequest('POST', `${API}/templates/${templateId}/publish`, token);
  console.log(`[global-setup] Template published: ${templateId}`);

  // 6. Write state to .e2e-infra.json
  const state: InfraState = {
    token,
    email,
    nodeId,
    templateId,
    provisionId,
    credentialId,
    sshKeyId,
    dropletIp,
  };
  fs.writeFileSync(INFRA_STATE_PATH, JSON.stringify(state, null, 2));
  console.log(`[global-setup] Infrastructure state written to ${INFRA_STATE_PATH}`);
  console.log('[global-setup] Done. Ready for tests.');
}

export default globalSetup;
