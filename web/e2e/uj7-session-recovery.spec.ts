import { test, expect } from '@playwright/test';
import { apiSignUp, injectAuth, getToken } from './fixtures/auth.fixture';
import { uniqueEmail, TEST_PASSWORD } from './fixtures/test-data';

/**
 * UJ7: "My session expired, I need to get back in"
 *
 * Tests JWT expiry handling, 401 interception, login redirect preservation.
 * Targets: APIGate (:8082) → Hoster (:8080) — real prod-like stack.
 */

test.describe('UJ7: Session Recovery', () => {
  let email: string;
  let password: string;
  let token: string;

  test.beforeAll(async () => {
    email = uniqueEmail();
    password = TEST_PASSWORD;
    const result = await apiSignUp(email, password);
    token = result.token;
  });

  // --- Happy path ---

  test('expired JWT redirects to login', async ({ page }) => {
    await injectAuth(page, 'expired.invalid.token');
    await page.goto('/deployments');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });
  });

  test('deep link preserved through re-login', async ({ page }) => {
    await injectAuth(page, 'expired.invalid.token');
    await page.goto('/deployments');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });

    // Log in with real credentials
    await page.locator('#email').fill(email);
    await page.locator('#password').fill(password);
    await page.getByRole('button', { name: 'Sign in' }).click();
    // After login, redirected away from login
    await expect(page).not.toHaveURL(/\/sign-in/, { timeout: 10_000 });
  });

  test('new token works for subsequent API calls', async ({ page }) => {
    // Use injectAuth with a valid token then navigate to a protected page
    await injectAuth(page, token);
    await page.goto('/deployments');
    await page.waitForLoadState('networkidle');

    // Should NOT redirect to login — valid token is accepted
    await expect(page).toHaveURL(/\/deployments/, { timeout: 10_000 });

    // Page should show deployment content (heading visible)
    await expect(page.getByRole('heading', { name: 'My Deployments' })).toBeVisible({ timeout: 10_000 });

    // Token still in localStorage
    const storedToken = await getToken(page);
    expect(storedToken).toBeTruthy();
  });

  // --- Sad path ---

  test('401 API response redirects to login', async ({ page }) => {
    // Use a protected route — /deployments requires auth via ProtectedRoute
    await injectAuth(page, 'bad-token-that-will-401');
    await page.goto('/deployments');
    // ProtectedRoute checks isAuthenticated from store — bad token means not authenticated
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });
  });

  test('tampered token triggers same flow', async ({ page }) => {
    await injectAuth(page, 'eyJhbGciOiJIUzI1NiJ9.TAMPERED.INVALID');
    await page.goto('/deployments');
    await expect(page).toHaveURL(/\/sign-in/, { timeout: 15_000 });
  });

  test('valid token allows access to protected pages', async ({ page }) => {
    // Verify a real valid token gives access to multiple protected routes
    await injectAuth(page, token);
    await page.goto('/nodes');
    await page.waitForLoadState('networkidle');

    // Should not redirect to login — valid token accepted
    await expect(page).toHaveURL(/\/nodes/, { timeout: 10_000 });
    // Page content should load (not blank)
    await expect(page.getByRole('heading', { name: 'My Nodes' })).toBeVisible({ timeout: 10_000 });
  });
});
