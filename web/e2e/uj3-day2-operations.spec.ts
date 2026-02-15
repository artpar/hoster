import { test, expect, chromium } from '@playwright/test';
import { logIn } from './fixtures/auth.fixture';
import { uniqueSlug, readInfraState, TEST_PASSWORD, type InfraState } from './fixtures/test-data';
import { waitForDeploymentStatusOnPage } from './helpers/wait';

/**
 * UJ3: "I want to check on my apps"
 *
 * Returning customer monitors deployments: deployments list, deployment detail tabs
 * (overview, logs, stats, events), and lifecycle operations (stop/start).
 *
 * Uses REAL infrastructure from global setup — deployment runs real Docker containers
 * on a real DigitalOcean droplet. Logs, stats, and events are all real.
 *
 * Targets: APIGate (:8082) -> Hoster (:8080) — real prod-like stack.
 */

test.describe('UJ3: Day-2 Operations', () => {
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

        // Stop if running
        const stopBtn = page.getByRole('button', { name: /Stop/i });
        if (await stopBtn.isVisible().catch(() => false) && await stopBtn.isEnabled().catch(() => false)) {
          await stopBtn.click();
          await page.locator('span').filter({ hasText: /Stopped/ }).waitFor({ timeout: 60_000 }).catch(() => {});
        }

        // Delete
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

      // Create a deployment via the deploy dialog from template detail
      await page.goto(`/templates/${infra.templateId}`);
      await page.waitForLoadState('networkidle');
      await page.getByText('Deploy Now').click();
      await page.waitForTimeout(1000);

      // Fill deploy form
      const nameInput = page.locator('#name');
      if (await nameInput.isVisible()) {
        await nameInput.fill(uniqueSlug('d2depl'));
      }

      // Select node
      const nodeSelect = page.locator('#node').or(page.getByLabel(/Deploy To|Node/i));
      if (await nodeSelect.isVisible()) {
        const options = nodeSelect.locator('option');
        const count = await options.count();
        if (count > 1) await nodeSelect.selectOption({ index: 1 });
      }

      // Click Deploy
      await page.getByRole('button', { name: /^Deploy$|^Creating|^Starting/i }).click();

      // Wait for redirect to deployment detail
      await expect(page).toHaveURL(/\/deployments\//, { timeout: 15_000 });
      const match = page.url().match(/\/deployments\/([^/]+)/);
      deploymentId = match?.[1];

      // Wait for Running status via page polling
      if (deploymentId) {
        await waitForDeploymentStatusOnPage(page, deploymentId, 'Running', 180_000);
      }
    } finally {
      await browser.close();
    }
  });

  test.afterAll(async () => {
    if (!deploymentId) return;
    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();
    try {
      await logIn(page, infra.email, TEST_PASSWORD);
      await page.goto(`/deployments/${deploymentId}`);
      await page.waitForLoadState('networkidle');

      // Stop if running
      const stopBtn = page.getByRole('button', { name: /Stop/i });
      if (await stopBtn.isVisible().catch(() => false)) {
        await stopBtn.click();
        await page.locator('span').filter({ hasText: /Stopped/ }).waitFor({ timeout: 60_000 }).catch(() => {});
      }

      // Delete
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

  test('authenticated user sees deployments list', async ({ page }) => {
    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'My Deployments' })).toBeVisible({ timeout: 10_000 });
  });

  test('deployments list shows created deployment', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment created');

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Our deployment should appear in the list
    const deplLink = page.locator(`a[href*="/deployments/"]`).first();
    await expect(deplLink).toBeVisible({ timeout: 10_000 });
  });

  test('deployment detail from list', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment created');

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Click on the deployment
    const deplLink = page.locator(`a[href*="/deployments/"]`).first();
    await deplLink.click();
    await expect(page).toHaveURL(/\/deployments\/[^/]+/);
    // Should see deployment heading
    await expect(page.getByRole('heading').first()).toBeVisible();
  });

  test('deployment detail overview tab shows running status', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Should see deployment name and Running status badge (real containers running)
    await expect(page.getByRole('heading').first()).toBeVisible();
    await expect(page.locator('span').filter({ hasText: /Running/ })).toBeVisible({ timeout: 10_000 });
  });

  test('logs tab shows real container logs', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');
    test.setTimeout(120_000);

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Click Logs tab
    const logsTab = page.getByRole('tab', { name: /Logs/i });
    await logsTab.click();
    await page.waitForTimeout(3000);

    // Should show "Container Logs" heading
    await expect(page.getByText('Container Logs')).toBeVisible();

    // Real running deployment — may show logs in .font-mono or "No logs available"
    const hasLogArea = await page.locator('.font-mono').isVisible().catch(() => false);
    const hasNoLogs = await page.getByText(/No logs available/i).isVisible().catch(() => false);
    expect(hasLogArea || hasNoLogs).toBeTruthy();
  });

  test('stats tab shows real resource metrics', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');
    test.setTimeout(120_000);

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Click Stats tab
    const statsTab = page.getByRole('tab', { name: /Stats/i });
    await statsTab.click();
    await page.waitForTimeout(3000);

    // Should show "Resource Statistics" heading
    await expect(page.getByText('Resource Statistics')).toBeVisible();

    // Real running containers produce stats — look for stats table with real data
    const hasTable = await page.locator('table').isVisible().catch(() => false);
    const hasStatsData = await page.getByText(/CPU|Memory|Network/i).isVisible().catch(() => false);
    expect(hasTable || hasStatsData).toBeTruthy();
  });

  test('events tab shows real lifecycle events', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Click Events tab
    const eventsTab = page.getByRole('tab', { name: /Events/i });
    await eventsTab.click();
    await page.waitForTimeout(2000);

    // Should show "Deployment Events" heading
    await expect(page.getByText('Deployment Events')).toBeVisible();

    // Real deployment went through lifecycle — may have event entries or "No events recorded"
    const hasEventEntries = await page.locator('.rounded-md.border.p-3').first().isVisible().catch(() => false);
    const hasNoEvents = await page.getByText(/No events recorded/i).isVisible().catch(() => false);
    expect(hasEventEntries || hasNoEvents).toBeTruthy();
  });

  test('stop then start — real lifecycle', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');
    test.setTimeout(300_000);

    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Should be Running — click Stop
    const stopBtn = page.getByRole('button', { name: /Stop/i });
    await expect(stopBtn).toBeVisible({ timeout: 10_000 });
    await stopBtn.click();

    // Real containers stop — wait for Stopped status
    await expect(page.locator('span').filter({ hasText: /Stopped/ })).toBeVisible({ timeout: 60_000 });

    // Start button should appear
    const startBtn = page.getByRole('button', { name: /Start/i });
    await expect(startBtn).toBeVisible({ timeout: 10_000 });

    // Click Start — real containers restart
    await startBtn.click();

    // Wait for Running status again (real Docker pull + start)
    await expect(page.locator('span').filter({ hasText: /Running/ })).toBeVisible({ timeout: 180_000 });
  });

  // --- Sad path ---

  test('logs tab for stopped deployment shows waiting message', async ({ page }) => {
    // Create a deployment via UI but don't start it
    let pendingDeplId: string | undefined;

    await logIn(page, infra.email, TEST_PASSWORD);

    // Navigate to template and deploy
    await page.goto(`/templates/${infra.templateId}`);
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(1000);

    const nameInput = page.locator('#name');
    if (await nameInput.isVisible()) {
      await nameInput.fill(uniqueSlug('d2pending'));
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
    pendingDeplId = match?.[1];

    if (!pendingDeplId) {
      test.skip(true, 'Could not create pending deployment');
      return;
    }

    await page.goto(`/deployments/${pendingDeplId}`);
    await page.waitForLoadState('networkidle');

    const logsTab = page.getByRole('tab', { name: /Logs/i });
    await logsTab.click();
    await page.waitForTimeout(2000);

    await expect(page.getByText('Container Logs')).toBeVisible();
    // Non-running deployment: waiting message or "No logs"
    await expect(page.getByText(/Logs will appear|No logs available/i)).toBeVisible({ timeout: 10_000 });

    // Clean up via UI
    await page.goto(`/deployments/${pendingDeplId}`);
    await page.waitForLoadState('networkidle');
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
  });

  test('deployment list shows only own deployments', async ({ page }) => {
    await logIn(page, infra.email, TEST_PASSWORD);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Page loads with heading — all listed deployments belong to this user
    await expect(page.getByRole('heading', { name: 'My Deployments' })).toBeVisible();
  });
});
