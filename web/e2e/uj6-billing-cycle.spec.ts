import { test, expect } from '@playwright/test';
import { apiSignUp, injectAuth } from './fixtures/auth.fixture';
import { uniqueEmail, TEST_PASSWORD } from './fixtures/test-data';

/**
 * UJ6: "I want to review my costs and pay"
 *
 * Customer reviews billing summary, invoices, running deployment costs,
 * and payment flow. All tests use real API responses — no mocking.
 *
 * Targets: APIGate (:8082) → Hoster (:8080) — real prod-like stack.
 */

test.describe('UJ6: Billing Cycle', () => {
  let token: string;

  test.beforeAll(async () => {
    const email = uniqueEmail();
    const result = await apiSignUp(email, TEST_PASSWORD);
    token = result.token;
  });

  // --- Happy path ---

  test('billing page shows summary cards', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // Title
    await expect(page.getByText('Billing & Usage')).toBeVisible();

    // Summary cards
    await expect(page.getByRole('heading', { name: 'Monthly Cost' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Active Deployments' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Usage Events' })).toBeVisible();
  });

  test('running deployments section visible', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // Running deployments section
    await expect(page.getByRole('heading', { name: 'Running Deployments' })).toBeVisible();
    // For a new user: "No running deployments" with link to browse templates
    const hasDeployments = await page.locator('a[href^="/deployments/"]').isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No running deployments').isVisible().catch(() => false);
    expect(hasDeployments || hasEmpty).toBeTruthy();
  });

  test('invoices section visible', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // Invoices section heading
    await expect(page.getByRole('heading', { name: 'Invoices' })).toBeVisible();

    // For a new user: either real invoices or "No invoices yet"
    const hasInvoices = await page.locator('.rounded-md.border.p-4').isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No invoices yet').isVisible().catch(() => false);
    expect(hasInvoices || hasEmpty).toBeTruthy();
  });

  test('usage history section visible', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // Usage History section heading
    await expect(page.getByRole('heading', { name: 'Usage History' })).toBeVisible();
    // Wait for content to render — either event entries or empty state
    await page.waitForTimeout(1000);
    const hasEvents = await page.locator('.rounded-md.border.p-3').first().isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No usage events recorded yet').isVisible().catch(() => false);
    expect(hasEvents || hasEmpty).toBeTruthy();
  });

  // --- Sad path ---

  test('new user has no invoices', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // A fresh user with no deployments should have no invoices
    await expect(page.getByText('No invoices yet')).toBeVisible({ timeout: 10_000 });
  });

  test('payment cancelled shows error', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing?payment=cancelled');
    await page.waitForLoadState('networkidle');

    // Should show "Payment was cancelled." error
    await expect(page.getByText('Payment was cancelled.')).toBeVisible({ timeout: 5_000 });
  });

  test('new user monthly cost is $0', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/billing');
    await page.waitForLoadState('networkidle');

    // For a new user with no deployments, monthly cost should be $0.00
    await expect(page.getByText('$0.00')).toBeVisible({ timeout: 5_000 });
  });
});
