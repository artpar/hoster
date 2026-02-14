import { defineConfig } from '@playwright/test';

/**
 * E2E tests run against the real prod-like local stack:
 *   Browser -> APIGate (:8082) -> Hoster (:8080)
 *
 * Both APIGate and Hoster must be running before tests start.
 * See specs/local-e2e-setup.md for setup instructions.
 *
 * Global setup provisions a real DigitalOcean droplet shared by all tests.
 * Global teardown destroys the droplet and cleans up.
 */
export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  globalTeardown: './e2e/global-teardown.ts',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 1,
  reporter: [['html', { open: 'never' }], ['list']],
  use: {
    baseURL: 'http://localhost:8082',
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
});
