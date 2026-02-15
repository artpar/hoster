import { type Page, expect } from '@playwright/test';

/**
 * Wait until a status badge containing the expected text appears on the current page.
 */
export async function waitForStatus(
  page: Page,
  status: string,
  timeout = 30_000,
): Promise<void> {
  await expect(page.getByText(status, { exact: false })).toBeVisible({ timeout });
}

/**
 * Wait for a deployment to reach the target status by polling the deployment detail page.
 * Navigates to the deployment detail if not already there, then reloads periodically.
 */
export async function waitForDeploymentStatusOnPage(
  page: Page,
  deploymentId: string,
  targetStatus: string,
  timeout = 180_000,
  interval = 5_000,
): Promise<void> {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    const badge = page.locator('span').filter({ hasText: new RegExp(targetStatus, 'i') });
    if (await badge.isVisible().catch(() => false)) return;

    // Bail early on failed
    if (targetStatus.toLowerCase() !== 'failed') {
      const failedBadge = page.locator('span').filter({ hasText: /Failed/i });
      if (await failedBadge.isVisible().catch(() => false)) {
        throw new Error(`Deployment ${deploymentId} entered "failed" state while waiting for "${targetStatus}"`);
      }
    }

    await new Promise(r => setTimeout(r, interval));
  }
  throw new Error(`Deployment ${deploymentId} did not reach status "${targetStatus}" within ${timeout}ms`);
}

/**
 * Wait for a cloud provision to reach the target status by polling the cloud servers page.
 * Looks for the provision by instance name and checks its status badge.
 */
export async function waitForProvisionStatusOnPage(
  page: Page,
  targetStatus: string,
  timeout = 300_000,
  interval = 10_000,
): Promise<void> {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    await page.goto('/nodes/cloud');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    const badge = page.locator('span').filter({ hasText: new RegExp(targetStatus, 'i') });
    if (await badge.isVisible().catch(() => false)) return;

    // Bail early on failed (unless we're waiting for failed)
    if (targetStatus.toLowerCase() !== 'failed') {
      const failedBadge = page.locator('span').filter({ hasText: /Failed/i });
      if (await failedBadge.isVisible().catch(() => false)) {
        throw new Error(`Provision entered "failed" state while waiting for "${targetStatus}"`);
      }
    }

    await new Promise(r => setTimeout(r, interval));
  }
  throw new Error(`Provision did not reach status "${targetStatus}" within ${timeout}ms`);
}
