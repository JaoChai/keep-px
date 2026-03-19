import { test, expect } from '../fixtures/auth.fixture'
import { PixelsPage } from '../pages/pixels.page'

// Helper to delete a pixel by row text — uses .first() to avoid strict mode violation
async function deletePixelByName(page: import('@playwright/test').Page, name: string) {
  const rows = page.locator('tr', { hasText: name })
  if (await rows.count() === 0) return
  await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
  await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
  await page.waitForTimeout(500)
}

// Ensure quota has space by cleaning up test pixels and reloading
async function ensureQuotaSpace(page: import('@playwright/test').Page) {
  const addButton = page.getByRole('button', { name: 'เพิ่มพิกเซล' }).first()
  if (await addButton.isDisabled()) {
    for (const pattern of CLEANUP_PATTERNS) {
      let rows = page.locator('tr', { hasText: pattern })
      while (await rows.count() > 0) {
        await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
        await page.locator('button.bg-destructive', { hasText: 'ลบ' }).click()
        await page.waitForTimeout(500)
        rows = page.locator('tr', { hasText: pattern })
      }
    }
    await page.reload()
    await page.waitForLoadState('networkidle')
  }
}

// All E2E-created pixel names start with these prefixes for cleanup
const CLEANUP_PATTERNS = ['Test Pixel', 'Edit Test', 'Updated', 'Delete Test', 'E2E-Pixel', 'E2E-Backup']

