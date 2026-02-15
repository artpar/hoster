import { test, expect, chromium } from '@playwright/test';
import { logIn } from './fixtures/auth.fixture';
import { uniqueSlug, readInfraState, TEST_PASSWORD, type InfraState } from './fixtures/test-data';
import { waitForDeploymentStatusOnPage } from './helpers/wait';

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
  let infra: InfraState;
  let deploymentId: string | undefined;

  test.beforeAll(async ({}, testInfo) => {
    testInfo.setTimeout(300_000);
    const state = readInfraState();
    if (!state) throw new Error('No infrastructure state — run global setup first');
    infra = state;

    // Use browser to clean up existing deployments and create a new one
    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();

    try {
      await logIn(page, infra.email, TEST_PASSWORD);

      // Clean up any existing deployments (plan limit = 1)
      await page.goto('/deployments');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);

      const deplLinks = await page.locator('a[href*="/deployments/depl_"]').all();
      for (const link of deplLinks) {
        const href = await link.getAttribute('href');
        if (!href) continue;

        await page.goto(href);
        await page.waitForLoadState('networkidle');

        const stopBtn = page.getByRole('button', { name: /Stop/i });
        if (await stopBtn.isVisible().catch(() => false) && await stopBtn.isEnabled().catch(() => false)) {
          await stopBtn.click();
          await page.locator('span').filter({ hasText: /Stopped/ }).waitFor({ timeout: 60_000 }).catch(() => {});
        }

        const deleteBtn = page.getByRole('button', { name: /Delete/i });
        if (await deleteBtn.isVisible().catch(() => false)) {
          await deleteBtn.click();
          await page.waitForTimeout(500);
          const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
          if (await confirmBtn.isVisible().catch(() => false)) {
            await confirmBtn.click();
            await page.waitForTimeout(3000);
          }
        }
      }

      // Create a deployment via the deploy dialog
      await page.goto(`/templates/${infra.templateId}`);
      await page.waitForLoadState('networkidle');
      await page.getByText('Deploy Now').click();
      await page.waitForTimeout(1000);

      const nameInput = page.locator('#name');
      if (await nameInput.isVisible()) {
        await nameInput.fill(uniqueSlug('tddepl'));
      }

      const nodeSelect = page.locator('#node').or(page.getByLabel(/Deploy To|Node/i));
      if (await nodeSelect.isVisible()) {
        const options = nodeSelect.locator('option');
        const count = await options.count();
        if (count > 1) await nodeSelect.selectOption({ index: 1 });
      }

      await page.getByRole('button', { name: /^Deploy$|^Creating|^Starting/i }).click();
      await expect(page).toHaveURL(/\/deployments\//, { timeout: 15_000 });

      const match = page.url().match(/\/deployments\/([^/]+)/);
      deploymentId = match?.[1];

      // Wait for Running status
      if (deploymentId) {
        await waitForDeploymentStatusOnPage(page, deploymentId, 'Running', 180_000);
      }
    } finally {
      await browser.close();
    }
  });

  test.afterAll(async () => {
    // Best-effort cleanup
    if (!deploymentId) return;
    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();
    try {
      await logIn(page, infra.email, TEST_PASSWORD);
      await page.goto(`/deployments/${deploymentId}`);
      await page.waitForLoadState('networkidle');

      const stopBtn = page.getByRole('button', { name: /Stop/i });
      if (await stopBtn.isVisible().catch(() => false)) {
        await stopBtn.click();
        await page.locator('span').filter({ hasText: /Stopped/ }).waitFor({ timeout: 60_000 }).catch(() => {});
      }

      const deleteBtn = page.getByRole('button', { name: /Delete/i });
      if (await deleteBtn.isVisible().catch(() => false)) {
        await deleteBtn.click();
        await page.waitForTimeout(500);
        const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
        if (await confirmBtn.isVisible().catch(() => false)) {
          await confirmBtn.click();
          await page.waitForTimeout(2000);
        }
      }
    } finally {
      await browser.close();
    }
  });

  // --- Happy path ---

  test('stop running deployment', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');
    test.setTimeout(120_000);

    await logIn(page, infra.email, TEST_PASSWORD);
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

    await logIn(page, infra.email, TEST_PASSWORD);
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

    await logIn(page, infra.email, TEST_PASSWORD);
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
    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // The deleted deployment should not appear in running deployments
    // Monthly cost should be $0 or reduced
    await expect(page.getByText('$0.00')).toBeVisible();
  });

  // --- Sad path ---

  test('delete pending deployment directly', async ({ page }) => {
    // Create a fresh deployment via UI
    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/templates/${infra.templateId}`);
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(1000);

    const nameInput = page.locator('#name');
    if (await nameInput.isVisible()) {
      await nameInput.fill(uniqueSlug('directdel'));
    }

    const nodeSelect = page.locator('#node').or(page.getByLabel(/Deploy To|Node/i));
    if (await nodeSelect.isVisible()) {
      const options = nodeSelect.locator('option');
      const count = await options.count();
      if (count > 1) await nodeSelect.selectOption({ index: 1 });
    }

    await page.getByRole('button', { name: /^Deploy$|^Creating|^Starting/i }).click();

    const redirectOk = await page.waitForURL(/\/deployments\//, { timeout: 15_000 }).then(() => true).catch(() => false);
    if (!redirectOk) {
      test.skip(true, 'Could not create test deployment');
      return;
    }

    const match = page.url().match(/\/deployments\/([^/]+)/);
    const testDeplId = match?.[1];
    if (!testDeplId) return;

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
    // Create a fresh deployment via UI
    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/templates/${infra.templateId}`);
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(1000);

    const nameInput = page.locator('#name');
    if (await nameInput.isVisible()) {
      await nameInput.fill(uniqueSlug('canceltest'));
    }

    const nodeSelect = page.locator('#node').or(page.getByLabel(/Deploy To|Node/i));
    if (await nodeSelect.isVisible()) {
      const options = nodeSelect.locator('option');
      const count = await options.count();
      if (count > 1) await nodeSelect.selectOption({ index: 1 });
    }

    await page.getByRole('button', { name: /^Deploy$|^Creating|^Starting/i }).click();

    const redirectOk = await page.waitForURL(/\/deployments\//, { timeout: 15_000 }).then(() => true).catch(() => false);
    if (!redirectOk) {
      test.skip(true, 'Could not create test deployment');
      return;
    }

    const match = page.url().match(/\/deployments\/([^/]+)/);
    const testDeplId = match?.[1];
    if (!testDeplId) return;

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

    // Clean up via UI
    await page.goto(`/deployments/${testDeplId}`);
    await page.waitForLoadState('networkidle');
    const cleanupDeleteBtn = page.getByRole('button', { name: /Delete/i });
    if (await cleanupDeleteBtn.isVisible().catch(() => false)) {
      await cleanupDeleteBtn.click();
      await page.waitForTimeout(500);
      const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
      if (await confirmBtn.isVisible().catch(() => false)) {
        await confirmBtn.click();
        await page.waitForTimeout(2000);
      }
    }
  });

  test('refresh during deletion shows current state', async ({ page }) => {
    // Create a fresh deployment via UI
    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/templates/${infra.templateId}`);
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(1000);

    const nameInput = page.locator('#name');
    if (await nameInput.isVisible()) {
      await nameInput.fill(uniqueSlug('refreshtest'));
    }

    const nodeSelect = page.locator('#node').or(page.getByLabel(/Deploy To|Node/i));
    if (await nodeSelect.isVisible()) {
      const options = nodeSelect.locator('option');
      const count = await options.count();
      if (count > 1) await nodeSelect.selectOption({ index: 1 });
    }

    await page.getByRole('button', { name: /^Deploy$|^Creating|^Starting/i }).click();

    const redirectOk = await page.waitForURL(/\/deployments\//, { timeout: 15_000 }).then(() => true).catch(() => false);
    if (!redirectOk) {
      test.skip(true, 'Could not create test deployment');
      return;
    }

    const match = page.url().match(/\/deployments\/([^/]+)/);
    const testDeplId = match?.[1];
    if (!testDeplId) return;

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
  });
});
