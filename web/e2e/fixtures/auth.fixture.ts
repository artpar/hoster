import { type Page, expect } from '@playwright/test';

const BASE = 'http://localhost:8082';

/**
 * Sign up via the UI form at /sign-up.
 * Returns the JWT token from localStorage after successful signup.
 */
export async function signUp(page: Page, email: string, password: string): Promise<string> {
  await page.goto('/sign-up');
  await page.locator('#email').fill(email);
  await page.locator('#password').fill(password);
  await page.locator('#confirmPassword').fill(password);
  await page.getByRole('button', { name: 'Create account' }).click();
  // Wait for redirect away from /sign-up
  await expect(page).not.toHaveURL(/\/sign-up/);
  const token = await getToken(page);
  expect(token).toBeTruthy();
  return token!;
}

/**
 * Log in via the UI form at /sign-in.
 * Returns the JWT token from localStorage after successful login.
 */
export async function logIn(page: Page, email: string, password: string): Promise<string> {
  await page.goto('/sign-in');
  await page.locator('#email').fill(email);
  await page.locator('#password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  // Wait for redirect away from /sign-in
  await expect(page).not.toHaveURL(/\/sign-in/);
  const token = await getToken(page);
  expect(token).toBeTruthy();
  return token!;
}

/**
 * Log out by clicking the sidebar "Sign Out" action.
 * Waits for redirect to the landing page.
 */
export async function logOut(page: Page): Promise<void> {
  // The sidebar has a Sign Out button/link
  await page.getByText('Sign Out').click();
  await page.waitForURL('/');
}

/**
 * Inject auth token directly into localStorage (skip UI login).
 * Useful for test setup where login is not the thing being tested.
 */
export async function injectAuth(page: Page, token: string, user?: Record<string, unknown>): Promise<void> {
  await page.goto('/');
  await page.evaluate(
    ({ token, user }) => {
      const state = {
        state: {
          user: user ?? { id: 'test-user', email: 'test@test.local', name: 'test', plan_id: 'free', plan_limits: { max_deployments: 10, max_cpu_cores: 4, max_memory_mb: 4096, max_disk_gb: 50 } },
          token,
          isAuthenticated: true,
        },
        version: 0,
      };
      localStorage.setItem('hoster-auth', JSON.stringify(state));
    },
    { token, user },
  );
  await page.reload();
}

/**
 * Clear auth state from localStorage and reload.
 */
export async function clearAuth(page: Page): Promise<void> {
  await page.evaluate(() => localStorage.removeItem('hoster-auth'));
  await page.reload();
}

/**
 * Read the JWT token from the Zustand persisted auth store in localStorage.
 */
export async function getToken(page: Page): Promise<string | null> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('hoster-auth');
    if (!raw) return null;
    try {
      const parsed = JSON.parse(raw);
      return parsed?.state?.token ?? null;
    } catch {
      return null;
    }
  });
}

/**
 * Sign up via the API directly (no browser interaction).
 * Useful for beforeAll data setup.
 */
export async function apiSignUp(email: string, password: string): Promise<{ token: string; user: Record<string, unknown> }> {
  const res = await fetch(`${BASE}/mod/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, name: email.split('@')[0] }),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API signup failed (${res.status}): ${err}`);
  }
  const data = await res.json();
  return { token: data.token, user: data.user ?? {} };
}

/**
 * Log in via the API directly (no browser interaction).
 */
export async function apiLogIn(email: string, password: string): Promise<{ token: string }> {
  const res = await fetch(`${BASE}/mod/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(`API login failed (${res.status}): ${err}`);
  }
  const data = await res.json();
  return { token: data.token };
}
