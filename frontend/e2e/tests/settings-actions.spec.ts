import { test, expect } from '../fixtures/auth.fixture'
import { SettingsPage } from '../pages/settings.page'

test.describe('Settings Actions', () => {
  test('profile shows plan info', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()

    await expect(settingsPage.heading).toBeVisible()
    // Plan label and input (Label lacks htmlFor, use sibling combinator)
    const planInput = page.locator('label:has-text("แพลน") + input')
    await expect(planInput).toBeVisible()
    await expect(planInput).toHaveValue(/.+/)
  })

  test('API key show/hide toggle works', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()

    await expect(settingsPage.apiKeySection).toBeVisible()

    // API key should be masked by default (bullet characters)
    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible()
    const maskedValue = await apiKeyInput.inputValue()
    expect(maskedValue).toContain('•')

    // Click show button (Eye icon) — first button with eye icon in API key section
    const showBtn = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await showBtn.click()

    // Key should now be visible (no bullets)
    const visibleValue = await apiKeyInput.inputValue()
    expect(visibleValue).not.toContain('•')

    // Click hide button (EyeOff icon)
    await page.locator('button', { has: page.locator('[class*="lucide-eye-off"]') }).click()

    // Key should be masked again
    const remaskedValue = await apiKeyInput.inputValue()
    expect(remaskedValue).toContain('•')
  })

  test('regenerate API key dialog opens and can cancel', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()

    // Click "สร้างคีย์ใหม่" button
    await page.getByRole('button', { name: 'สร้างคีย์ใหม่' }).click()

    // Confirmation dialog should appear
    await expect(page.getByRole('heading', { name: 'สร้าง API Key ใหม่' })).toBeVisible()
    await expect(page.getByText('คีย์เดิมจะใช้ไม่ได้ทันที')).toBeVisible()

    // Cancel should close dialog
    await page.getByRole('button', { name: 'ยกเลิก' }).click()
    await expect(page.getByRole('heading', { name: 'สร้าง API Key ใหม่' })).not.toBeVisible()
  })

  test('API key helper text visible', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()

    await expect(page.getByText('คีย์นี้ใช้สำหรับเทมเพลตเซลเพจ')).toBeVisible()
  })
})
