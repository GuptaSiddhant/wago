import { defineConfig, devices } from '@playwright/test'
import { existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = fileURLToPath(new URL('../', import.meta.url))
const dataDir = path.join(repoRoot, 'frontend', 'e2e', '.pb_data')
const systemChromium = '/usr/bin/chromium'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: 'http://127.0.0.1:8091',
    headless: true,
    trace: 'retain-on-failure',
    launchOptions: {
      executablePath: existsSync(systemChromium) ? systemChromium : undefined,
    },
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: `rm -rf ${dataDir} && nub run build && ./tmp/wago serve --http=127.0.0.1:8091 --dir=${dataDir}`,
    cwd: repoRoot,
    url: 'http://127.0.0.1:8091/login',
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    env: {
      ...process.env,
      ADMIN_EMAIL: 'admin@wago.local',
      ADMIN_PASSWORD: 'test-admin-pass',
      WA_WEBHOOK_VERIFY_TOKEN: 'test-verify-token',
    },
  },
})
