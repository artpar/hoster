/**
 * Direct API helpers for test data setup/teardown.
 * All requests go through APIGate (localhost:8082) — the real stack.
 * Uses JSON:API format (application/vnd.api+json).
 */

const BASE = 'http://localhost:8082/api/v1';

interface JsonApiPayload {
  data: {
    type: string;
    attributes: Record<string, unknown>;
  };
}

function jsonApiBody(type: string, attributes: Record<string, unknown>): JsonApiPayload {
  return { data: { type, attributes } };
}

async function apiRequest(
  method: string,
  path: string,
  token: string,
  body?: unknown,
): Promise<Record<string, unknown>> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/vnd.api+json',
    Accept: 'application/vnd.api+json',
    Authorization: `Bearer ${token}`,
  };
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API ${method} ${path} failed (${res.status}): ${text}`);
  }
  if (res.status === 204) return {};
  return res.json();
}

// --- Templates ---

export async function apiCreateTemplate(
  token: string,
  attrs: {
    name: string;
    slug: string;
    description: string;
    version: string;
    compose_spec: string;
    category?: string;
    price_monthly_cents?: number;
  },
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('POST', '/templates', token, jsonApiBody('templates', attrs));
  const d = (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
  return d;
}

export async function apiPublishTemplate(token: string, id: string): Promise<void> {
  await apiRequest('POST', `/templates/${id}/publish`, token);
}

export async function apiDeleteTemplate(token: string, id: string): Promise<void> {
  await apiRequest('DELETE', `/templates/${id}`, token).catch(() => {});
}

// --- SSH Keys ---

export async function apiCreateSSHKey(
  token: string,
  attrs: { name: string; private_key: string },
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('POST', '/ssh_keys', token, jsonApiBody('ssh_keys', attrs));
  const d = (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
  return d;
}

export async function apiDeleteSSHKey(token: string, id: string): Promise<void> {
  await apiRequest('DELETE', `/ssh_keys/${id}`, token).catch(() => {});
}

// --- Nodes ---

export async function apiCreateNode(
  token: string,
  attrs: {
    name: string;
    ssh_host: string;
    ssh_port: number;
    ssh_user: string;
    ssh_key_id: string;
  },
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('POST', '/nodes', token, jsonApiBody('nodes', attrs));
  const d = (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
  return d;
}

export async function apiDeleteNode(token: string, id: string): Promise<void> {
  await apiRequest('DELETE', `/nodes/${id}`, token).catch(() => {});
}

// --- Deployments ---

export async function apiCreateDeployment(
  token: string,
  attrs: {
    name: string;
    template_id: string;
    node_id: string;
    custom_domain?: string;
    environment_variables?: Record<string, string>;
  },
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('POST', '/deployments', token, jsonApiBody('deployments', attrs));
  const d = (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
  return d;
}

export async function apiStartDeployment(token: string, id: string): Promise<void> {
  await apiRequest('POST', `/deployments/${id}/start`, token);
}

export async function apiStopDeployment(token: string, id: string): Promise<void> {
  await apiRequest('POST', `/deployments/${id}/stop`, token);
}

export async function apiDeleteDeployment(token: string, id: string): Promise<void> {
  await apiRequest('DELETE', `/deployments/${id}`, token).catch(() => {});
}

// --- Cloud Credentials ---

export async function apiCreateCloudCredential(
  token: string,
  attrs: { name: string; provider: string; api_key: string; default_region?: string },
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  // Backend expects `credentials` as a JSON-encoded string (encrypted at rest)
  const credentials = attrs.provider === 'aws'
    ? JSON.stringify({ access_key_id: attrs.api_key, secret_access_key: 'test-secret' })
    : JSON.stringify({ api_token: attrs.api_key });
  const res = await apiRequest('POST', '/cloud_credentials', token, jsonApiBody('cloud_credentials', {
    name: attrs.name,
    provider: attrs.provider,
    credentials,
    default_region: attrs.default_region,
  }));
  const d = (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
  return d;
}

export async function apiDeleteCloudCredential(token: string, id: string): Promise<void> {
  await apiRequest('DELETE', `/cloud_credentials/${id}`, token).catch(() => {});
}

// --- Cloud Provisions ---

export async function apiCreateCloudProvision(
  token: string,
  attrs: {
    credential_id: string;
    instance_name: string;
    region: string;
    size: string;
  },
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('POST', '/cloud_provisions', token, jsonApiBody('cloud_provisions', attrs));
  const d = (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
  return d;
}

export async function apiGetCloudProvision(
  token: string,
  id: string,
): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('GET', `/cloud_provisions/${id}`, token);
  return (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
}

export async function apiDestroyCloudProvision(token: string, id: string): Promise<void> {
  await apiRequest('POST', `/cloud_provisions/${id}/transition/destroying`, token);
}

// --- Listing helpers ---

export async function apiListTemplates(token: string): Promise<Array<{ id: string; attributes: Record<string, unknown> }>> {
  const res = await apiRequest('GET', '/templates', token);
  return ((res as { data: unknown[] }).data ?? []) as Array<{ id: string; attributes: Record<string, unknown> }>;
}

export async function apiListDeployments(token: string): Promise<Array<{ id: string; attributes: Record<string, unknown> }>> {
  const res = await apiRequest('GET', '/deployments', token);
  return ((res as { data: unknown[] }).data ?? []) as Array<{ id: string; attributes: Record<string, unknown> }>;
}

export async function apiGetDeployment(token: string, id: string): Promise<{ id: string; attributes: Record<string, unknown> }> {
  const res = await apiRequest('GET', `/deployments/${id}`, token);
  return (res as { data: { id: string; attributes: Record<string, unknown> } }).data;
}
