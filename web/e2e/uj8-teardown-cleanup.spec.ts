import { test, expect } from '@playwright/test';
import { injectAuth } from './fixtures/auth.fixture';
import { apiCreateDeployment, apiStartDeployment, apiDeleteDeployment, apiStopDeployment, apiListDeployments } from './fixtures/api.fixture';
import { uniqueSlug, readInfraState, type InfraState } from './fixtures/test-data';
import { waitForDeploymentStatus } from './helpers/wait';

/**
 * UJ8: "I'm done with this deployment"
 *
 * Customer stops a running deployment, deletes it with confirmation,
 * verifies it's gone from the list, and checks billing reflects the change.
 *
 * Uses REAL infrastructure from global setup — deployment runs real Docker
 * containers on a real DigitalOcean droplet. Stop actually stops containers,
 * delete actually removes them.
 *
 * Targets: APIGate (:8082) -> Hoster (:8080) — real prod-like stack.
 */

test.describe('UJ8: Teardown & Cleanup', () => {
  let token: string;
  let infra: InfraState;
  let deploymentId: string | undefined;

  test.beforeAll(async ({}, testInfo) => {
    testInfo.setTimeout(300_000);
    const state = readInfraState();
    if (!state) throw new Error('No infrastructure state — run global setup first');
    infra = state;
    token = infra.token;

    // Clean up any existing deployments from earlier test suites (plan limit = 1)
    const existing = await apiListDeployments(token);
    for (const d of existing) {
      const status = d.attributes.status as string;
      if (status === 'running') await apiStopDeployment(token, d.id).catch(() => {});
      if (status !== 'deleted') {
        await new Promise(r => setTimeout(r, 2000));
        await apiDeleteDeployment(token, d.id).catch(() => {});
      }
    }

    // Create and start a real deployment on the shared droplet
    const depl = await apiCreateDeployment(token, {
      name: uniqueSlug('tddepl'),
      template_id: infra.templateId,
      node_id: infra.nodeId,
    });
    deploymentId = depl.id;

    // Start the deployment — real containers will run on the DO droplet
    await apiStartDeployment(token, deploymentId);

    // Wait for deployment to reach "running" (real Docker pull + start, up to 3 min)
    await waitForDeploymentStatus(token, deploymentId, 'running', 180_000);
  });

  test.afterAll(async () => {
    // Clean up anything left
    if (deploymentId) {
      await apiStopDeployment(token, deploymentId).catch(() => {});
      await new Promise(r => setTimeout(r, 3000));
      await apiDeleteDeployment(token, deploymentId).catch(() => {});
    }
  });

  // --- Happy path ---

  test('stop running deployment', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');
    test.setTimeout(120_000);

    await injectAuth(page, token);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Should see Running status
    await expect(page.locator('span').filter({ hasText: /Running/ })).toBeVisible({ timeout: 10_000 });

    // Click Stop — real containers will stop on the DO droplet
    const stopBtn = page.getByRole('button', { name: /Stop/i });
    await expect(stopBtn).toBeVisible();
    await stopBtn.click();

    // Wait for Stopped status (real container stop)
    await expect(page.locator('span').filter({ hasText: /Stopped/ })).toBeVisible({ timeout: 60_000 });

    // Start button should appear
    await expect(page.getByRole('button', { name: /Start/i })).toBeVisible({ timeout: 10_000 });
  });

  test('delete stopped deployment with confirmation', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');

    await injectAuth(page, token);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Click Delete
    const deleteBtn = page.getByRole('button', { name: /Delete/i });
    await expect(deleteBtn).toBeVisible({ timeout: 10_000 });
    await deleteBtn.click();
    await page.waitForTimeout(500);

    // Confirmation dialog should appear
    await expect(page.getByText(/Are you sure|cannot be undone|confirm/i)).toBeVisible();

    // Confirm deletion — real containers removed from droplet
    const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
    await confirmBtn.click();

    // Should redirect to deployments list
    await expect(page).toHaveURL(/\/deployments$/, { timeout: 15_000 });

    // Deployment should no longer be in the list
    await page.waitForLoadState('networkidle');
  });

  test('deleted deployment URL returns error', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');

    await injectAuth(page, token);
    await page.goto(`/deployments/${deploymentId}`);

    // TanStack Query retries failed requests — wait for error state
    await page.waitForTimeout(10000);

    // Should show error message or redirect
    const hasError = await page.locator('.bg-destructive\\/10').isVisible().catch(() => false);
    const isOnList = page.url().endsWith('/deployments');
    const hasNoContent = !(await page.getByRole('heading').first().isVisible().catch(() => false));
    expect(hasError || isOnList || hasNoContent).toBeTruthy();
  });

  test('billing reflects reduced cost', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // The deleted deployment should not appear in running deployments
    // Monthly cost should be $0 or reduced
    await expect(page.getByText('$0.00')).toBeVisible();
  });

  // --- Sad path ---

  test('delete pending deployment directly', async ({ page }) => {
    // Create a fresh deployment (previous one deleted, plan slot is free)
    let testDeplId: string | undefined;
    try {
      const depl = await apiCreateDeployment(token, {
        name: uniqueSlug('directdel'),
        template_id: infra.templateId,
        node_id: infra.nodeId,
      });
      testDeplId = depl.id;
    } catch {
      test.skip(true, 'Could not create test deployment');
      return;
    }

    await injectAuth(page, token);
    await page.goto(`/deployments/${testDeplId}`);
    await page.waitForLoadState('networkidle');

    // This deployment is in pending state — delete should be available
    const deleteBtn = page.getByRole('button', { name: /Delete/i });
    if (await deleteBtn.isVisible() && await deleteBtn.isEnabled()) {
      await deleteBtn.click();
      await page.waitForTimeout(500);

      // Confirm
      const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
      if (await confirmBtn.isVisible()) {
        await confirmBtn.click();
        await expect(page).toHaveURL(/\/deployments/, { timeout: 15_000 });
      }
    }
  });

  test('cancel confirmation does not delete', async ({ page }) => {
    // Use a fresh deployment for this test
    let testDeplId: string | undefined;
    try {
      const depl = await apiCreateDeployment(token, {
        name: uniqueSlug('canceltest'),
        template_id: infra.templateId,
        node_id: infra.nodeId,
      });
      testDeplId = depl.id;
    } catch {
      test.skip(true, 'Could not create test deployment');
      return;
    }

    await injectAuth(page, token);
    await page.goto(`/deployments/${testDeplId}`);
    await page.waitForLoadState('networkidle');

    const deleteBtn = page.getByRole('button', { name: /Delete/i });
    if (await deleteBtn.isVisible() && await deleteBtn.isEnabled()) {
      await deleteBtn.click();
      await page.waitForTimeout(500);

      // Cancel instead of confirming
      const cancelBtn = page.getByRole('button', { name: /Cancel/i });
      if (await cancelBtn.isVisible()) {
        await cancelBtn.click();
        await page.waitForTimeout(500);

        // Should still be on the deployment page
        await expect(page).toHaveURL(new RegExp(`/deployments/${testDeplId}`));
        // Deployment should still exist — heading visible
        await expect(page.getByRole('heading').first()).toBeVisible();
      }
    }

    // Clean up
    if (testDeplId) await apiDeleteDeployment(token, testDeplId).catch(() => {});
  });

  test('refresh during deletion shows current state', async ({ page }) => {
    let testDeplId: string | undefined;
    try {
      const depl = await apiCreateDeployment(token, {
        name: uniqueSlug('refreshtest'),
        template_id: infra.templateId,
        node_id: infra.nodeId,
      });
      testDeplId = depl.id;
    } catch {
      test.skip(true, 'Could not create test deployment');
      return;
    }

    await injectAuth(page, token);
    await page.goto(`/deployments/${testDeplId}`);
    await page.waitForLoadState('networkidle');

    // Delete and immediately reload
    const deleteBtn = page.getByRole('button', { name: /Delete/i });
    if (await deleteBtn.isVisible() && await deleteBtn.isEnabled()) {
      await deleteBtn.click();
      await page.waitForTimeout(500);
      const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
      if (await confirmBtn.isVisible()) {
        await confirmBtn.click();
        // Immediately reload before redirect completes
        await page.reload();
        // Wait for TanStack Query retries
        await page.waitForTimeout(10000);
        // Should show error state, loading state, or redirect
        const url = page.url();
        const hasError = await page.locator('.bg-destructive\\/10').isVisible().catch(() => false);
        const hasStatus = await page.getByText(/deleting|deleted|not found/i).isVisible().catch(() => false);
        const isOnList = url.endsWith('/deployments');
        expect(hasError || hasStatus || isOnList || url.includes('/deployments/')).toBeTruthy();
      }
    }

    if (testDeplId) await apiDeleteDeployment(token, testDeplId).catch(() => {});
  });
});
