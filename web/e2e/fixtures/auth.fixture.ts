import { type Page, expect } from '@playwright/test';

/**
 * Sign up via the UI form at /sign-up.
 * After successful signup, the user is authenticated in the browser session.
 */
export async function signUp(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/sign-up');
  await page.locator('#email').fill(email);
  await page.locator('#password').fill(password);
  await page.locator('#confirmPassword').fill(password);
  await page.getByRole('button', { name: 'Create account' }).click();
  // Wait for redirect away from /sign-up
  await expect(page).not.toHaveURL(/\/sign-up/, { timeout: 15_000 });
}

/**
 * Log in via the UI form at /sign-in.
 * After successful login, the user is authenticated in the browser session.
 */
export async function logIn(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/sign-in');
  await page.locator('#email').fill(email);
  await page.locator('#password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  // Wait for redirect away from /sign-in
  await expect(page).not.toHaveURL(/\/sign-in/, { timeout: 15_000 });
}

/**
 * Log out by clicking the sidebar "Sign Out" action.
 * Waits for redirect to the landing page.
 */
export async function logOut(page: Page): Promise<void> {
  await page.getByText('Sign Out').click();
  await page.waitForURL('/');
}