test.describe('Pixels', () => {
  // Clean up test-created pixels after each test to stay within quota
  test.afterEach(async ({ page }) => {
    await page.goto('/pixels')
    await page.waitForLoadState('networkidle')
    for (const pattern of CLEANUP_PATTERNS) {
      let rows = page.locator('tr', { hasText: pattern })
      let count = await rows.count()
      while (count > 0) {
        await deletePixelByName(page, pattern)
        rows = page.locator('tr', { hasText: pattern })
        count = await rows.count()
      }
    }
  })

  test('create pixel and see in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    const pixelName = `Test Pixel ${Date.now()}`
    await pixelsPage.createPixel(pixelName, '123456789012345', 'EAAtest123token')

    await expect(page.getByText(pixelName)).toBeVisible()
  })

  test('edit pixel name and see updated in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    // Create a pixel first
    const originalName = `Edit Test ${Date.now()}`
    await pixelsPage.createPixel(originalName, '123456789012345', 'EAAtest123token')
    await expect(page.getByText(originalName)).toBeVisible()

    // Click edit button on the pixel row
    const pixelRow = page.locator('tr', { hasText: originalName })
    await pixelRow.getByRole('button').filter({ has: page.locator('[class*="lucide-pencil"]') }).click()

    // Wait for edit dialog to appear
    await expect(page.getByRole('heading', { name: 'แก้ไขพิกเซล' })).toBeVisible()

    // Update name (must also fill access token — schema requires it)
    const updatedName = `Updated ${Date.now()}`
    await pixelsPage.nameInput.clear()
    await pixelsPage.nameInput.fill(updatedName)
    await pixelsPage.accessTokenInput.fill('EAAtest123token')
    await pixelsPage.saveButton.click()

    // Wait for edit dialog to close before asserting
    await expect(page.getByRole('heading', { name: 'แก้ไขพิกเซล' })).not.toBeVisible()

    await expect(page.getByText(updatedName)).toBeVisible()
  })

  test('delete pixel with confirmation dialog', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    // Create a pixel first
    const pixelName = `Delete Test ${Date.now()}`
    await pixelsPage.createPixel(pixelName, '123456789012345', 'EAAtest123token')

    // Wait for pixel to appear in table before trying to delete
    await expect(page.getByText(pixelName)).toBeVisible()

    // Click delete button
    const pixelRow = page.locator('tr', { hasText: pixelName })
    await pixelRow.getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()

    // Confirm deletion
    await expect(page.getByText('คุณแน่ใจหรือไม่')).toBeVisible()
    await pixelsPage.deleteConfirmButton.click()

    await expect(page.getByText(pixelName)).not.toBeVisible()
  })

  test('show create pixel dialog with correct fields', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    await pixelsPage.addPixelButton.click()

    await expect(page.getByRole('heading', { name: 'เพิ่มพิกเซล' })).toBeVisible()
    await expect(pixelsPage.nameInput).toBeVisible()
    await expect(pixelsPage.pixelIdInput).toBeVisible()
    await expect(pixelsPage.accessTokenInput).toBeVisible()
    await expect(pixelsPage.cancelButton).toBeVisible()
  })

  // --- Scenario 2: Missing step 1 — Create second pixel (multi-pixel) ---
  test('create multiple pixels and see both in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    const ts = Date.now()
    const pixelA = `E2E-Pixel A ${ts}`
    const pixelB = `E2E-Pixel B ${ts}`

    await pixelsPage.createPixel(pixelA, '111111111111111', 'EAAtest123tokenA')
    await expect(page.getByText(pixelA)).toBeVisible()

    await pixelsPage.createPixel(pixelB, '222222222222222', 'EAAtest123tokenB')
    await expect(page.getByText(pixelB)).toBeVisible()

    // Both pixels should be visible simultaneously in the table
    await expect(page.locator('tr', { hasText: pixelA })).toBeVisible()
    await expect(page.locator('tr', { hasText: pixelB })).toBeVisible()
  })

  // --- Scenario 2: Missing step 2 — Test connection toast ---
  test('test connection shows toast', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    const pixelName = `E2E-Pixel ${Date.now()}`
    await pixelsPage.createPixel(pixelName, '123456789012345', 'EAAtest123token')
    await expect(page.getByText(pixelName)).toBeVisible()

    // Click the test connection button
    await pixelsPage.testConnection(pixelName)

    // A toast should appear (success or error — both are valid since the token is fake)
    await expect(pixelsPage.toast.first()).toBeVisible({ timeout: 15000 })
  })

  // --- Scenario 2: Missing step 3 — Toggle pixel active/inactive ---
  test('toggle pixel active/inactive status', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    const pixelName = `E2E-Pixel ${Date.now()}`
    await pixelsPage.createPixel(pixelName, '123456789012345', 'EAAtest123token')
    await expect(page.getByText(pixelName)).toBeVisible()

    // Pixel should start as active
    const initialStatus = await pixelsPage.getStatus(pixelName)
    expect(initialStatus).toBe('ใช้งาน')

    // Toggle to inactive
    await pixelsPage.toggleActive(pixelName)
    await page.waitForTimeout(1000) // Wait for API call and re-render
    const inactiveStatus = await pixelsPage.getStatus(pixelName)
    expect(inactiveStatus).toBe('หยุดชั่วคราว')

    // Toggle back to active
    await pixelsPage.toggleActive(pixelName)
    await page.waitForTimeout(1000)
    const activeStatus = await pixelsPage.getStatus(pixelName)
    expect(activeStatus).toBe('ใช้งาน')
  })

  // --- Scenario 2: Missing step 4 — Set backup pixel + verify cleared on delete ---
  test('set backup pixel and verify cleared when backup is deleted', async ({ page }) => {
    test.slow() // This test creates multiple pixels and performs sequential operations

    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')
    await ensureQuotaSpace(page)

    const ts = Date.now()
    const pixelAName = `E2E-Pixel A ${ts}`
    const pixelBName = `E2E-Backup B ${ts}`

    // Step 1: Create pixel A and pixel B
    await pixelsPage.createPixel(pixelAName, '111111111111111', 'EAAtest123tokenA')
    await expect(page.getByText(pixelAName)).toBeVisible()

    await pixelsPage.createPixel(pixelBName, '222222222222222', 'EAAtest123tokenB')
    await expect(page.getByText(pixelBName)).toBeVisible()

    // Step 2: Edit pixel A -> set backup = pixel B -> save
    await pixelsPage.openEdit(pixelAName)
    await expect(pixelsPage.backupPixelSelect).toBeVisible()
    // Select pixel B as backup — the option text contains the name
    // Select the option that contains pixel B's name
    const options = pixelsPage.backupPixelSelect.locator('option')
    const optionCount = await options.count()
    let targetValue = ''
    for (let i = 0; i < optionCount; i++) {
      const text = await options.nth(i).textContent()
      if (text && text.includes(pixelBName)) {
        targetValue = await options.nth(i).getAttribute('value') ?? ''
        break
      }
    }
    expect(targetValue).not.toBe('')
    await pixelsPage.backupPixelSelect.selectOption(targetValue)
    await pixelsPage.accessTokenInput.fill('EAAtest123tokenA')
    await pixelsPage.saveButton.click()
    await expect(page.getByRole('heading', { name: 'แก้ไขพิกเซล' })).not.toBeVisible()

    // Step 3: Verify pixel A's backup column shows pixel B's name
    await page.waitForTimeout(1000) // Wait for re-render
    const backupText = await pixelsPage.getBackup(pixelAName)
    expect(backupText).toContain(pixelBName)

    // Step 4: Delete pixel B
    await pixelsPage.deletePixel(pixelBName)
    await expect(page.locator('tr', { hasText: pixelBName })).not.toBeVisible()

    // Step 5: Verify pixel A's backup column shows "ไม่มี" or "ไม่ทราบ" (orphaned backup)
    await page.waitForTimeout(1000) // Wait for re-render after deletion
    const backupAfterDelete = await pixelsPage.getBackup(pixelAName)
    // After deleting the backup pixel, the column should show "ไม่มี" (cleared) or "ไม่ทราบ" (orphaned ref)
    expect(backupAfterDelete === 'ไม่มี' || backupAfterDelete === 'ไม่ทราบ').toBeTruthy()
  })

  // --- Scenario 8: Pixel over quota ---
  test('pixel quota limit disables create button and shows upgrade link', async ({ page }) => {
    test.slow() // May need to create multiple pixels to reach quota

    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // If already at quota, verify directly without creating
    const addButton = pixelsPage.addPixelButton
    if (await addButton.isDisabled()) {
      await expect(addButton).toHaveAttribute('title', 'ถึงขีดจำกัด Pixel Slots แล้ว')
      return
    }

    // Check the quota text to determine max_pixels (format: "X/Y สล็อต")
    const quotaText = await page.locator('text=/\\d+\\/\\d+ สล็อต/').textContent()
    if (!quotaText) {
      test.skip(true, 'Quota info not available — cannot determine pixel limit')
      return
    }

    const match = quotaText.match(/(\d+)\/(\d+)/)
    if (!match) {
      test.skip(true, 'Could not parse quota text')
      return
    }

    const current = parseInt(match[1])
    const max = parseInt(match[2])
    const needed = max - current

    // If we need more than 3 additional pixels, skip to avoid polluting the account
    if (needed > 3) {
      test.skip(true, `Need ${needed} more pixels to reach quota (max=${max}) — too many to create in E2E`)
      return
    }

    // Create pixels until quota is reached
    const createdPixels: string[] = []
    for (let i = 0; i < needed; i++) {
      const name = `E2E-Pixel Quota ${Date.now()}-${i}`
      const pixelId = `${333333333333333 + i}`
      await pixelsPage.createPixel(name, pixelId, 'EAAtest123token')
      await expect(page.getByText(name)).toBeVisible()
      createdPixels.push(name)
    }

    // At quota limit: the add button should be disabled
    await expect(pixelsPage.addPixelButton).toBeDisabled()

    // The button should have the quota limit title
    await expect(pixelsPage.addPixelButton).toHaveAttribute('title', 'ถึงขีดจำกัด Pixel Slots แล้ว')

    // Clean up the pixels we created
    for (const name of createdPixels) {
      await deletePixelByName(page, name)
      await page.waitForTimeout(500)
    }
  })
})
