import { defineConfig, devices } from '@playwright/test'

const isCI = !!process.env.CI
const useExternalURL = !!process.env.E2E_BASE_URL

export default defineConfig({
  testDir: './e2e/tests',
  fullyParallel: !useExternalURL,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  workers: isCI || useExternalURL ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    trace: 'on-first-retry',
    video: isCI ? 'on-first-retry' : 'off',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  globalSetup: './e2e/support/global-setup.ts',
  ...(!useExternalURL && {
    webServer: [
      {
        command: 'cd ../backend && go run ./cmd/server',
        url: 'http://localhost:8080/health',
        reuseExistingServer: !isCI,
        timeout: 120_000,
      },
      {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !isCI,
        timeout: 30_000,
      },
    ],
  }),
})
