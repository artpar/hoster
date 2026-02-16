import { test, expect, chromium } from '@playwright/test';
import { signUp, logIn } from './fixtures/auth.fixture';
import { uniqueEmail, uniqueName, TEST_PASSWORD, TEST_DO_API_KEY, readInfraState, type InfraState } from './fixtures/test-data';
import { waitForProvisionStatusOnPage } from './helpers/wait';

/**
 * UJ4: "I want to scale up my infrastructure"
 *
 * Operator manages cloud credentials, provisions cloud servers, and verifies
 * auto-registered nodes.
 *
 * Test 4 actually provisions a REAL second DigitalOcean droplet through the UI,
 * waits for it to become ready, and destroys it in cleanup.
 *
 * Targets: APIGate (:8082) -> Hoster (:8080) — real prod-like stack.
 */

test.describe('UJ4: Infrastructure Scaling', () => {
  let email: string;
  let infra: InfraState | null;
  let uj4ProvisionCreated = false;

  test.beforeAll(async () => {
    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();
    try {
      email = uniqueEmail();
      await signUp(page, email, TEST_PASSWORD);
    } finally {
      await browser.close();
    }
    infra = readInfraState();
  });

  test.afterAll(async ({}, testInfo) => {
    testInfo.setTimeout(480_000);

    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();

    try {
      await logIn(page, email, TEST_PASSWORD);

      // Destroy provisions via cloud servers UI
      if (uj4ProvisionCreated) {
        // Wait for provision to reach a destroyable state (ready or failed)
        console.log('[UJ4 afterAll] Waiting for provision to reach destroyable state...');
        const deadline = Date.now() + 240_000;
        let reachedDestroyable = false;
        while (Date.now() < deadline) {
          await page.goto('/nodes/cloud');
          await page.waitForLoadState('networkidle');
          await page.waitForTimeout(2000);

          const destroyBtn = page.getByRole('button', { name: /Destroy/i }).first();
          if (await destroyBtn.isVisible().catch(() => false)) {
            reachedDestroyable = true;
            break;
          }

          await new Promise(r => setTimeout(r, 10_000));
        }

        if (!reachedDestroyable) {
          throw new Error('[UJ4 afterAll] Provision never showed a Destroy button — droplet may be leaked!');
        }

        // Click Destroy on each provision — assert the button exists
        await page.goto('/nodes/cloud');
        await page.waitForLoadState('networkidle');
        const destroyBtns = await page.getByRole('button', { name: /Destroy/i }).all();
        console.log(`[UJ4 afterAll] Found ${destroyBtns.length} Destroy button(s)`);

        for (const btn of destroyBtns) {
          await expect(btn).toBeVisible({ timeout: 5_000 });
          await btn.click();
          await page.waitForTimeout(500);

          // Assert confirm dialog appears
          const confirmBtn = page.getByRole('button', { name: /Destroy|Confirm/i }).last();
          await expect(confirmBtn).toBeVisible({ timeout: 5_000 });
          await confirmBtn.click();
          await page.waitForTimeout(3000);
        }

        // Assert destruction completes — no silent catch
        console.log('[UJ4 afterAll] Waiting for Destroyed status...');
        await waitForProvisionStatusOnPage(page, 'Destroyed', 180_000);
        console.log('[UJ4 afterAll] All provisions destroyed');
      }

      // Clean up credentials — assert each delete works
      await page.goto('/nodes/credentials');
      await page.waitForLoadState('networkidle');
      const deleteBtns = await page.getByRole('button', { name: /Delete/i }).all();
      for (const btn of deleteBtns) {
        await expect(btn).toBeVisible({ timeout: 5_000 });
        await btn.click();
        await page.waitForTimeout(500);

        const confirmBtn = page.getByRole('button', { name: /Delete|Confirm/i }).last();
        await expect(confirmBtn).toBeVisible({ timeout: 5_000 });
        await confirmBtn.click();
        await page.waitForTimeout(1000);

        // Reload after each delete
        await page.goto('/nodes/credentials');
        await page.waitForLoadState('networkidle');
      }
    } finally {
      await browser.close();
    }
  });

  // --- Happy path ---

  test('nodes page has three tabs', async ({ page }) => {
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');

    // Three navigation tabs: Nodes, Provisioning, Credentials
    await expect(page.getByText('Nodes').first()).toBeVisible();
    await expect(page.getByText('Provisioning')).toBeVisible();
    await expect(page.getByText('Credentials')).toBeVisible();
  });

  test('add cloud credential for AWS', async ({ page }) => {
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes/credentials/new');
    await page.waitForLoadState('networkidle');

    // Should show "Add Cloud Credential" heading
    await expect(page.getByText('Add Cloud Credential')).toBeVisible();

    const credName = uniqueName('awscred');
    await page.locator('#cred-name').fill(credName);

    // Provider defaults to AWS, so just verify it
    await expect(page.locator('#cred-provider')).toHaveValue('aws');

    // Fill AWS-specific fields
    await page.locator('#aws-access-key').fill('AKIAIOSFODNN7EXAMPLE');
    await page.locator('#aws-secret-key').fill('wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY');

    await page.getByRole('button', { name: 'Add Credential' }).click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Should redirect to credentials list
    await expect(page).toHaveURL(/\/nodes\/credentials/, { timeout: 5_000 });
    await expect(page.getByText(credName)).toBeVisible({ timeout: 5_000 });
  });

  test('add cloud credential for DigitalOcean', async ({ page }) => {
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes/credentials/new');
    await page.waitForLoadState('networkidle');

    const credName = uniqueName('docred');
    await page.locator('#cred-name').fill(credName);

    // Select DigitalOcean provider
    await page.locator('#cred-provider').selectOption('digitalocean');
    await page.waitForTimeout(300);

    // Fill DO-specific field (API Token)
    await page.locator('#api-token').fill('dop_v1_example_token_for_testing');

    await page.getByRole('button', { name: 'Add Credential' }).click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Should redirect to credentials list
    await expect(page).toHaveURL(/\/nodes\/credentials/, { timeout: 5_000 });
    await expect(page.getByText(credName)).toBeVisible({ timeout: 5_000 });
  });

  test('provision real DO droplet through UI', async ({ page }) => {
    test.setTimeout(420_000);

    // Create a real DO credential via UI first
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes/credentials/new');
    await page.waitForLoadState('networkidle');

    const credName = uniqueName('doreal');
    await page.locator('#cred-name').fill(credName);
    await page.locator('#cred-provider').selectOption('digitalocean');
    await page.waitForTimeout(300);
    await page.locator('#api-token').fill(TEST_DO_API_KEY);
    await page.getByRole('button', { name: 'Add Credential' }).click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Navigate to cloud server creation
    await page.goto('/nodes/cloud/new');
    await page.waitForLoadState('networkidle');

    // Should show "Create Cloud Server" heading
    await expect(page.getByText('Create Cloud Server')).toBeVisible();

    // Fill instance name
    const instanceName = uniqueName('uj4node');
    await page.locator('#prov-name').fill(instanceName);

    // Select the real DO credential by name (earlier tests created other credentials)
    const credSelect = page.locator('#prov-credential');
    await credSelect.selectOption({ label: `${credName} (digitalocean)` });

    // Wait for real regions/sizes to load from DigitalOcean API
    await expect(page.locator('#prov-region')).toBeEnabled({ timeout: 15_000 });
    await expect(page.locator('#prov-size')).toBeEnabled({ timeout: 15_000 });

    // Select sfo3 region and smallest size
    const regionSelect = page.locator('#prov-region');
    const regionOptions = regionSelect.locator('option');
    const regionCount = await regionOptions.count();
    // Try to find sfo3, otherwise pick first available
    let selectedRegion = false;
    for (let i = 1; i < regionCount; i++) {
      const val = await regionOptions.nth(i).getAttribute('value');
      if (val === 'sfo3') {
        await regionSelect.selectOption('sfo3');
        selectedRegion = true;
        break;
      }
    }
    if (!selectedRegion && regionCount > 1) {
      await regionSelect.selectOption({ index: 1 });
    }

    // Select smallest size
    await page.locator('#prov-size').selectOption({ index: 1 });

    // Click "Create Server" — this ACTUALLY provisions a real DO droplet
    await page.getByRole('button', { name: 'Create Server' }).click();
    uj4ProvisionCreated = true;

    // Should redirect to cloud servers list
    await expect(page).toHaveURL(/\/nodes\/cloud/, { timeout: 15_000 });

    // The cloud servers tab filters out "ready" provisions, so poll the Nodes tab
    // for the auto-created NodeCard — distinguishable from ProvisionCard by SSH info.
    // NodeCard shows "root@<ip>:22"; ProvisionCard does not.
    const deadline = Date.now() + 360_000;
    while (Date.now() < deadline) {
      await page.goto('/nodes');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      // NodeCard renders SSH connection info: "{ssh_user}@{ssh_host}:{ssh_port}"
      // This text only appears when the node is created (provision completed).
      const sshInfo = page.getByText(/root@\d+\.\d+\.\d+\.\d+:\d+/);
      if (await sshInfo.isVisible().catch(() => false)) {
        break;
      }

      // Check cloud servers tab for "failed" to bail early
      await page.goto('/nodes/cloud');
      await page.waitForLoadState('networkidle');
      const errorEl = page.locator('.bg-destructive\\/10');
      if (await errorEl.isVisible().catch(() => false)) {
        throw new Error('UJ4: Cloud provision failed');
      }

      await new Promise(r => setTimeout(r, 10_000));
    }

    // Verify the node was auto-created — NodeCard shows SSH connection info
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(/root@\d+\.\d+\.\d+\.\d+:\d+/)).toBeVisible({ timeout: 10_000 });
  });

  // --- Sad path ---

  test('empty credentials shows empty state', async ({ page }) => {
    // Create a fresh user with no credentials
    const freshEmail = uniqueEmail();
    const browser2 = await chromium.launch();
    const ctx2 = await browser2.newContext({ baseURL: 'http://localhost:8082' });
    const page2 = await ctx2.newPage();
    try {
      await signUp(page2, freshEmail, TEST_PASSWORD);
    } finally {
      await browser2.close();
    }

    // Log in as the fresh user in the test page
    await logIn(page, freshEmail, TEST_PASSWORD);
    await page.goto('/nodes/credentials');
    await page.waitForLoadState('networkidle');

    // Should show empty state
    await expect(page.getByText('No cloud credentials')).toBeVisible({ timeout: 5_000 });
  });

  test('create credential without required fields', async ({ page }) => {
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes/credentials/new');
    await page.waitForLoadState('networkidle');

    // Try to submit without filling name (AWS is default, needs access key too)
    await page.getByRole('button', { name: 'Add Credential' }).click();
    await page.waitForTimeout(500);

    // Should show validation error
    await expect(page.getByText('Name is required')).toBeVisible();
  });

  test('create cloud server without credential', async ({ page }) => {
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes/cloud/new');
    await page.waitForLoadState('networkidle');

    // Fill name but don't select credential
    await page.locator('#prov-name').fill('test-server');

    // Click submit
    await page.getByRole('button', { name: 'Create Server' }).click();
    await page.waitForTimeout(500);

    // Should show "Select a credential" error
    await expect(page.getByText('Select a credential', { exact: true })).toBeVisible();
  });

  test('region/size selects disabled before credential chosen', async ({ page }) => {
    await logIn(page, email, TEST_PASSWORD);
    await page.goto('/nodes/cloud/new');
    await page.waitForLoadState('networkidle');

    // Region and size should be disabled before credential is selected
    await expect(page.locator('#prov-region')).toBeDisabled();
    await expect(page.locator('#prov-size')).toBeDisabled();
  });
});
