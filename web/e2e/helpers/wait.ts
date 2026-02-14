import { type Page, expect } from '@playwright/test';

const BASE = 'http://localhost:8082/api/v1';

/**
 * Poll the page until a status badge containing the expected text appears.
 * Useful for waiting on state transitions (e.g., pending -> running).
 */
export async function waitForStatus(
  page: Page,
  status: string,
  timeout = 30_000,
): Promise<void> {
  await expect(page.getByText(status, { exact: false })).toBeVisible({ timeout });
}

/**
 * Wait for a deployment status via the API.
 * Polls the deployment endpoint until the status matches or timeout.
 */
export async function waitForDeploymentStatus(
  token: string,
  deploymentId: string,
  targetStatus: string,
  timeout = 180_000,
  interval = 3_000,
): Promise<void> {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const res = await fetch(`${BASE}/deployments/${deploymentId}`, {
      headers: {
        Accept: 'application/vnd.api+json',
        Authorization: `Bearer ${token}`,
      },
    });
    if (res.ok) {
      const data = await res.json();
      const status = data?.data?.attributes?.status;
      if (status === targetStatus) return;
      if (status === 'failed') {
        throw new Error(`Deployment ${deploymentId} entered "failed" state while waiting for "${targetStatus}"`);
      }
    }
    await new Promise((r) => setTimeout(r, interval));
  }
  throw new Error(`Deployment ${deploymentId} did not reach status "${targetStatus}" within ${timeout}ms`);
}

/**
 * Wait for a cloud provision to reach the target status via the API.
 * Polls the provision endpoint until the status matches or timeout.
 */
export async function waitForProvisionStatus(
  token: string,
  provisionId: string,
  targetStatus: string,
  timeout = 300_000,
  interval = 5_000,
): Promise<void> {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const res = await fetch(`${BASE}/cloud_provisions/${provisionId}`, {
      headers: {
        Accept: 'application/vnd.api+json',
        Authorization: `Bearer ${token}`,
      },
    });
    if (res.ok) {
      const data = await res.json();
      const status = data?.data?.attributes?.status;
      const step = data?.data?.attributes?.current_step || '';
      console.log(`[wait] Provision ${provisionId}: status=${status}, step=${step}`);
      if (status === targetStatus) return;
      if (status === 'failed' && targetStatus !== 'failed') {
        const errMsg = data?.data?.attributes?.error_message || 'unknown';
        throw new Error(`Provision ${provisionId} entered "failed" state: ${errMsg}`);
      }
    }
    await new Promise((r) => setTimeout(r, interval));
  }
  throw new Error(`Provision ${provisionId} did not reach status "${targetStatus}" within ${timeout}ms`);
}
