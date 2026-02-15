import { test, expect, chromium } from '@playwright/test';
import { signUp, logIn } from './fixtures/auth.fixture';
import { uniqueEmail, uniqueName, uniqueSlug, TEST_PASSWORD, TEST_TEMPLATE_COMPOSE, readInfraState, type InfraState } from './fixtures/test-data';

/**
 * UJ2: "I want to deploy my first app"
 *
 * New user signs up, discovers they need infrastructure, sets up SSH key + node,
 * then deploys a template to a REAL node with REAL containers starting.
 *
 * Uses shared infrastructure from global setup for the deployment test.
 * SSH key and node creation tests use the UI forms.
 *
 * Targets: APIGate (:8082) -> Hoster (:8080) — real prod-like stack.
 *
 * NOTE: /sign-up through the Hoster SPA shows Hoster's signup form.
 * APIGate's reserved paths (/login, /signup, /dashboard) are avoided.
 */

test.describe('UJ2: First Deployment', () => {
  let email: string;
  let password: string;
  let signedUp = false;
  let infra: InfraState | null;
  const cleanupIds: { deployments: string[]; templates: string[] } = {
    deployments: [],
    templates: [],
  };

  test.beforeAll(async () => {
    email = uniqueEmail();
    password = TEST_PASSWORD;
    infra = readInfraState();
  });

  test.afterAll(async () => {
    if (!signedUp) return;
    // Clean up via browser
    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();
    try {
      await logIn(page, email, password);

      // Delete deployments via UI
      for (const id of cleanupIds.deployments) {
        await page.goto(`/deployments/${id}`);
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
      }

      // Delete templates via UI
      for (const id of cleanupIds.templates) {
        await page.goto(`/templates/${id}`);
        await page.waitForLoadState('networkidle');
        const deleteBtn = page.getByRole('button', { name: /Delete/i });
        if (await deleteBtn.isVisible().catch(() => false)) {
          await deleteBtn.click();
          await page.waitForTimeout(500);
          const confirmBtn = page.getByRole('button', { name: /Delete/i }).last();
          if (await confirmBtn.isVisible().catch(() => false)) {
            await confirmBtn.click();
            await page.waitForTimeout(1000);
          }
        }
      }
    } finally {
      await browser.close();
    }
  });

  // --- Happy path ---

  test('signup creates account via Hoster SPA', async ({ page }) => {
    await page.goto('/sign-up');
    // Hoster SPA signup form
    await page.locator('#email').fill(email);
    await page.locator('#password').fill(password);
    await page.locator('#confirmPassword').fill(password);

    await page.getByRole('button', { name: /Create account/i }).click();

    // Wait for redirect away from /sign-up
    await expect(page).not.toHaveURL(/\/sign-up/, { timeout: 15_000 });

    signedUp = true;
  });

  test('deploy dialog opens with node selector', async ({ page }) => {
    test.skip(!signedUp, 'Signup did not complete');

    // Use infra user if available (has real node), otherwise use test user
    const useInfra = !!infra;

    // Ensure a published template exists — create via UI if no infra
    let tmplId: string;
    if (useInfra) {
      tmplId = infra!.templateId;
      await logIn(page, infra!.email, TEST_PASSWORD);
    } else {
      // Create template via UI
      await logIn(page, email, password);
      await page.goto('/templates/new');
      await page.waitForLoadState('networkidle');

      await page.getByLabel('Name').fill(uniqueName('deploytest'));
      if (await page.getByLabel('Slug').isVisible()) {
        await page.getByLabel('Slug').fill(uniqueSlug('deploytest'));
      }
      await page.getByLabel('Description').fill('E2E deploy test template');
      await page.getByLabel('Version').fill('1.0.0');
      await page.locator('#compose').fill(TEST_TEMPLATE_COMPOSE);

      await page.getByRole('button', { name: /Create|Save|Submit/i }).click();
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      // Extract template ID from URL
      const match = page.url().match(/\/templates\/([^/]+)/);
      if (!match) {
        test.skip(true, 'Could not create template');
        return;
      }
      tmplId = match[1];
      cleanupIds.templates.push(tmplId);

      // Publish the template
      const publishBtn = page.getByRole('button', { name: /Publish/i });
      if (await publishBtn.isVisible().catch(() => false)) {
        await publishBtn.click();
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(1000);
      }
    }

    // Already logged in from above, navigate to template
    await page.goto(`/templates/${tmplId}`);
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();

    // Deploy dialog should open with node selector and deploy button
    await expect(page.getByText(/Deployment Name/i)).toBeVisible({ timeout: 5_000 });
    await expect(page.getByText(/Deploy To/i)).toBeVisible();
    await expect(page.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();
  });

  test('create SSH key with name and private key', async ({ page }) => {
    test.skip(!signedUp, 'Signup did not complete');

    await logIn(page, email, password);
    await page.goto('/ssh-keys');
    await page.waitForLoadState('networkidle');

    // Click Add SSH Key
    await page.getByRole('button', { name: /Add SSH Key/i }).first().click();
    await page.waitForTimeout(500);

    // Fill in the dialog — backend auto-generates the key pair, just need a name
    const keyName = uniqueName('testkey');
    await page.locator('#key-name').fill(keyName);

    // Private key field is optional — the backend auto-generates if omitted
    // But if visible, fill it with a valid-looking key
    const privateKeyField = page.locator('#private-key');
    if (await privateKeyField.isVisible()) {
      // Use a minimal valid ed25519 key format
      await privateKeyField.fill('-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\nQyNTUxOQAAACCV72vBq6jSMnH54gZOJZgcetAfgu3QE2SijuCFwuKXHwAAAJAHE4oSBxOK\nEgAAAAtzc2gtZWQyNTUxOQAAACCV72vBq6jSMnH54gZOJZgcetAfgu3QE2SijuCFwuKXHw\nAAAECvseKpcxEqiUWRu4bjypSh/R2+mwSMHitTecgttUe/vZXva8GrqNIycfniBk4lmBx6\n0B+C7dATZKKO4IXC4pcfAAAADGUyZUB0ZXN0LmtleQE=\n-----END OPENSSH PRIVATE KEY-----');
    }

    // Submit
    await page.getByRole('button', { name: /Add|Save|Create|Submit/i }).last().click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Key should appear in the list
    await expect(page.getByText(keyName)).toBeVisible({ timeout: 5_000 });
  });

  test('create node with SSH key reference', async ({ page }) => {
    test.skip(!signedUp, 'Signup did not complete');

    await logIn(page, email, password);
    await page.goto('/nodes/new');
    await page.waitForLoadState('networkidle');

    const nodeName = uniqueName('testnode');
    await page.getByLabel('Name').fill(nodeName);
    // Use real droplet IP if available, otherwise use a placeholder
    const sshHost = infra?.dropletIp ?? '203.0.113.1';
    await page.getByLabel(/SSH Host|Host/i).fill(sshHost);
    if (await page.getByLabel(/Port/i).isVisible()) {
      await page.getByLabel(/Port/i).fill('22');
    }
    await page.getByLabel(/User/i).fill('root');

    // Select SSH key from dropdown
    const keySelect = page.getByLabel(/SSH Key/i).or(page.locator('select').first());
    if (await keySelect.isVisible()) {
      const options = keySelect.locator('option');
      const count = await options.count();
      if (count > 1) {
        await keySelect.selectOption({ index: 1 });
      }
    }

    await page.getByRole('button', { name: /Add|Save|Create|Submit/i }).click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Node should appear in the list
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(nodeName)).toBeVisible({ timeout: 15_000 });
  });

  test('deploy to real node — containers actually start', async ({ page }) => {
    test.skip(!infra, 'No real infrastructure from global setup');
    test.setTimeout(300_000);

    // Log in as the infra user (has real node + template)
    await logIn(page, infra!.email, TEST_PASSWORD);

    await page.goto(`/templates/${infra!.templateId}`);
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(1000);

    // Select the real node from global setup
    const nodeSelect = page.locator('#node').or(page.getByLabel(/Deploy To|Node/i));
    if (await nodeSelect.isVisible()) {
      const options = nodeSelect.locator('option');
      const count = await options.count();
      if (count > 1) {
        await nodeSelect.selectOption({ index: 1 });
      }
    }

    // Click Deploy
    const deployBtn = page.getByRole('button', { name: /^Deploy$|^Creating|^Starting/i });
    await deployBtn.click();

    // Wait for redirect to deployment detail
    await expect(page).toHaveURL(/\/deployments\//, { timeout: 15_000 });

    // Track for cleanup
    const match = page.url().match(/\/deployments\/([^/]+)/);
    if (match) cleanupIds.deployments.push(match[1]);

    // Wait for real Running status — containers pull and start on DO droplet
    await expect(page.locator('span').filter({ hasText: /Running/ })).toBeVisible({ timeout: 180_000 });
  });

  // --- Sad path ---

  test('signup with short password shows validation', async ({ page }) => {
    await page.goto('/sign-up');
    // Hoster SPA signup form
    await page.locator('#email').fill(uniqueEmail());
    await page.locator('#password').fill('short');

    // Password requirements should show hint text
    await expect(page.getByText(/At least 8 characters/i)).toBeVisible();

    // Try to submit - should stay on sign-up
    await page.locator('#confirmPassword').fill('short');
    await page.getByRole('button', { name: /Create account/i }).click();
    await page.waitForTimeout(1000);
    // Should still be on sign-up page
    await expect(page).toHaveURL(/\/sign-up/);
  });

  test('signup with existing email shows error', async ({ page }) => {
    // Create a user via browser first
    const existingEmail = uniqueEmail();
    const browser2 = await chromium.launch();
    const ctx2 = await browser2.newContext({ baseURL: 'http://localhost:8082' });
    const page2 = await ctx2.newPage();
    try {
      await signUp(page2, existingEmail, password);
    } finally {
      await browser2.close();
    }

    await page.goto('/sign-up');
    await page.locator('#email').fill(existingEmail);
    await page.locator('#password').fill(password);
    await page.locator('#confirmPassword').fill(password);

    await page.getByRole('button', { name: /Create account/i }).click();
    await page.waitForTimeout(2000);

    // Should show error about existing email or stay on sign-up
    const hasError = await page.getByText(/already exists|already registered|error/i).isVisible().catch(() => false);
    const stillOnSignup = page.url().includes('/sign-up');
    expect(hasError || stillOnSignup).toBeTruthy();
  });

  test('invalid SSH key format shows error', async ({ page }) => {
    test.skip(!signedUp, 'Signup did not complete');

    await logIn(page, email, password);
    await page.goto('/ssh-keys');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /Add SSH Key/i }).first().click();
    await page.waitForTimeout(500);

    await page.locator('#key-name').fill(uniqueName('badkey'));
    await page.locator('#private-key').fill('not-a-valid-ssh-key');

    await page.getByRole('button', { name: /Add|Save|Create|Submit/i }).last().click();
    await page.waitForTimeout(1000);

    // Should show an error about invalid format
    const hasError = await page.getByText(/invalid|error|format/i).isVisible().catch(() => false);
    // Dialog should still be open or error shown
    expect(hasError || await page.locator('#key-name').isVisible()).toBeTruthy();
  });

  test('deploy without selecting node shows error', async ({ page }) => {
    test.skip(!signedUp, 'Signup did not complete');

    await logIn(page, email, password);
    await page.goto('/templates');
    await page.waitForLoadState('networkidle');
    const firstCard = page.locator('a[href^="/templates/tmpl_"]').first();
    test.skip(!(await firstCard.isVisible()), 'No templates');

    await firstCard.click();
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(500);

    // Don't select a node — just try to deploy
    const deployBtn = page.getByRole('button', { name: /^Deploy$/i });
    // If no nodes: button is disabled. If nodes exist but none selected: error on submit.
    if (await deployBtn.isEnabled()) {
      await deployBtn.click();
      await page.waitForTimeout(500);
      await expect(page.locator('.text-destructive').filter({ hasText: /select a node/i })).toBeVisible();
    } else {
      // Button disabled is also correct — no nodes or no selection
      await expect(deployBtn).toBeDisabled();
    }
  });

  test('deploy with invalid name format shows error', async ({ page }) => {
    test.skip(!signedUp, 'Signup did not complete');

    await logIn(page, email, password);
    await page.goto('/templates');
    await page.waitForLoadState('networkidle');
    const firstCard = page.locator('a[href^="/templates/tmpl_"]').first();
    test.skip(!(await firstCard.isVisible()), 'No templates');

    await firstCard.click();
    await page.waitForLoadState('networkidle');
    await page.getByText('Deploy Now').click();
    await page.waitForTimeout(500);

    // Enter an invalid name (leading hyphen)
    const nameInput = page.locator('#name');
    if (await nameInput.isVisible()) {
      await nameInput.clear();
      await nameInput.fill('-invalid-name-');

      // Select a node if available
      const nodeSelect = page.locator('#node');
      if (await nodeSelect.isVisible()) {
        const options = nodeSelect.locator('option');
        const count = await options.count();
        if (count > 1) await nodeSelect.selectOption({ index: 1 });
      }

      const deployBtn = page.getByRole('button', { name: /^Deploy$/i });
      if (await deployBtn.isEnabled()) {
        await deployBtn.click();
        await page.waitForTimeout(500);
        // Should show name format error
        await expect(page.locator('.text-destructive').filter({ hasText: /lowercase|alphanumeric/i })).toBeVisible();
      }
    }
  });
});
