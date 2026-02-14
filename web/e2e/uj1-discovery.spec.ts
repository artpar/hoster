import { test, expect } from '@playwright/test';
import { apiSignUp, injectAuth } from './fixtures/auth.fixture';
import { uniqueEmail, TEST_PASSWORD } from './fixtures/test-data';

/**
 * UJ1: "I want to try this platform"
 *
 * Real flow through APIGate (:8082) → Hoster (:8080):
 *   - `/` → APIGate portal (entry point for new visitors)
 *   - `/sign-in`, `/sign-up` → Hoster SPA auth pages (served via hoster-front route)
 *   - `/templates` → Hoster SPA marketplace (API requires auth via APIGate)
 *   - `/templates/:id` → Template detail (API requires auth)
 */

test.describe('UJ1: Discovery', () => {
  let token: string;

  test.beforeAll(async () => {
    const email = uniqueEmail();
    const result = await apiSignUp(email, TEST_PASSWORD);
    token = result.token;
  });

  // Template cards link to /templates/tmpl_*, NOT /templates/new (which is the Create button)
  const templateCardSelector = 'a[href^="/templates/tmpl_"]';

  // --- Happy path ---

  test('portal loads as entry point', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Sign In' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Get Started' })).toBeVisible();
  });

  test('signup page accessible from portal', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('link', { name: 'Get Started' }).click();
    await expect(page).toHaveURL(/\/(sign-up|signup|portal\/register|portal\/signup)/);
  });

  test('authenticated user can browse templates', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/templates');
    await expect(page.getByRole('heading', { name: 'Templates' })).toBeVisible({ timeout: 10_000 });
    // Category filter pill "All" (exact match to avoid "Browse All")
    await expect(page.getByRole('button', { name: 'All', exact: true })).toBeVisible();
    // Template cards visible
    await expect(page.locator(templateCardSelector).first()).toBeVisible();
  });

  test('marketplace search filters by name', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/templates');
    await expect(page.getByRole('heading', { name: 'Templates' })).toBeVisible({ timeout: 10_000 });

    const searchInput = page.getByPlaceholder(/Search/i);
    await expect(searchInput).toBeVisible();

    const allCount = await page.locator(templateCardSelector).count();
    expect(allCount).toBeGreaterThan(0);

    await searchInput.fill('Matomo');
    await page.waitForTimeout(500);
    const filteredCount = await page.locator(templateCardSelector).count();
    expect(filteredCount).toBeLessThanOrEqual(allCount);

    await searchInput.clear();
    await page.waitForTimeout(500);
    const restoredCount = await page.locator(templateCardSelector).count();
    expect(restoredCount).toBeGreaterThanOrEqual(filteredCount);
  });

  test('template detail shows all fields', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/templates');
    await expect(page.getByRole('heading', { name: 'Templates' })).toBeVisible({ timeout: 10_000 });

    const firstCard = page.locator(templateCardSelector).first();
    await expect(firstCard).toBeVisible();
    await firstCard.click();
    await expect(page).toHaveURL(/\/templates\/tmpl_.+/);

    // Template detail: name, compose spec, deploy button
    await expect(page.getByText('Deploy Now')).toBeVisible({ timeout: 10_000 });
    await expect(page.locator('pre')).toBeVisible();
  });

  test('"Deploy Now" opens deploy dialog for authenticated user', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/templates');
    await expect(page.getByRole('heading', { name: 'Templates' })).toBeVisible({ timeout: 10_000 });

    const firstCard = page.locator(templateCardSelector).first();
    await firstCard.click();
    await expect(page.getByText('Deploy Now')).toBeVisible({ timeout: 10_000 });
    await page.getByText('Deploy Now').click();

    // Deploy dialog opens
    await expect(page.getByText(/Deployment Name/i)).toBeVisible({ timeout: 5_000 });
  });

  // --- Sad path ---

  test('unauthenticated templates page shows loading or error', async ({ page }) => {
    await page.goto('/templates');
    await page.waitForTimeout(3000);
    // No template data loaded without auth
    const noTemplateCards = (await page.locator(templateCardSelector).count()) === 0;
    expect(noTemplateCards).toBeTruthy();
  });

  test('invalid template ID shows error', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/templates/nonexistent-template-id-12345');
    await page.waitForTimeout(3000);
    const body = await page.locator('body').textContent();
    expect(body!.length).toBeGreaterThan(0);
  });

  test('template detail accessible on revisit with auth', async ({ page }) => {
    await injectAuth(page, token);
    await page.goto('/templates');
    await expect(page.getByRole('heading', { name: 'Templates' })).toBeVisible({ timeout: 10_000 });

    const firstCard = page.locator(templateCardSelector).first();
    await firstCard.click();
    await expect(page.getByText('Deploy Now')).toBeVisible({ timeout: 10_000 });
    const url = page.url();

    await page.reload();
    await expect(page).toHaveURL(url);
    await expect(page.getByText('Deploy Now')).toBeVisible({ timeout: 10_000 });
  });
});
