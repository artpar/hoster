/**
 * Global setup for E2E tests.
 *
 * Provisions a real DigitalOcean droplet shared by all test suites.
 * Writes infrastructure state to .e2e-infra.json for tests to read.
 *
 * Requires:
 *   - APIGate (:8082) and Hoster (:8080) running
 *   - TEST_DO_API_KEY set in test-data.ts or environment variable
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { chromium, expect } from '@playwright/test';
import { signUp } from './fixtures/auth.fixture';
import { TEST_DO_API_KEY, TEST_PASSWORD, TEST_TEMPLATE_COMPOSE, uniqueEmail, uniqueName } from './fixtures/test-data';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BASE = 'http://localhost:8082';
const INFRA_STATE_PATH = path.join(__dirname, '.e2e-infra.json');

interface InfraState {
  email: string;
  templateId: string;
  dropletIp: string;
}

async function globalSetup() {
  console.log('[global-setup] Starting real infrastructure provisioning...');

  const browser = await chromium.launch();
  const context = await browser.newContext({ baseURL: BASE });
  const page = await context.newPage();

  try {
    // 1. Sign up test user via UI
    const email = uniqueEmail();
    console.log(`[global-setup] Registering user: ${email}`);
    await signUp(page, email, TEST_PASSWORD);
    console.log('[global-setup] User registered successfully');

    // Write partial state immediately so teardown can always log in and clean up
    // even if setup fails after creating cloud resources
    fs.writeFileSync(INFRA_STATE_PATH, JSON.stringify({ email, templateId: '', dropletIp: '' }, null, 2));

    // 2. Create cloud credential with real DO API key via UI
    const credName = uniqueName('e2e-cred');
    console.log(`[global-setup] Creating cloud credential: ${credName}`);

    await page.goto('/nodes/credentials/new');
    await page.waitForLoadState('networkidle');
    await page.locator('#cred-name').fill(credName);
    await page.locator('#cred-provider').selectOption('digitalocean');
    await page.waitForTimeout(300);
    await page.locator('#api-token').fill(TEST_DO_API_KEY);
    await page.getByRole('button', { name: 'Add Credential' }).click();

    // Wait for redirect to credentials list
    await expect(page).toHaveURL(/\/nodes\/credentials$/, { timeout: 10_000 });
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(credName)).toBeVisible({ timeout: 15_000 });
    console.log(`[global-setup] Credential created: ${credName}`);

    // 3. Create cloud provision (real DO droplet) via UI
    const instanceName = `e2e-${Date.now()}`;
    console.log(`[global-setup] Creating cloud provision: ${instanceName} (sfo3, s-1vcpu-1gb)`);

    await page.goto('/nodes/cloud/new');
    await page.waitForLoadState('networkidle');
    await page.locator('#prov-name').fill(instanceName);

    // Select the credential
    const credSelect = page.locator('#prov-credential');
    await credSelect.waitFor({ state: 'attached', timeout: 5_000 });
    await page.waitForFunction(() => {
      const sel = document.querySelector('#prov-credential') as HTMLSelectElement;
      return sel && sel.options.length > 1;
    }, { timeout: 10_000 });
    const credOptions = credSelect.locator('option');
    const credCount = await credOptions.count();
    await credSelect.selectOption({ index: credCount - 1 });

    // Wait for real regions/sizes to load from DigitalOcean API
    await page.waitForTimeout(2000);
    const regionSelect = page.locator('#prov-region');
    await regionSelect.waitFor({ state: 'attached', timeout: 15_000 });

    // Select sfo3 region
    const regionOptions = regionSelect.locator('option');
    const regionCount = await regionOptions.count();
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

    // Click Create Server
    await page.getByRole('button', { name: 'Create Server' }).click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 4. Poll cloud servers page until provision is ready (timeout: 5 min)
    console.log('[global-setup] Waiting for provision to become ready...');
    const deadline = Date.now() + 5 * 60 * 1000;
    let provisionReady = false;
    while (Date.now() < deadline) {
      // Check the Nodes tab for auto-created node (indicates provision completed)
      await page.goto('/nodes');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      const sshInfo = page.getByText(/root@\d+\.\d+\.\d+\.\d+:\d+/);
      if (await sshInfo.isVisible().catch(() => false)) {
        provisionReady = true;
        break;
      }

      // Check cloud servers for failure
      await page.goto('/nodes/cloud');
      await page.waitForLoadState('networkidle');
      const errorEl = page.locator('.bg-destructive\\/10');
      if (await errorEl.isVisible().catch(() => false)) {
        throw new Error('[global-setup] Provision failed');
      }

      const statusText = await page.locator('body').textContent();
      const stepMatch = statusText?.match(/provisioning|installing|configuring|creating/i);
      console.log(`[global-setup] Provision status: in progress (${stepMatch?.[0] ?? 'waiting'})`);

      await new Promise(r => setTimeout(r, 10_000));
    }

    if (!provisionReady) {
      throw new Error('[global-setup] Provision did not reach ready state within 5 minutes');
    }

    // Extract droplet IP from the visible node card (root@IP:PORT text we already found)
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');
    const sshInfoText = await page.getByText(/root@\d+\.\d+\.\d+\.\d+:\d+/).textContent() ?? '';
    const ipMatch = sshInfoText.match(/@(\d+\.\d+\.\d+\.\d+)/);
    const dropletIp = ipMatch?.[1] ?? '';
    console.log(`[global-setup] Provision ready: ip=${dropletIp}`);

    // 5. Create and publish a template via UI
    const tmplName = uniqueName('e2e-tmpl');
    console.log(`[global-setup] Creating template: ${tmplName}`);

    await page.goto('/templates/new');
    await page.waitForLoadState('networkidle');

    // Form fields: Name, Description, Version, Monthly Price (USD), Docker Compose Specification
    await expect(page.getByLabel('Template Name')).toBeVisible({ timeout: 5_000 });
    await page.getByLabel('Template Name').fill(tmplName);
    await page.getByLabel('Description').fill('E2E test template - nginx:alpine');

    // Version field already has default "1.0.0" — clear and re-fill to be explicit
    const versionField = page.getByLabel('Version');
    await expect(versionField).toBeVisible({ timeout: 5_000 });
    await versionField.fill('1.0.0');

    // Price in dollars (frontend multiplies by 100 for cents)
    const priceField = page.getByLabel('Monthly Price (USD)');
    await expect(priceField).toBeVisible({ timeout: 5_000 });
    await priceField.fill('5');

    await page.locator('#compose').fill(TEST_TEMPLATE_COMPOSE);

    const createBtn = page.getByRole('button', { name: 'Create Template' });
    await expect(createBtn).toBeVisible({ timeout: 5_000 });
    await createBtn.click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Extract template ID from redirect URL: /templates/{id}
    const tmplUrl = page.url();
    const tmplMatch = tmplUrl.match(/\/templates\/([^/]+)/);
    const templateId = tmplMatch?.[1] ?? '';

    if (!templateId) {
      throw new Error(`[global-setup] Could not extract template ID from URL: ${tmplUrl}`);
    }

    // Publish: the Publish button is on the TemplateCard in the "My Templates" view
    await page.goto('/templates');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Switch to "My Templates" view — default "Browse All" only shows published templates
    const myTemplatesBtn = page.getByRole('button', { name: 'My Templates' });
    await expect(myTemplatesBtn).toBeVisible({ timeout: 5_000 });
    await myTemplatesBtn.click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // The draft template should now be visible with a Publish button
    const publishBtn = page.getByRole('button', { name: /Publish/i });
    await expect(publishBtn).toBeVisible({ timeout: 5_000 });
    await publishBtn.click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    console.log(`[global-setup] Template published: ${templateId}`);

    // 6. Update state file with full infrastructure info
    const state: InfraState = {
      email,
      templateId,
      dropletIp,
    };
    fs.writeFileSync(INFRA_STATE_PATH, JSON.stringify(state, null, 2));
    console.log(`[global-setup] Infrastructure state written to ${INFRA_STATE_PATH}`);
    console.log('[global-setup] Done. Ready for tests.');
  } finally {
    await browser.close();
  }
}

export default globalSetup;
