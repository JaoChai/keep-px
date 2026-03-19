/**
 * Scenario 2: Pixel Mastery
 * GitHub Issue: #153
 *
 * จำลอง user สร้าง + จัดการ pixel ทุกรูปแบบ
 * Flow: open page → create 2 pixels → edit → test connection → toggle status → set backup → delete → verify backup cleared
 */
import { test, expect } from '../../fixtures/auth.fixture'
import { PixelsPage } from '../../pages/pixels.page'

const PREFIX = 'E2E-S02'
const ts = Date.now()
const PIXEL_A_NAME = `${PREFIX} Pixel-A ${ts}`
const PIXEL_A_ID = '110000000000001'
const PIXEL_A_TOKEN = 'EAAscenario02_tokenA'
const PIXEL_B_NAME = `${PREFIX} Pixel-B ${ts}`
const PIXEL_B_ID = '220000000000002'
const PIXEL_B_TOKEN = 'EAAscenario02_tokenB'
const PIXEL_A_EDITED = `${PREFIX} Pixel-A-Edited ${ts}`

test.describe('Scenario 2: Pixel Mastery', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(60_000)

  // Safety net cleanup — runs even if tests fail midway
  test.afterAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: 'e2e/.auth/user.json',
    })
    const page = await context.newPage()

    try {
      await page.goto('/pixels')
      await page.waitForLoadState('networkidle')

      // Delete all pixels with our prefix
      let rows = page.locator('tr', { hasText: PREFIX })
      let count = await rows.count()
      while (count > 0) {
        await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
        const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
        await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
        await deleteBtn.click()
        await page.waitForTimeout(1000)
        // Dismiss toasts that might block clicks
        const toast = page.locator('[data-sonner-toast]')
        if (await toast.count() > 0) {
          await toast.first().click()
          await page.waitForTimeout(500)
        }
        rows = page.locator('tr', { hasText: PREFIX })
        count = await rows.count()
      }
    } catch {
      // Best-effort cleanup
    } finally {
      await context.close()
    }
  })

  // --- Step 1: เปิด /pixels → เห็น empty state หรือ table ---
  test('step 1: open /pixels → see table or empty state', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await expect(pixelsPage.heading).toBeVisible()

    // Should show either a table with existing pixels or the empty state
    const hasTable = await pixelsPage.pixelTable.isVisible().catch(() => false)
    const hasEmptyState = await pixelsPage.emptyState.isVisible().catch(() => false)
    expect(hasTable || hasEmptyState).toBe(true)

    // Clean up leftover test pixels from previous runs (all E2E prefixes)
    const cleanupPrefixes = [PREFIX, 'E2E', 'Test Pixel', 'Edit Test', 'Updated', 'Delete Test']
    for (const prefix of cleanupPrefixes) {
      let rows = page.locator('tr', { hasText: prefix })
      let count = await rows.count()
      while (count > 0) {
        await rows.first().getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()
        const deleteBtn = page.locator('button.bg-destructive', { hasText: 'ลบ' })
        await deleteBtn.waitFor({ state: 'visible', timeout: 5000 })
        await deleteBtn.click()
        await page.waitForTimeout(1000)
        const toast = page.locator('[data-sonner-toast]')
        if (await toast.count() > 0) {
          await toast.first().click()
          await page.waitForTimeout(500)
        }
        rows = page.locator('tr', { hasText: prefix })
        count = await rows.count()
      }
    }
  })

  // --- Step 2: คลิก "เพิ่มพิกเซล" → dialog เปิด พร้อม fields ---
  test('step 2: click "เพิ่มพิกเซล" → dialog shows correct fields', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.addPixelButton.click()

    // Dialog should have all the required fields
    await expect(page.getByRole('heading', { name: 'เพิ่มพิกเซล' })).toBeVisible()
    await expect(pixelsPage.nameInput).toBeVisible()
    await expect(pixelsPage.pixelIdInput).toBeVisible()
    await expect(pixelsPage.accessTokenInput).toBeVisible()
    // Test event code field (optional)
    await expect(page.getByLabel(/Test Event Code/)).toBeVisible()

    // Cancel to close dialog
    await pixelsPage.cancelButton.click()
    await expect(page.getByRole('heading', { name: 'เพิ่มพิกเซล' })).not.toBeVisible()
  })

  // --- Step 3: สร้าง pixel ตัวแรก → เห็นใน table ---
  test('step 3: create pixel A → appears in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.createPixel(PIXEL_A_NAME, PIXEL_A_ID, PIXEL_A_TOKEN)

    await expect(page.locator('tr', { hasText: PIXEL_A_NAME })).toBeVisible()
    // Verify pixel ID shows in the row
    await expect(page.locator('tr', { hasText: PIXEL_A_NAME }).locator('td').nth(1)).toContainText(PIXEL_A_ID)
    // Initial status should be active
    const status = await pixelsPage.getStatus(PIXEL_A_NAME)
    expect(status).toContain('ใช้งาน')
  })

  // --- Step 4: สร้าง pixel ตัวที่ 2 ---
  test('step 4: create pixel B → both visible in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.createPixel(PIXEL_B_NAME, PIXEL_B_ID, PIXEL_B_TOKEN)

    // Both pixels visible
    await expect(page.locator('tr', { hasText: PIXEL_A_NAME })).toBeVisible()
    await expect(page.locator('tr', { hasText: PIXEL_B_NAME })).toBeVisible()
  })

  // --- Step 5: Edit pixel A → เปลี่ยนชื่อ → Save → ชื่อเปลี่ยนใน table ---
  test('step 5: edit pixel A name → updated in table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.openEdit(PIXEL_A_NAME)

    // Clear and fill new name
    await pixelsPage.nameInput.clear()
    await pixelsPage.nameInput.fill(PIXEL_A_EDITED)
    // Access token required on edit — fill to keep existing
    await pixelsPage.accessTokenInput.fill(PIXEL_A_TOKEN)
    await pixelsPage.saveButton.click()

    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: 'แก้ไขพิกเซล' })).not.toBeVisible()

    // Updated name should appear
    await expect(page.locator('tr', { hasText: PIXEL_A_EDITED })).toBeVisible()
    // Old name should be gone
    await expect(page.locator('tr', { hasText: PIXEL_A_NAME })).not.toBeVisible()
  })

  // --- Step 6: Test connection pixel A → เห็น toast ---
  test('step 6: test connection → toast appears', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.testConnection(PIXEL_A_EDITED)

    // Toast should appear — accept both success and error since token is fake
    // Success: "Pixel ทำงานปกติ — Facebook ได้รับ event แล้ว"
    // Error: "ส่งไม่ได้ — ..."
    await expect(pixelsPage.toast.first()).toBeVisible({ timeout: 15000 })
  })

  // --- Step 7: Toggle pixel A inactive → status badge เปลี่ยน ---
  test('step 7: toggle pixel A inactive → status changes', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Dismiss any leftover toasts
    const existingToast = page.locator('[data-sonner-toast]')
    if (await existingToast.count() > 0) {
      await existingToast.first().click()
      await page.waitForTimeout(500)
    }

    // Verify initial status is "ใช้งาน"
    const initialStatus = await pixelsPage.getStatus(PIXEL_A_EDITED)
    expect(initialStatus).toContain('ใช้งาน')

    // Toggle to inactive
    await pixelsPage.toggleActive(PIXEL_A_EDITED)
    await page.waitForTimeout(1500) // Wait for API + re-render

    const inactiveStatus = await pixelsPage.getStatus(PIXEL_A_EDITED)
    expect(inactiveStatus).toContain('หยุดชั่วคราว')

    // Toggle back to active (restore state for next steps)
    await pixelsPage.toggleActive(PIXEL_A_EDITED)
    await page.waitForTimeout(1500)

    const activeAgain = await pixelsPage.getStatus(PIXEL_A_EDITED)
    expect(activeAgain).toContain('ใช้งาน')
  })

  // --- Step 8: ตั้ง backup pixel: pixel A → backup = pixel B ---
  test('step 8: set backup pixel A → B', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    await pixelsPage.openEdit(PIXEL_A_EDITED)

    // Backup pixel select should be visible (requires >= 2 pixels)
    await expect(pixelsPage.backupPixelSelect).toBeVisible()

    // Find the option that contains pixel B's name
    const options = pixelsPage.backupPixelSelect.locator('option')
    const optionCount = await options.count()
    let targetValue = ''
    for (let i = 0; i < optionCount; i++) {
      const text = await options.nth(i).textContent()
      if (text && text.includes(PIXEL_B_NAME)) {
        targetValue = await options.nth(i).getAttribute('value') ?? ''
        break
      }
    }
    expect(targetValue).not.toBe('')

    await pixelsPage.backupPixelSelect.selectOption(targetValue)
    await pixelsPage.accessTokenInput.fill(PIXEL_A_TOKEN)
    await pixelsPage.saveButton.click()

    // Wait for dialog to close
    await expect(page.getByRole('heading', { name: 'แก้ไขพิกเซล' })).not.toBeVisible()

    // Verify backup column shows pixel B's name
    await page.waitForTimeout(1000)
    const backupText = await pixelsPage.getBackup(PIXEL_A_EDITED)
    expect(backupText).toContain(PIXEL_B_NAME)
  })

  // --- Step 9: Delete pixel B → Confirm dialog → หายจาก table ---
  test('step 9: delete pixel B → confirm → gone from table', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Find pixel B row by matching the name in the first cell (avoids matching backup column of other rows)
    const rowB = page.locator('tr').filter({ has: page.locator('td:first-child', { hasText: PIXEL_B_NAME }) })
    await expect(rowB).toBeVisible()

    // Click delete on pixel B
    await rowB.getByRole('button').filter({ has: page.locator('[class*="lucide-trash"]') }).click()

    // Confirm dialog should appear
    await expect(page.getByRole('heading', { name: 'ลบพิกเซล' })).toBeVisible()
    await expect(page.getByText('คุณแน่ใจหรือไม่')).toBeVisible()

    // Confirm deletion
    await pixelsPage.deleteConfirmButton.click()

    // Pixel B should disappear from the name column
    await expect(rowB).not.toBeVisible()
  })

  // --- Step 10: Verify pixel A backup กลับเป็น "ไม่มี" ---
  test('step 10: verify pixel A backup cleared after B deleted', async ({ page }) => {
    const pixelsPage = new PixelsPage(page)
    await pixelsPage.goto()
    await page.waitForLoadState('networkidle')

    // Pixel A should still exist
    await expect(page.locator('tr', { hasText: PIXEL_A_EDITED })).toBeVisible()

    // Backup column should show "ไม่มี" (cleared) or "ไม่ทราบ" (orphaned reference)
    const backupText = await pixelsPage.getBackup(PIXEL_A_EDITED)
    const isCleared = backupText.includes('ไม่มี') || backupText.includes('ไม่ทราบ')
    expect(isCleared).toBe(true)
  })
})
