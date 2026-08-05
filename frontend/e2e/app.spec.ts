import { test, expect, type Page } from '@playwright/test'

const ADMIN_EMAIL = 'admin@wago.local'
const ADMIN_PASSWORD = 'test-admin-pass'

async function signIn(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Email').fill(ADMIN_EMAIL)
  await page.getByLabel('Password').fill(ADMIN_PASSWORD)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(/\/inbox/)
}

// Creates a new org through the UI and waits for it to become the active org.
// A unique name keeps tests independent within a shared data dir.
async function createOrg(page: Page, name: string) {
  await page.getByRole('button', { name: 'New organization' }).click()
  await page.getByLabel('Organization name').fill(name)
  await page.getByRole('button', { name: 'Create organization' }).click()
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(page.getByRole('button', { name: new RegExp(`^${name}`) })).toBeVisible()
}

function unique(prefix: string) {
  return `${prefix} ${Math.floor(Math.random() * 1e6)}`
}

test.describe('superuser', () => {
  test('sees and uses the New organization control', async ({ page }) => {
    await signIn(page)

    const newOrgButton = page.getByRole('button', { name: 'New organization' })
    await expect(newOrgButton).toBeVisible()

    await newOrgButton.click()
    await page.getByLabel('Organization name').fill('Acme Corp')
    await page.getByRole('button', { name: 'Create organization' }).click()

    await expect(page.getByRole('dialog')).toBeHidden()
    await expect(page.getByRole('button', { name: /^Acme Corp/ })).toBeVisible()
  })

  test('can manage numbers without an org selected', async ({ page }) => {
    await signIn(page)
    await page.goto('/settings/numbers')
    await expect(page.getByRole('button', { name: /Connect number/i })).toBeVisible()
  })

  test('can invite team members without an org selected', async ({ page }) => {
    await signIn(page)
    await page.goto('/settings/team')
    await expect(page.getByRole('button', { name: /Invite/i })).toBeVisible()
  })

  test('creates and annotates a contact', async ({ page }) => {
    await signIn(page)
    await createOrg(page, unique('Contacts'))

    await page.goto('/contacts')
    const contact = unique('Alice')
    await page.getByRole('button', { name: /Add contact/i }).click()
    await page.getByLabel('Name').fill(contact)
    await page.getByLabel('Phone').fill('+1555000' + String(Math.floor(1000 + Math.random() * 9000)))
    await page.getByRole('button', { name: 'Create' }).click()
    await expect(page.getByRole('dialog')).toBeHidden()

    const row = page.getByRole('row', { name: new RegExp(contact) })
    await expect(row).toBeVisible()

    // Open the detail dialog and add a tag + note.
    await row.click()
    await page.getByPlaceholder('Add a tag…').fill('vip')
    await page.getByRole('button', { name: 'Add', exact: true }).click()
    await expect(page.getByText('vip', { exact: true })).toBeVisible()
    await page
      .getByPlaceholder('Internal notes about this contact…')
      .fill('High priority account')
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByText('Saved.')).toBeVisible()
  })

  test('creates a team', async ({ page }) => {
    await signIn(page)
    await createOrg(page, unique('Team'))

    await page.goto('/settings/team')
    await page.getByRole('button', { name: /Add team/i }).click()
    const team = unique('Marketing')
    await page.getByLabel('Name').fill(team)
    await page.getByRole('button', { name: 'Create' }).click()
    await expect(page.getByRole('dialog')).toBeHidden()
    await expect(page.getByText(team, { exact: true })).toBeVisible()
  })

  test('invites a member as an owner without a team', async ({ page }) => {
    await signIn(page)
    await createOrg(page, unique('Invite'))

    await page.goto('/settings/team')
    await page.getByRole('button', { name: /Invite member/i }).click()

    await page.getByLabel('Email').fill('teammate@example.com')
    await page.getByRole('button', { name: 'Agent', exact: true }).click()
    await page.getByRole('option', { name: 'Owner' }).click()
    await page.getByRole('button', { name: 'Create invite' }).click()

    await expect(page.getByText('Invite sent to')).toBeVisible()
    await expect(page.getByText(/\/join\?t=/)).toBeVisible()
  })

  test('signs out', async ({ page }) => {
    await signIn(page)
    await page.getByRole('button', { name: 'Log out' }).click()
    await expect(page).toHaveURL(/\/login/)
    await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
  })
})