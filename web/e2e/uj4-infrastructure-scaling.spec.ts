import { test, expect } from '@playwright/test';
import { apiSignUp, injectAuth } from './fixtures/auth.fixture';
import { apiCreateCloudCredential, apiDeleteCloudCredential, apiDestroyCloudProvision, apiGetCloudProvision } from './fixtures/api.fixture';
import { uniqueEmail, uniqueName, TEST_PASSWORD, TEST_DO_API_KEY, readInfraState, type InfraState } from './fixtures/test-data';
import { waitForProvisionStatus } from './helpers/wait';

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
  let token: string;
  let infra: InfraState | null;
  const credentialIds: string[] = [];
  let uj4ProvisionId: string | undefined;

  test.beforeAll(async () => {
    const email = uniqueEmail();
    const result = await apiSignUp(email, TEST_PASSWORD);
    token = result.token;
    infra = readInfraState();
  });

  test.afterAll(async ({}, testInfo) => {
    testInfo.setTimeout(300_000);

    // Destroy any provisioned droplet from test 4
    if (uj4ProvisionId) {
      try {
        // Wait for provision to reach a destroyable state (ready or failed)
        // State machine: only ready → destroying and failed → destroying are valid
        await waitForProvisionStatus(token, uj4ProvisionId, 'ready', 240_000);
      } catch {
        // If it didn't reach "ready", it might be "failed" — that's also destroyable
      }
      try {
        await apiDestroyCloudProvision(token, uj4ProvisionId);
        await waitForProvisionStatus(token, uj4ProvisionId, 'destroyed', 180_000);
      } catch (err) {
        console.warn('UJ4 teardown: failed to destroy provision:', err);
      }
    }
    for (const id of credentialIds) {
      await apiDeleteCloudCredential(token, id).catch(() => {});
    }
  });

  // --- Happy path ---

  test('nodes page has three tabs', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');

    // Three navigation tabs: Nodes, Cloud Servers, Credentials
    await expect(page.getByText('Nodes').first()).toBeVisible();
    await expect(page.getByText('Cloud Servers')).toBeVisible();
    await expect(page.getByText('Credentials')).toBeVisible();
  });

  test('add cloud credential for AWS', async ({ page }) => {
    await injectAuth(page, token);
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
    await injectAuth(page, token);
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
    test.setTimeout(300_000);

    // Create a real DO credential with real API key for this test user
    const cred = await apiCreateCloudCredential(token, {
      name: uniqueName('doreal'),
      provider: 'digitalocean',
      api_key: TEST_DO_API_KEY,
    });
    credentialIds.push(cred.id);

    await injectAuth(page, token);
    await page.goto('/nodes/cloud/new');
    await page.waitForLoadState('networkidle');

    // Should show "Create Cloud Server" heading
    await expect(page.getByText('Create Cloud Server')).toBeVisible();

    // Fill instance name
    const instanceName = uniqueName('uj4node');
    await page.locator('#prov-name').fill(instanceName);

    // Select the real DO credential by its known ID (not by index — other creds exist)
    await page.locator('#prov-credential').selectOption(cred.id);

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

    // Should redirect to cloud servers list
    await expect(page).toHaveURL(/\/nodes\/cloud/, { timeout: 15_000 });

    // The cloud servers tab filters out "ready" provisions, so poll the Nodes tab
    // for the auto-created NodeCard — distinguishable from ProvisionCard by SSH info.
    // NodeCard shows "root@<ip>:22"; ProvisionCard does not.
    const deadline = Date.now() + 300_000;
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

    // Track provision ID for cleanup — fetch via page context
    try {
      uj4ProvisionId = await page.evaluate(async (tok) => {
        const res = await fetch('/api/v1/cloud_provisions', {
          headers: { Accept: 'application/vnd.api+json', Authorization: `Bearer ${tok}` },
        });
        if (!res.ok) return undefined;
        const data = await res.json();
        const provisions = (data as any).data ?? [];
        for (const p of provisions) {
          if (p.attributes?.instance_name?.includes('uj4node')) return p.id;
        }
        return undefined;
      }, token);
    } catch {
      // Best effort — teardown will skip if no ID
    }
  });

  // --- Sad path ---

  test('empty credentials shows empty state', async ({ page }) => {
    // Create a fresh user with no credentials
    const freshEmail = uniqueEmail();
    const fresh = await apiSignUp(freshEmail, TEST_PASSWORD);

    await injectAuth(page, fresh.token);
    await page.goto('/nodes/credentials');
    await page.waitForLoadState('networkidle');

    // Should show empty state
    await expect(page.getByText('No cloud credentials')).toBeVisible({ timeout: 5_000 });
  });

  test('create credential without required fields', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/nodes/credentials/new');
    await page.waitForLoadState('networkidle');

    // Try to submit without filling name (AWS is default, needs access key too)
    await page.getByRole('button', { name: 'Add Credential' }).click();
    await page.waitForTimeout(500);

    // Should show validation error
    await expect(page.getByText('Name is required')).toBeVisible();
  });

  test('create cloud server without credential', async ({ page }) => {
    await injectAuth(page, token);
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
    await injectAuth(page, token);
    await page.goto('/nodes/cloud/new');
    await page.waitForLoadState('networkidle');

    // Region and size should be disabled before credential is selected
    await expect(page.locator('#prov-region')).toBeDisabled();
    await expect(page.locator('#prov-size')).toBeDisabled();
  });
});
