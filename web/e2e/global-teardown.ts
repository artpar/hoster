/**
 * Global teardown for E2E tests.
 *
 * Destroys the shared DigitalOcean droplet and cleans up all resources.
 * Reads infrastructure state from .e2e-infra.json.
 *
 * PRINCIPLE: Every step asserts what it expects. No silent skips.
 * If setup created a resource, teardown MUST find and destroy it.
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { chromium, expect } from '@playwright/test';
import { logIn } from './fixtures/auth.fixture';
import { TEST_DO_API_KEY, TEST_PASSWORD } from './fixtures/test-data';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BASE = 'http://localhost:8082';
const INFRA_STATE_PATH = path.join(__dirname, '.e2e-infra.json');

async function globalTeardown() {
  console.log('[global-teardown] Starting cleanup...');

  if (!fs.existsSync(INFRA_STATE_PATH)) {
    console.log('[global-teardown] No infra state file found — nothing to clean up');
    return;
  }

  const state = JSON.parse(fs.readFileSync(INFRA_STATE_PATH, 'utf-8'));
  const { email, templateId } = state;

  if (!email) {
    throw new Error('[global-teardown] State file exists but has no email — corrupt state');
  }

  const browser = await chromium.launch();
  const context = await browser.newContext({ baseURL: BASE });
  const page = await context.newPage();

  try {
    await logIn(page, email, TEST_PASSWORD);

    // 1. Stop and delete any remaining deployments
    console.log('[global-teardown] Cleaning deployments...');
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    const deplLinks = await page.locator('a[href^="/deployments/"]').all();
    for (const link of deplLinks) {
      const href = await link.getAttribute('href');
      if (!href) continue;

      console.log(`[global-teardown] Cleaning deployment at ${href}`);
      await page.goto(href);
      await page.waitForLoadState('networkidle');

      // Stop if running
      const stopBtn = page.getByRole('button', { name: /Stop/i });
      if (await stopBtn.isVisible().catch(() => false) && await stopBtn.isEnabled().catch(() => false)) {
        await stopBtn.click();
        await expect(page.locator('span').filter({ hasText: /Stopped/i })).toBeVisible({ timeout: 60_000 });
      }

      // Delete — this MUST work for every deployment we found
      const deleteBtn = page.getByRole('button', { name: /Delete/i });
      await expect(deleteBtn).toBeVisible({ timeout: 5_000 });
      await deleteBtn.click();
      await page.waitForTimeout(500);

      // Confirm deletion dialog
      const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
      await expect(confirmBtn).toBeVisible({ timeout: 5_000 });
      await confirmBtn.click();
      await page.waitForTimeout(2000);
    }
    if (deplLinks.length > 0) {
      console.log(`[global-teardown] Deleted ${deplLinks.length} deployment(s)`);
    } else {
      console.log('[global-teardown] No deployments to clean');
    }

    // 2. Destroy ALL provisions (droplets) via cloud servers UI
    console.log('[global-teardown] Destroying all provisions via UI...');
    await page.goto('/nodes/cloud');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Click Destroy on each provision one at a time
    let destroyCount = 0;
    const maxAttempts = 10;
    while (destroyCount < maxAttempts) {
      const destroyBtn = page.getByRole('button', { name: /Destroy/i }).first();
      if (!(await destroyBtn.isVisible().catch(() => false))) {
        break;
      }
      destroyCount++;
      console.log(`[global-teardown] Destroying provision ${destroyCount}...`);
      await destroyBtn.click();
      await page.waitForTimeout(500);

      // Confirm destruction dialog
      const confirmBtn = page.getByRole('button', { name: /Destroy|Confirm/i }).last();
      await expect(confirmBtn).toBeVisible({ timeout: 5_000 });
      await confirmBtn.click();
      await page.waitForTimeout(3000);

      // Reload to see remaining provisions
      await page.goto('/nodes/cloud');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);
    }

    // Wait for all provisions to reach Destroyed status
    if (destroyCount > 0) {
      console.log(`[global-teardown] Initiated destruction of ${destroyCount} provision(s), waiting for completion...`);
      const deadline = Date.now() + 3 * 60 * 1000;
      while (Date.now() < deadline) {
        await page.goto('/nodes/cloud');
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(2000);

        // Check if any Destroy buttons remain (meaning provisions still active)
        const pendingDestroy = page.getByRole('button', { name: /Destroy/i });
        if (!(await pendingDestroy.isVisible().catch(() => false))) {
          // No more Destroy buttons — all provisions are destroyed or destroying
          const destroyingBadge = page.locator('span').filter({ hasText: /Destroying/i });
          if (!(await destroyingBadge.isVisible().catch(() => false))) {
            // No "Destroying" badges either — all done
            break;
          }
        }

        await new Promise(r => setTimeout(r, 5000));
      }

      // Final assertion: no Destroy buttons should remain
      await page.goto('/nodes/cloud');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);
      const remainingDestroy = page.getByRole('button', { name: /Destroy/i });
      if (await remainingDestroy.isVisible().catch(() => false)) {
        throw new Error('[global-teardown] Provisions still have Destroy buttons after 3 min — droplets may be leaked!');
      }
      console.log('[global-teardown] All provisions destroyed');
    } else {
      console.log('[global-teardown] No provisions to destroy');
    }

    // 3. Delete template — from "My Templates" view in the templates list
    if (templateId) {
      console.log(`[global-teardown] Deleting template: ${templateId}`);
      await page.goto('/templates');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);

      // Switch to "My Templates" view to see our template (with Delete button on TemplateCard)
      const myTemplatesBtn = page.getByRole('button', { name: 'My Templates' });
      await expect(myTemplatesBtn).toBeVisible({ timeout: 5_000 });
      await myTemplatesBtn.click();
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);

      const deleteBtn = page.getByRole('button', { name: /Delete/i });
      await expect(deleteBtn).toBeVisible({ timeout: 5_000 });
      await deleteBtn.click();
      await page.waitForTimeout(500);

      // Confirm dialog
      const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
      await expect(confirmBtn).toBeVisible({ timeout: 5_000 });
      await confirmBtn.click();
      await page.waitForTimeout(1000);
      console.log('[global-teardown] Template deleted');
    }

    // 4. Delete credential — we KNOW we created one
    console.log('[global-teardown] Deleting credential...');
    await page.goto('/nodes/credentials');
    await page.waitForLoadState('networkidle');

    const credDeleteBtn = page.getByRole('button', { name: /Delete/i }).first();
    await expect(credDeleteBtn).toBeVisible({ timeout: 5_000 });
    await credDeleteBtn.click();
    await page.waitForTimeout(500);

    const credConfirmBtn = page.getByRole('button', { name: /Delete|Confirm/i }).last();
    await expect(credConfirmBtn).toBeVisible({ timeout: 5_000 });
    await credConfirmBtn.click();
    await page.waitForTimeout(1000);
    console.log('[global-teardown] Credential deleted');
  } finally {
    await browser.close();
  }

  // 5. Verify droplets are actually gone on DigitalOcean — not just in our DB.
  // This catches bugs where our system reports "destroyed" but the actual cloud
  // resource was never deleted (e.g., state machine transition failures).
  console.log('[global-teardown] Verifying no leaked droplets on DigitalOcean...');
  const doResp = await fetch('https://api.digitalocean.com/v2/droplets?per_page=200', {
    headers: { 'Authorization': `Bearer ${TEST_DO_API_KEY}` },
  });
  if (!doResp.ok) {
    console.warn(`[global-teardown] WARNING: Could not verify droplets (DO API returned ${doResp.status})`);
  } else {
    const doData = await doResp.json() as { droplets: Array<{ id: number; name: string; status: string }> };
    // Match all known E2E test droplet name prefixes.
    // global-setup uses "e2e-<timestamp>", UJ4 tests use uniqueName('uj4node') → "uj4node-<uid>".
    // Any droplet matching these patterns was created by our tests and should not exist after teardown.
    const testPrefixes = ['e2e-', 'uj4node-'];
    const leaked = doData.droplets.filter(d =>
      testPrefixes.some(prefix => d.name.startsWith(prefix))
    );

    if (leaked.length > 0) {
      console.error(`[global-teardown] LEAKED DROPLETS FOUND: ${leaked.length}`);
      for (const d of leaked) {
        console.error(`[global-teardown]   Destroying leaked droplet ${d.id} (${d.name}, ${d.status})`);
        await fetch(`https://api.digitalocean.com/v2/droplets/${d.id}`, {
          method: 'DELETE',
          headers: { 'Authorization': `Bearer ${TEST_DO_API_KEY}` },
        });
      }
      throw new Error(
        `[global-teardown] ${leaked.length} leaked droplet(s) found and force-destroyed: ` +
        leaked.map(d => `${d.id}(${d.name})`).join(', ') +
        '. This indicates a bug in the destroy flow — the system reported success but the cloud resource was not deleted.'
      );
    }
    console.log('[global-teardown] Verified: 0 leaked droplets on DigitalOcean');
  }

  // 6. Remove state file only after all cleanup succeeded
  if (fs.existsSync(INFRA_STATE_PATH)) {
    fs.unlinkSync(INFRA_STATE_PATH);
  }
  console.log('[global-teardown] Cleanup complete.');
}

export default globalTeardown;
