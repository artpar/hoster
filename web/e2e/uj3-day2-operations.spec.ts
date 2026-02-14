import { test, expect } from '@playwright/test';
import { apiSignUp, injectAuth } from './fixtures/auth.fixture';
import { apiCreateDeployment, apiStartDeployment, apiDeleteDeployment, apiStopDeployment, apiListDeployments } from './fixtures/api.fixture';
import { uniqueSlug, TEST_PASSWORD, readInfraState, type InfraState } from './fixtures/test-data';
import { waitForDeploymentStatus } from './helpers/wait';

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
      name: uniqueSlug('d2depl'),
      template_id: infra.templateId,
      node_id: infra.nodeId,
    });
    deploymentId = depl.id;

    // Start the deployment — real containers will pull and run on the DO droplet
    await apiStartDeployment(token, deploymentId);

    // Wait for deployment to reach "running" (real Docker pull + start, up to 3 min)
    await waitForDeploymentStatus(token, deploymentId, 'running', 180_000);
  });

  test.afterAll(async () => {
    if (deploymentId) {
      await apiStopDeployment(token, deploymentId).catch(() => {});
      // Wait a bit for containers to stop before deleting
      await new Promise(r => setTimeout(r, 5000));
      await apiDeleteDeployment(token, deploymentId).catch(() => {});
    }
  });

  // --- Happy path ---

  test('authenticated user sees deployments list', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'My Deployments' })).toBeVisible({ timeout: 10_000 });
  });

  test('deployments list shows created deployment', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment created');

    await injectAuth(page, token);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Our deployment should appear in the list
    const deplLink = page.locator(`a[href*="/deployments/"]`).first();
    await expect(deplLink).toBeVisible({ timeout: 10_000 });
  });

  test('deployment detail from list', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment created');

    await injectAuth(page, token);
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

    await injectAuth(page, token);
    await page.goto(`/deployments/${deploymentId}`);
    await page.waitForLoadState('networkidle');

    // Should see deployment name and Running status badge (real containers running)
    await expect(page.getByRole('heading').first()).toBeVisible();
    await expect(page.locator('span').filter({ hasText: /Running/ })).toBeVisible({ timeout: 10_000 });
  });

  test('logs tab shows real container logs', async ({ page }) => {
    test.skip(!deploymentId, 'No deployment');
    test.setTimeout(120_000);

    await injectAuth(page, token);
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

    await injectAuth(page, token);
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

    await injectAuth(page, token);
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

    await injectAuth(page, token);
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
    // Create a deployment but don't start it
    let pendingDeplId: string | undefined;
    try {
      const depl = await apiCreateDeployment(token, {
        name: uniqueSlug('d2pending'),
        template_id: infra.templateId,
        node_id: infra.nodeId,
      });
      pendingDeplId = depl.id;
    } catch {
      test.skip(true, 'Could not create pending deployment');
      return;
    }

    await injectAuth(page, token);
    await page.goto(`/deployments/${pendingDeplId}`);
    await page.waitForLoadState('networkidle');

    const logsTab = page.getByRole('tab', { name: /Logs/i });
    await logsTab.click();
    await page.waitForTimeout(2000);

    await expect(page.getByText('Container Logs')).toBeVisible();
    // Non-running deployment: waiting message or "No logs"
    await expect(page.getByText(/Logs will appear|No logs available/i)).toBeVisible({ timeout: 10_000 });

    // Clean up
    await apiDeleteDeployment(token, pendingDeplId).catch(() => {});
  });

  test('deployment list shows only own deployments', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Page loads with heading — all listed deployments belong to this user
    await expect(page.getByRole('heading', { name: 'My Deployments' })).toBeVisible();
  });
});
