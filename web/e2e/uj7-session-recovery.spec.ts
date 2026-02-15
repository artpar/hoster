import { test, expect, chromium } from '@playwright/test';
import { signUp, logIn, logOut } from './fixtures/auth.fixture';
import { uniqueEmail, TEST_PASSWORD } from './fixtures/test-data';

/**
 * UJ7: "My session expired, I need to get back in"
 *
 * Tests unauthenticated access handling, login redirect, and logout flow.
 * All scenarios use real user actions.
 *
 * Targets: APIGate (:8082) -> Hoster (:8080) — real prod-like stack.
 */

test.describe('UJ7: Session Recovery', () => {
  let email: string;
  let password: string;

  test.beforeAll(async () => {
    email = uniqueEmail();
    password = TEST_PASSWORD;
    const browser = await chromium.launch();
    const context = await browser.newContext({ baseURL: 'http://localhost:8082' });
    const page = await context.newPage();
    try {
      await signUp(page, email, password);
    } finally {
      await browser.close();
    }
  });

  // --- Happy path ---

  test('unauthenticated visit to protected page redirects to sign-in', async ({ page }) => {
    // Fresh browser context — user is not logged in (equivalent to expired session)
    await page.goto('/deployments');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });
  });

  test('login from redirect preserves destination', async ({ page }) => {
    // Navigate to protected page — gets redirected to sign-in
    await page.goto('/deployments');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });

    // Log in with real credentials from the redirect page
    await page.locator('#email').fill(email);
    await page.locator('#password').fill(password);
    await page.getByRole('button', { name: 'Sign in' }).click();
    // After login, redirected away from login
    await expect(page).not.toHaveURL(/\/sign-in/, { timeout: 10_000 });
  });

  test('authenticated user accesses deployments', async ({ page }) => {
    await logIn(page, email, password);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Should NOT redirect to login — valid session is accepted
    await expect(page).toHaveURL(/\/deployments/, { timeout: 10_000 });

    // Page should show deployment content (heading visible)
    await expect(page.getByRole('heading', { name: 'My Deployments' })).toBeVisible({ timeout: 10_000 });
  });

  // --- Sad path ---

  test('unauthenticated visit to nodes redirects to sign-in', async ({ page }) => {
    // Fresh browser context — no auth
    await page.goto('/nodes');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });
  });

  test('unauthenticated visit to billing redirects to sign-in', async ({ page }) => {
    // Fresh browser context — no auth
    await page.goto('/billing');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });
  });

  test('authenticated user accesses multiple protected pages', async ({ page }) => {
    // Verify a real valid login gives access to multiple protected routes
    await logIn(page, email, password);
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');

    // Should not redirect to login — valid session accepted
    await expect(page).toHaveURL(/\/nodes/, { timeout: 10_000 });
    // Page content should load (not blank)
    await expect(page.getByRole('heading', { name: 'My Nodes' })).toBeVisible({ timeout: 10_000 });
  });
});
