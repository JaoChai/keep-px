/**
 * Scenario 7: Settings & API Key Security
 *
 * จำลอง user จัดการตั้งค่า: ดูโปรไฟล์, reveal/hide API key, copy, regenerate key,
 * ยืนยันว่า key เก่าใช้ไม่ได้ + key ใหม่ใช้ได้
 * Flow: open settings → profile → API key mask → reveal → hide → copy →
 *       regenerate → confirm → verify new key → API validation
 *
 * WARNING: This test regenerates the API key permanently.
 * Other tests that depend on the API key will need to re-read it from /settings.
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { SettingsPage } from '../../pages/settings.page'

// const PREFIX = 'E2E-S07'

test.describe('Scenario 7: Settings & API Key Security', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(90_000)

  /** Shared state across serial steps */
  let oldKey = ''
  let newKey = ''

  // No cleanup needed — settings page is mostly read-only
  // (API key regeneration is permanent and intentional)

  // --- Step 1: Open /settings → see heading "ตั้งค่า" ---
  test('step 1: open /settings → see heading', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(settingsPage.heading).toBeVisible({ timeout: 15000 })
  })

  // --- Step 2: See profile section → name + email visible ---
  test('step 2: see profile section with name and email', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(settingsPage.heading).toBeVisible({ timeout: 15000 })

    // Profile section should be visible
    await expect(settingsPage.profileSection).toBeVisible()

    // Name and email inputs should be visible
    await expect(settingsPage.nameInput).toBeVisible()
    await expect(settingsPage.emailInput).toBeVisible()
  })

  // --- Step 3: Verify name matches test user (soft check) ---
  test('step 3: verify name field has a value', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    // Name input should have a non-empty value
    const nameValue = await settingsPage.nameInput.inputValue()
    expect(nameValue.length).toBeGreaterThan(0)

    // Email should also have a value
    const emailValue = await settingsPage.emailInput.inputValue()
    expect(emailValue).toContain('@')
  })

  // --- Step 4: See API Key section visible ---
  test('step 4: see API key section', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(settingsPage.apiKeySection).toBeVisible()
  })

  // --- Step 5: API key is masked (contains •) ---
  test('step 5: API key is masked by default', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    // Key should be masked — input value contains bullet characters
    const maskedValue = await apiKeyInput.inputValue()
    expect(maskedValue).toContain('•')
  })

  // --- Step 6: Click reveal (eye button) → key visible, no • characters ---
  test('step 6: click reveal → key visible without mask', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    // Verify masked first
    const maskedValue = await apiKeyInput.inputValue()
    expect(maskedValue).toContain('•')

    // Click eye button to reveal
    const eyeButton = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    // Now key should be revealed — no bullet characters
    const revealedValue = await apiKeyInput.inputValue()
    expect(revealedValue).not.toContain('•')
    expect(revealedValue.length).toBeGreaterThan(5)
  })

  // --- Step 7: Save key value to variable ---
  test('step 7: save revealed key to oldKey', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    // Reveal the key
    const eyeButton = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    oldKey = await apiKeyInput.inputValue()
    expect(oldKey).toBeTruthy()
    expect(oldKey).not.toContain('•')
    expect(oldKey.length).toBeGreaterThan(5)
  })

  // --- Step 8: Click hide → key masked again ---
  test('step 8: click hide → key masked again', async ({ page }) => {
    test.skip(!oldKey, 'No API key from step 7')

    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    // Reveal first
    const eyeButton = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    // Verify revealed
    const revealedValue = await apiKeyInput.inputValue()
    expect(revealedValue).not.toContain('•')

    // Click again to hide (same eye button toggles)
    await eyeButton.click()
    await page.waitForTimeout(500)

    // Now should be masked again
    const maskedValue = await apiKeyInput.inputValue()
    expect(maskedValue).toContain('•')
  })

  // --- Step 9: Click copy button → verify button exists and clickable ---
  test('step 9: copy button exists and is clickable', async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    // Copy button should be visible
    await expect(settingsPage.copyApiKeyButton).toBeVisible()

    // Click copy — should not throw
    await settingsPage.copyApiKeyButton.click()
    await page.waitForTimeout(500)

    // After clicking, a toast or icon change may appear (copied feedback)
    // Just verify the button is still functional (no crash)
    await expect(settingsPage.copyApiKeyButton).toBeVisible()
  })

  // --- Step 10: Click "สร้างคีย์ใหม่" → see confirmation dialog ---
  test('step 10: click regenerate → see confirmation dialog', async ({ page }) => {
    test.skip(!oldKey, 'No API key from step 7')

    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    // Dismiss any existing toasts before clicking
    const toast = page.locator('[data-sonner-toast]')
    if (await toast.count() > 0) {
      await toast.first().click()
      await page.waitForTimeout(300)
    }

    // Find and click the regenerate button
    const regenerateButton = page.getByRole('button', { name: /สร้างคีย์ใหม่|Regenerate/ })
    await expect(regenerateButton).toBeVisible()
    await regenerateButton.click()

    // Custom dialog (no role="dialog") — detect via heading or warning text
    const dialogHeading = page.getByText(/สร้างคีย์ใหม่|ยืนยัน|คีย์เดิม|ใช้ไม่ได้ทันที/i).first()
    await expect(dialogHeading).toBeVisible({ timeout: 5000 })

    // Close the dialog without confirming (we'll confirm in next step)
    const cancelBtn = page.getByRole('button', { name: /ยกเลิก|Cancel/ }).first()
    const cancelVisible = await cancelBtn.isVisible().catch(() => false)
    if (cancelVisible) {
      await cancelBtn.click()
    } else {
      await page.keyboard.press('Escape')
    }
    await page.waitForTimeout(500)
  })

  // --- Step 11: Click confirm in dialog → see toast success ---
  test('step 11: confirm regenerate → toast success', async ({ page }) => {
    test.skip(!oldKey, 'No API key from step 7')

    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    // Dismiss any existing toasts
    const existingToast = page.locator('[data-sonner-toast]')
    if (await existingToast.count() > 0) {
      await existingToast.first().click()
      await page.waitForTimeout(300)
    }

    // Click regenerate button
    const regenerateButton = page.getByRole('button', { name: /สร้างคีย์ใหม่|Regenerate/ })
    await regenerateButton.click()

    // Wait for confirmation dialog (custom, no role="dialog")
    const dialogText = page.getByText(/สร้างคีย์ใหม่|ยืนยัน|คีย์เดิม|ใช้ไม่ได้ทันที/i).first()
    await expect(dialogText).toBeVisible({ timeout: 5000 })

    // Click confirm button
    const confirmBtn = page.getByRole('button', { name: /ยืนยัน|ตกลง|Confirm|OK/ }).first()
    await expect(confirmBtn).toBeVisible()
    await confirmBtn.click()

    // Wait for success toast
    await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 10000 })
  })

  // --- Step 12: Reveal new key → different from oldKey ---
  test('step 12: reveal new key → different from old key', async ({ page }) => {
    test.skip(!oldKey, 'No API key from step 7')

    const settingsPage = new SettingsPage(page)
    await settingsPage.goto()
    await page.waitForLoadState('networkidle')

    const apiKeyInput = page.locator('input.font-mono')
    await expect(apiKeyInput).toBeVisible({ timeout: 10000 })

    // Reveal the new key
    const eyeButton = page.locator('button', { has: page.locator('[class*="lucide-eye"]') }).first()
    await eyeButton.click()
    await page.waitForTimeout(500)

    newKey = await apiKeyInput.inputValue()
    expect(newKey).toBeTruthy()
    expect(newKey).not.toContain('•')
    expect(newKey.length).toBeGreaterThan(5)

    // New key must be different from old key
    expect(newKey).not.toBe(oldKey)
  })

  // --- Step 13: Verify old key is invalid, new key works ---
  test('step 13: old key invalid (401), new key works', async ({ page }) => {
    test.skip(!oldKey || !newKey, 'No keys from previous steps')

    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:5173'

    // Navigate to an app page first so localStorage is accessible
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    const dummyEvent = {
      events: [
        {
          pixel_id: '00000000-0000-0000-0000-000000000000',
          event_name: 'Test',
          event_time: new Date().toISOString(),
          event_data: {},
        },
      ],
    }

    // Old key should be rejected (401)
    const oldResp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: { 'X-API-Key': oldKey, 'Content-Type': 'application/json' },
      data: dummyEvent,
    })
    expect(oldResp.status()).toBe(401)

    // New key should be accepted (200/202 success, 402 quota exceeded, or 500 CAPI forward fail)
    const newResp = await page.request.post(`${baseURL}/api/v1/events/ingest`, {
      headers: { 'X-API-Key': newKey, 'Content-Type': 'application/json' },
      data: dummyEvent,
    })
    expect([200, 202, 402, 500]).toContain(newResp.status())
  })
})
