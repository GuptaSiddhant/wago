import AxeBuilder from '@axe-core/playwright'
import { test, expect, type Page } from '@playwright/test'

const ADMIN_EMAIL = 'admin@wago.local'
const ADMIN_PASSWORD = 'test-admin-pass'

async function scan(page: Page, path: string) {
  // The app keeps realtime sockets open, so 'networkidle' never settles.
  await page.goto(path, { waitUntil: 'load' })
  await page.waitForTimeout(750)
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze()
  const serious = results.violations.filter((v) =>
    ['critical', 'serious'].includes(v.impact ?? ''),
  )
  if (results.violations.length > serious.length) {
    console.warn(
      `[a11y] ${path}: ${results.violations.length - serious.length} minor violation(s)`,
    )
  }
  expect(
    serious,
    `${path} has serious/critical accessibility violations: ${serious
      .map((v) =>
        v.nodes
          .slice(0, 3)
          .map((n) => `${v.id}@${n.target.join(' ')} ${n.failureSummary ?? ''}`)
          .join(' | '),
      )
      .join(' ;; ')}`,
  ).toEqual([])
}

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN_EMAIL)
  await page.getByLabel('Password').fill(ADMIN_PASSWORD)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(/\/(inbox|select-org)/)
  if (!page.url().includes('/inbox')) {
    // Fresh data dir: no organizations yet. Create one to reach the app shell.
    await page.getByLabel('Organization name').fill('Axe Org')
    await page.getByRole('button', { name: 'Create & continue' }).click()
    await expect(page).toHaveURL(/\/inbox/)
  }
}

test.describe('public pages', () => {
  test('login has no serious violations', async ({ page }) => {
    await scan(page, '/login')
  })

  test('select-org has no serious violations', async ({ page }) => {
    await signIn(page)
    await scan(page, '/select-org')
    await page.getByRole('button', { name: 'Log out' }).first().click()
    await expect(page).toHaveURL(/\/login/)
  })
})

test.describe('authenticated pages', () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page)
  })

  for (const path of [
    '/inbox',
    '/contacts',
    '/broadcast',
    '/templates',
    '/analytics',
    '/settings/org',
    '/settings/team',
    '/settings/numbers',
    '/account',
  ]) {
    test(`${path} has no serious violations`, async ({ page }) => {
      await scan(page, path)
    })
  }

  test('instance settings have no serious violations', async ({ page }) => {
    // Admin-only route; the seeded e2e user is a superuser.
    await scan(page, '/settings/config')
  })
})
