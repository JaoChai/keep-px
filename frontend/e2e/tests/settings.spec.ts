import { test, expect } from '../fixtures/auth.fixture'
import { SettingsPage } from '../pages/settings.page'

test.describe('Settings', () => {
  test('profile info section visible', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()

    await expect(settingsPage.heading).toBeVisible()
    await expect(settingsPage.profileSection).toBeVisible()
    await expect(settingsPage.nameInput).toBeVisible()
    await expect(settingsPage.emailInput).toBeVisible()
  })

  test('API key section visible', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()

    await expect(settingsPage.apiKeySection).toBeVisible()
    await expect(page.getByText('API Key สำหรับรับข้อมูลอีเวนต์')).toBeVisible()
  })
})
